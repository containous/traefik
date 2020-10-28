package marathon

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"text/template"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gambol99/go-marathon"
	"github.com/sirupsen/logrus"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/job"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/provider"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/traefik/traefik/v2/pkg/types"
)

const (
	// DefaultTemplateRule The default template for the default rule.
	DefaultTemplateRule   = "Host(`{{ normalize .Name }}`)"
	traceMaxScanTokenSize = 1024 * 1024
	marathonEventIDs      = marathon.EventIDApplications |
		marathon.EventIDAddHealthCheck |
		marathon.EventIDDeploymentSuccess |
		marathon.EventIDDeploymentFailed |
		marathon.EventIDDeploymentInfo |
		marathon.EventIDDeploymentStepSuccess |
		marathon.EventIDDeploymentStepFailed
)

// TaskState denotes the Mesos state a task can have.
type TaskState string

const (
	taskStateRunning TaskState = "TASK_RUNNING"
	taskStateStaging TaskState = "TASK_STAGING"
)

var _ provider.Provider = (*Provider)(nil)

// Provider holds configuration of the provider.
type Provider struct {
	Constraints            string           `description:"Constraints is an expression that Traefik matches against the application's labels to determine whether to create any route for that application." json:"constraints,omitempty" toml:"constraints,omitempty" yaml:"constraints,omitempty" export:"true"`
	Trace                  bool             `description:"Display additional provider logs." json:"trace,omitempty" toml:"trace,omitempty" yaml:"trace,omitempty" export:"true"`
	Watch                  bool             `description:"Watch provider." json:"watch,omitempty" toml:"watch,omitempty" yaml:"watch,omitempty" export:"true"`
	Endpoint               string           `description:"Marathon server endpoint. You can also specify multiple endpoint for Marathon." json:"endpoint,omitempty" toml:"endpoint,omitempty" yaml:"endpoint,omitempty" export:"true"`
	DefaultRule            string           `description:"Default rule." json:"defaultRule,omitempty" toml:"defaultRule,omitempty" yaml:"defaultRule,omitempty"`
	ExposedByDefault       bool             `description:"Expose Marathon apps by default." json:"exposedByDefault,omitempty" toml:"exposedByDefault,omitempty" yaml:"exposedByDefault,omitempty" export:"true"`
	DCOSToken              string           `description:"DCOSToken for DCOS environment, This will override the Authorization header." json:"dcosToken,omitempty" toml:"dcosToken,omitempty" yaml:"dcosToken,omitempty" export:"true"`
	TLS                    *types.ClientTLS `description:"Enable TLS support." json:"tls,omitempty" toml:"tls,omitempty" yaml:"tls,omitempty" export:"true"`
	DialerTimeout          ptypes.Duration  `description:"Set a dialer timeout for Marathon." json:"dialerTimeout,omitempty" toml:"dialerTimeout,omitempty" yaml:"dialerTimeout,omitempty" export:"true"`
	ResponseHeaderTimeout  ptypes.Duration  `description:"Set a response header timeout for Marathon." json:"responseHeaderTimeout,omitempty" toml:"responseHeaderTimeout,omitempty" yaml:"responseHeaderTimeout,omitempty" export:"true"`
	TLSHandshakeTimeout    ptypes.Duration  `description:"Set a TLS handshake timeout for Marathon." json:"tlsHandshakeTimeout,omitempty" toml:"tlsHandshakeTimeout,omitempty" yaml:"tlsHandshakeTimeout,omitempty" export:"true"`
	KeepAlive              ptypes.Duration  `description:"Set a TCP Keep Alive time." json:"keepAlive,omitempty" toml:"keepAlive,omitempty" yaml:"keepAlive,omitempty" export:"true"`
	ForceTaskHostname      bool             `description:"Force to use the task's hostname." json:"forceTaskHostname,omitempty" toml:"forceTaskHostname,omitempty" yaml:"forceTaskHostname,omitempty" export:"true"`
	Basic                  *Basic           `description:"Enable basic authentication." json:"basic,omitempty" toml:"basic,omitempty" yaml:"basic,omitempty" export:"true"`
	RespectReadinessChecks bool             `description:"Filter out tasks with non-successful readiness checks during deployments." json:"respectReadinessChecks,omitempty" toml:"respectReadinessChecks,omitempty" yaml:"respectReadinessChecks,omitempty" export:"true"`
	readyChecker           *readinessChecker
	marathonClient         marathon.Marathon
	defaultRuleTpl         *template.Template
}

// SetDefaults sets the default values.
func (p *Provider) SetDefaults() {
	p.Watch = true
	p.Endpoint = "http://127.0.0.1:8080"
	p.ExposedByDefault = true
	p.DialerTimeout = ptypes.Duration(5 * time.Second)
	p.ResponseHeaderTimeout = ptypes.Duration(60 * time.Second)
	p.TLSHandshakeTimeout = ptypes.Duration(5 * time.Second)
	p.KeepAlive = ptypes.Duration(10 * time.Second)
	p.DefaultRule = DefaultTemplateRule
}

