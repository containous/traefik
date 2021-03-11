package http

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/traefik/paerser/file"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/job"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/provider"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/traefik/traefik/v2/pkg/tls"
	"github.com/traefik/traefik/v2/pkg/types"
)

var _ provider.Provider = (*Provider)(nil)

// Provider is a provider.Provider implementation that queries an HTTP(s) endpoint for a configuration.
type Provider struct {
	Endpoint              string           `description:"Load configuration from this endpoint." json:"endpoint" toml:"endpoint" yaml:"endpoint"`
	PollInterval          ptypes.Duration  `description:"Polling interval for endpoint." json:"pollInterval,omitempty" toml:"pollInterval,omitempty" yaml:"pollInterval,omitempty" export:"true"`
	PollTimeout           ptypes.Duration  `description:"Polling timeout for endpoint." json:"pollTimeout,omitempty" toml:"pollTimeout,omitempty" yaml:"pollTimeout,omitempty" export:"true"`
	TLS                   *types.ClientTLS `description:"Enable TLS support." json:"tls,omitempty" toml:"tls,omitempty" yaml:"tls,omitempty" export:"true"`
	httpClient            *http.Client
	lastConfigurationHash uint64
}

// SetDefaults sets the default values.
func (p *Provider) SetDefaults() {
	p.PollInterval = ptypes.Duration(5 * time.Second)
	p.PollTimeout = ptypes.Duration(5 * time.Second)
}

// Init the provider.
func (p *Provider) Init() error {
	if p.Endpoint == "" {
		return fmt.Errorf("non-empty endpoint is required")
	}

	if p.PollInterval <= 0 {
		return fmt.Errorf("poll interval must be greater than 0")
	}

	p.httpClient = &http.Client{
		Timeout: time.Duration(p.PollTimeout),
	}

	if p.TLS != nil {
		tlsConfig, err := p.TLS.CreateTLSConfig(context.Background())
		if err != nil {
			return fmt.Errorf("unable to create TLS configuration: %w", err)
		}

		p.httpClient.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	return nil
}

// Provide allows the provider to provide configurations to traefik using the given configuration channel.
func (p *Provider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	pool.GoCtx(func(routineCtx context.Context) {
		ctxLog := log.With(routineCtx, log.Str(log.ProviderName, "http"))
		logger := log.FromContext(ctxLog)

		operation := func() error {
			ticker := time.NewTicker(time.Duration(p.PollInterval))
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					configData, err := p.fetchConfigurationData()
					if err != nil {
						return fmt.Errorf("cannot fetch configuration data: %w", err)
					}

					fnvHasher := fnv.New64()

					_, err = fnvHasher.Write(configData)
					if err != nil {
						return fmt.Errorf("cannot hash configuration data: %w", err)
					}

					hash := fnvHasher.Sum64()
					if hash == p.lastConfigurationHash {
						continue
					}

					p.lastConfigurationHash = hash

					configuration, err := decodeConfiguration(configData)
					if err != nil {
						return fmt.Errorf("cannot decode configuration data: %w", err)
					}

					configurationChan <- dynamic.Message{
						ProviderName:  "http",
						Configuration: configuration,
					}

				case <-routineCtx.Done():
					return nil
				}
			}
		}

		notify := func(err error, time time.Duration) {
			logger.Errorf("Provider connection error %+v, retrying in %s", err, time)
		}
		err := backoff.RetryNotify(safe.OperationWithRecover(operation), backoff.WithContext(job.NewBackOff(backoff.NewExponentialBackOff()), ctxLog), notify)
		if err != nil {
			logger.Errorf("Cannot connect to server endpoint %+v", err)
		}
	})

	return nil
}

// fetchConfigurationData fetches the configuration data from the configured endpoint.
func (p *Provider) fetchConfigurationData() ([]byte, error) {
	res, err := p.httpClient.Get(p.Endpoint)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-ok response code: %d", res.StatusCode)
	}

	return io.ReadAll(res.Body)
}

// decodeConfiguration decodes and returns the dynamic configuration from the given data.
func decodeConfiguration(data []byte) (*dynamic.Configuration, error) {
	configuration := &dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Routers:           make(map[string]*dynamic.Router),
			Middlewares:       make(map[string]*dynamic.Middleware),
			Services:          make(map[string]*dynamic.Service),
			ServersTransports: make(map[string]*dynamic.ServersTransport),
		},
		TCP: &dynamic.TCPConfiguration{
			Routers:  make(map[string]*dynamic.TCPRouter),
			Services: make(map[string]*dynamic.TCPService),
		},
		TLS: &dynamic.TLSConfiguration{
			Stores:  make(map[string]tls.Store),
			Options: make(map[string]tls.Options),
		},
		UDP: &dynamic.UDPConfiguration{
			Routers:  make(map[string]*dynamic.UDPRouter),
			Services: make(map[string]*dynamic.UDPService),
		},
	}

	err := file.DecodeContent(string(data), ".yaml", configuration)
	if err != nil {
		return nil, err
	}

	return configuration, nil
}
