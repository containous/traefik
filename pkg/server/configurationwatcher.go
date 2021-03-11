package server

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/eapache/channels"
	"github.com/sirupsen/logrus"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/provider"
	"github.com/traefik/traefik/v2/pkg/safe"
)

// ConfigurationWatcher watches configuration changes.
type ConfigurationWatcher struct {
	provider provider.Provider

	defaultEntryPoints []string

	providersThrottleDuration time.Duration

	currentConfigurations safe.Safe

	configurationChan          chan dynamic.Message
	configurationValidatedChan chan dynamic.Message
	providerConfigUpdateMap    map[string]chan dynamic.Message

	requiredProvider       string
	configurationListeners []func(dynamic.Configuration)

	routinesPool *safe.Pool
}

// NewConfigurationWatcher creates a new ConfigurationWatcher.
func NewConfigurationWatcher(
	routinesPool *safe.Pool,
	pvd provider.Provider,
	providersThrottleDuration time.Duration,
	defaultEntryPoints []string,
	requiredProvider string,
) *ConfigurationWatcher {
	watcher := &ConfigurationWatcher{
		provider:                   pvd,
		configurationChan:          make(chan dynamic.Message, 100),
		configurationValidatedChan: make(chan dynamic.Message, 100),
		providerConfigUpdateMap:    make(map[string]chan dynamic.Message),
		providersThrottleDuration:  providersThrottleDuration,
		routinesPool:               routinesPool,
		defaultEntryPoints:         defaultEntryPoints,
		requiredProvider:           requiredProvider,
	}

	currentConfigurations := make(dynamic.Configurations)
	watcher.currentConfigurations.Set(currentConfigurations)

	return watcher
}

// Start the configuration watcher.
func (c *ConfigurationWatcher) Start() {
	c.routinesPool.GoCtx(c.listenProviders)
	c.routinesPool.GoCtx(c.listenConfigurations)
	c.startProvider()
}

// Stop the configuration watcher.
func (c *ConfigurationWatcher) Stop() {
	close(c.configurationChan)
	close(c.configurationValidatedChan)
}

// AddListener adds a new listener function used when new configuration is provided.
func (c *ConfigurationWatcher) AddListener(listener func(dynamic.Configuration)) {
	if c.configurationListeners == nil {
		c.configurationListeners = make([]func(dynamic.Configuration), 0)
	}
	c.configurationListeners = append(c.configurationListeners, listener)
}

func (c *ConfigurationWatcher) startProvider() {
	logger := log.WithoutContext()

	jsonConf, err := json.Marshal(c.provider)
	if err != nil {
		logger.Debugf("Unable to marshal provider configuration %T: %v", c.provider, err)
	}

	logger.Infof("Starting provider %T %s", c.provider, jsonConf)
	currentProvider := c.provider

	safe.Go(func() {
		err := currentProvider.Provide(c.configurationChan, c.routinesPool)
		if err != nil {
			logger.Errorf("Error starting provider %T: %s", currentProvider, err)
		}
	})
}

// listenProviders receives configuration changes from the providers.
// The configuration message then gets passed along a series of check
// to finally end up in a throttler that sends it to listenConfigurations (through c. configurationValidatedChan).
func (c *ConfigurationWatcher) listenProviders(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case configMsg, ok := <-c.configurationChan:
			if !ok {
				return
			}

			if configMsg.Configuration == nil {
				log.WithoutContext().WithField(log.ProviderName, configMsg.ProviderName).
					Debug("Received nil configuration from provider, skipping.")
				return
			}

			c.preLoadConfiguration(configMsg)
		}
	}
}

func (c *ConfigurationWatcher) listenConfigurations(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case configMsg, ok := <-c.configurationValidatedChan:
			if !ok || configMsg.Configuration == nil {
				return
			}
			c.loadMessage(configMsg)
		}
	}
}

func (c *ConfigurationWatcher) loadMessage(configMsg dynamic.Message) {
	currentConfigurations := c.currentConfigurations.Get().(dynamic.Configurations)

	// Copy configurations to new map so we don't change current if LoadConfig fails
	newConfigurations := currentConfigurations.DeepCopy()
	newConfigurations[configMsg.ProviderName] = configMsg.Configuration

	c.currentConfigurations.Set(newConfigurations)

	conf := mergeConfiguration(newConfigurations, c.defaultEntryPoints)
	conf = applyModel(conf)

	// We wait for first configuration of the require provider before applying configurations.
	if _, ok := newConfigurations[c.requiredProvider]; c.requiredProvider == "" || ok {
		for _, listener := range c.configurationListeners {
			listener(conf)
		}
	}
}