// Basic holds basic authentication specific configurations.
type Basic struct {
	HTTPBasicAuthUser string `description:"Basic authentication User." json:"httpBasicAuthUser,omitempty" toml:"httpBasicAuthUser,omitempty" yaml:"httpBasicAuthUser,omitempty"`
	HTTPBasicPassword string `description:"Basic authentication Password." json:"httpBasicPassword,omitempty" toml:"httpBasicPassword,omitempty" yaml:"httpBasicPassword,omitempty"`
}

// Init the provider.
func (p *Provider) Init() error {
	fm := template.FuncMap{
		"strsToItfs": func(values []string) []interface{} {
			var r []interface{}
			for _, v := range values {
				r = append(r, v)
			}
			return r
		},
	}

	defaultRuleTpl, err := provider.MakeDefaultRuleTemplate(p.DefaultRule, fm)
	if err != nil {
		return fmt.Errorf("error while parsing default rule: %w", err)
	}

	p.defaultRuleTpl = defaultRuleTpl
	return nil
}

// Provide allows the marathon provider to provide configurations to traefik
// using the given configuration channel.
func (p *Provider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	ctx := log.With(context.Background(), log.Str(log.ProviderName, "marathon"))
	logger := log.FromContext(ctx)

	operation := func() error {
		confg := marathon.NewDefaultConfig()
		confg.URL = p.Endpoint
		confg.EventsTransport = marathon.EventsTransportSSE
		if p.Trace {
			confg.LogOutput = log.CustomWriterLevel(logrus.DebugLevel, traceMaxScanTokenSize)
		}
		if p.Basic != nil {
			confg.HTTPBasicAuthUser = p.Basic.HTTPBasicAuthUser
			confg.HTTPBasicPassword = p.Basic.HTTPBasicPassword
		}
		var rc *readinessChecker
		if p.RespectReadinessChecks {
			logger.Debug("Enabling Marathon readiness checker")
			rc = defaultReadinessChecker(p.Trace)
		}
		p.readyChecker = rc

		if len(p.DCOSToken) > 0 {
			confg.DCOSToken = p.DCOSToken
		}
		TLSConfig, err := p.TLS.CreateTLSConfig(ctx)
		if err != nil {
			return err
		}
		confg.HTTPClient = &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					KeepAlive: time.Duration(p.KeepAlive),
					Timeout:   time.Duration(p.DialerTimeout),
				}).DialContext,
				ResponseHeaderTimeout: time.Duration(p.ResponseHeaderTimeout),
				TLSHandshakeTimeout:   time.Duration(p.TLSHandshakeTimeout),
				TLSClientConfig:       TLSConfig,
			},
		}
		client, err := marathon.NewClient(confg)
		if err != nil {
			logger.Errorf("Failed to create a client for marathon, error: %s", err)
			return err
		}
		p.marathonClient = client

		if p.Watch {
			update, err := client.AddEventsListener(marathonEventIDs)
			if err != nil {
				logger.Errorf("Failed to register for events, %s", err)
				return err
			}
			pool.GoCtx(func(ctxPool context.Context) {
				defer close(update)
				for {
					select {
					case <-ctxPool.Done():
						return
					case event := <-update:
						logger.Debugf("Received provider event %s", event)

						conf := p.getConfigurations(ctx)
						if conf != nil {
							configurationChan <- dynamic.Message{
								ProviderName:  "marathon",
								Configuration: conf,
							}
						}
					}
				}
			})
		}

		configuration := p.getConfigurations(ctx)
		configurationChan <- dynamic.Message{
			ProviderName:  "marathon",
			Configuration: configuration,
		}
		return nil
	}

	notify := func(err error, time time.Duration) {
		logger.Errorf("Provider connection error %+v, retrying in %s", err, time)
		configurationChan <- dynamic.Message{
			ProviderName:       "marathon",
			ErrorLoadingConfig: true,
		}
	}
	err := backoff.RetryNotify(safe.OperationWithRecover(operation), job.NewBackOff(backoff.NewExponentialBackOff()), notify)
	if err != nil {
		logger.Errorf("Cannot connect to Provider server: %+v", err)
		configurationChan <- dynamic.Message{
			ProviderName:       "marathon",
			ErrorLoadingConfig: true,
		}
	}
	return nil
}

func (p *Provider) getConfigurations(ctx context.Context) *dynamic.Configuration {
	applications, err := p.getApplications()
	if err != nil {
		log.FromContext(ctx).Errorf("Failed to retrieve Marathon applications: %v", err)
		return nil
	}

	return p.buildConfiguration(ctx, applications)
}

func (p *Provider) getApplications() (*marathon.Applications, error) {
	v := url.Values{}
	v.Add("embed", "apps.tasks")
	v.Add("embed", "apps.deployments")
	v.Add("embed", "apps.readiness")

	return p.marathonClient.Applications(v)
}