func (c *ConfigurationWatcher) preLoadConfiguration(configMsg dynamic.Message) {
	logger := log.WithoutContext().WithField(log.ProviderName, configMsg.ProviderName)
	if log.GetLevel() == logrus.DebugLevel {
		copyConf := configMsg.Configuration.DeepCopy()
		if copyConf.TLS != nil {
			copyConf.TLS.Certificates = nil

			for k := range copyConf.TLS.Stores {
				st := copyConf.TLS.Stores[k]
				st.DefaultCertificate = nil
				copyConf.TLS.Stores[k] = st
			}
		}

		jsonConf, err := json.Marshal(copyConf)
		if err != nil {
			logger.Errorf("Could not marshal dynamic configuration: %v", err)
			logger.Debugf("Configuration received from provider %s: [struct] %#v", configMsg.ProviderName, copyConf)
		} else {
			logger.Debugf("Configuration received from provider %s: %s", configMsg.ProviderName, string(jsonConf))
		}
	}

	if isEmptyConfiguration(configMsg.Configuration) {
		logger.Infof("Skipping empty Configuration for provider %s", configMsg.ProviderName)
		return
	}

	providerConfigUpdateCh, ok := c.providerConfigUpdateMap[configMsg.ProviderName]
	if !ok {
		providerConfigUpdateCh = make(chan dynamic.Message)
		c.providerConfigUpdateMap[configMsg.ProviderName] = providerConfigUpdateCh
		c.routinesPool.GoCtx(func(ctxPool context.Context) {
			c.throttleProviderConfigReload(ctxPool, c.providersThrottleDuration, c.configurationValidatedChan, providerConfigUpdateCh)
		})
	}

	providerConfigUpdateCh <- configMsg
}

// throttleProviderConfigReload throttles the configuration reload speed for a single provider.
// It will immediately publish a new configuration and then only publish the next configuration after the throttle duration.
// Note that in the case it receives N new configs in the timeframe of the throttle duration after publishing,
// it will publish the last of the newly received configurations.
func (c *ConfigurationWatcher) throttleProviderConfigReload(ctx context.Context, throttle time.Duration, publish chan<- dynamic.Message, in <-chan dynamic.Message) {
	ring := channels.NewRingChannel(1)
	defer ring.Close()

	c.routinesPool.GoCtx(func(ctxPool context.Context) {
		for {
			select {
			case <-ctxPool.Done():
				return
			case nextConfig := <-ring.Out():
				if config, ok := nextConfig.(dynamic.Message); ok {
					publish <- config
					time.Sleep(throttle)
				}
			}
		}
	})

	var previousConfig dynamic.Message
	for {
		select {
		case <-ctx.Done():
			return
		case nextConfig := <-in:
			if reflect.DeepEqual(previousConfig, nextConfig) {
				logger := log.WithoutContext().WithField(log.ProviderName, nextConfig.ProviderName)
				logger.Info("Skipping same configuration")
				continue
			}
			previousConfig = *nextConfig.DeepCopy()
			ring.In() <- *nextConfig.DeepCopy()
		}
	}
}

func isEmptyConfiguration(conf *dynamic.Configuration) bool {
	if conf == nil {
		return true
	}

	if conf.TCP == nil {
		conf.TCP = &dynamic.TCPConfiguration{}
	}
	if conf.HTTP == nil {
		conf.HTTP = &dynamic.HTTPConfiguration{}
	}
	if conf.UDP == nil {
		conf.UDP = &dynamic.UDPConfiguration{}
	}

	httpEmpty := conf.HTTP.Routers == nil && conf.HTTP.Services == nil && conf.HTTP.Middlewares == nil
	tlsEmpty := conf.TLS == nil || conf.TLS.Certificates == nil && conf.TLS.Stores == nil && conf.TLS.Options == nil
	tcpEmpty := conf.TCP.Routers == nil && conf.TCP.Services == nil
	udpEmpty := conf.UDP.Routers == nil && conf.UDP.Services == nil

	return httpEmpty && tlsEmpty && tcpEmpty && udpEmpty
}
