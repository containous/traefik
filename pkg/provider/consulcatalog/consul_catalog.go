package consulcatalog

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/consul/api"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/job"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/provider"
	"github.com/traefik/traefik/v2/pkg/provider/constraints"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/traefik/traefik/v2/pkg/types"
)

// DefaultTemplateRule The default template for the default rule.
const DefaultTemplateRule = "Host(`{{ normalize .Name }}`)"

var _ provider.Provider = (*Provider)(nil)

type itemData struct {
	ID        string
	Node      string
	Name      string
	Address   string
	Port      string
	Status    string
	Labels    map[string]string
	Tags      []string
	ExtraConf configuration
}

// Provider holds configurations of the provider.
type Provider struct {
	Constraints       string          `description:"Constraints is an expression that Traefik matches against the container's labels to determine whether to create any route for that container." json:"constraints,omitempty" toml:"constraints,omitempty" yaml:"constraints,omitempty" export:"true"`
	Endpoint          *EndpointConfig `description:"Consul endpoint settings" json:"endpoint,omitempty" toml:"endpoint,omitempty" yaml:"endpoint,omitempty" export:"true"`
	Prefix            string          `description:"Prefix for consul service tags. Default 'traefik'" json:"prefix,omitempty" toml:"prefix,omitempty" yaml:"prefix,omitempty" export:"true"`
	RefreshInterval   ptypes.Duration `description:"Interval for check Consul API. Default 100ms" json:"refreshInterval,omitempty" toml:"refreshInterval,omitempty" yaml:"refreshInterval,omitempty" export:"true"`
	RequireConsistent bool            `description:"Forces the read to be fully consistent." json:"requireConsistent,omitempty" toml:"requireConsistent,omitempty" yaml:"requireConsistent,omitempty" export:"true"`
	Stale             bool            `description:"Use stale consistency for catalog reads." json:"stale,omitempty" toml:"stale,omitempty" yaml:"stale,omitempty" export:"true"`
	Cache             bool            `description:"Use local agent caching for catalog reads." json:"cache,omitempty" toml:"cache,omitempty" yaml:"cache,omitempty" export:"true"`
	ExposedByDefault  bool            `description:"Expose containers by default." json:"exposedByDefault,omitempty" toml:"exposedByDefault,omitempty" yaml:"exposedByDefault,omitempty" export:"true"`
	DefaultRule       string          `description:"Default rule." json:"defaultRule,omitempty" toml:"defaultRule,omitempty" yaml:"defaultRule,omitempty"`

	client         *api.Client
	defaultRuleTpl *template.Template
}

// EndpointConfig holds configurations of the endpoint.
type EndpointConfig struct {
	Address          string                  `description:"The address of the Consul server" json:"address,omitempty" toml:"address,omitempty" yaml:"address,omitempty" export:"true"`
	Scheme           string                  `description:"The URI scheme for the Consul server" json:"scheme,omitempty" toml:"scheme,omitempty" yaml:"scheme,omitempty" export:"true"`
	DataCenter       string                  `description:"Data center to use. If not provided, the default agent data center is used" json:"datacenter,omitempty" toml:"datacenter,omitempty" yaml:"datacenter,omitempty" export:"true"`
	Token            string                  `description:"Token is used to provide a per-request ACL token which overrides the agent's default token" json:"token,omitempty" toml:"token,omitempty" yaml:"token,omitempty" export:"true"`
	TLS              *types.ClientTLS        `description:"Enable TLS support." json:"tls,omitempty" toml:"tls,omitempty" yaml:"tls,omitempty" export:"true"`
	HTTPAuth         *EndpointHTTPAuthConfig `description:"Auth info to use for http access" json:"httpAuth,omitempty" toml:"httpAuth,omitempty" yaml:"httpAuth,omitempty" export:"true"`
	EndpointWaitTime ptypes.Duration         `description:"WaitTime limits how long a Watch will block. If not provided, the agent default values will be used" json:"endpointWaitTime,omitempty" toml:"endpointWaitTime,omitempty" yaml:"endpointWaitTime,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (c *EndpointConfig) SetDefaults() {
	c.Address = "127.0.0.1:8500"
}

// EndpointHTTPAuthConfig holds configurations of the authentication.
type EndpointHTTPAuthConfig struct {
	Username string `description:"Basic Auth username" json:"username,omitempty" toml:"username,omitempty" yaml:"username,omitempty" export:"true"`
	Password string `description:"Basic Auth password" json:"password,omitempty" toml:"password,omitempty" yaml:"password,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (p *Provider) SetDefaults() {
	endpoint := &EndpointConfig{}
	endpoint.SetDefaults()
	p.Endpoint = endpoint
	p.RefreshInterval = ptypes.Duration(15 * time.Second)
	p.Prefix = "traefik"
	p.ExposedByDefault = true
	p.DefaultRule = DefaultTemplateRule
}

// Init the provider.
func (p *Provider) Init() error {
	defaultRuleTpl, err := provider.MakeDefaultRuleTemplate(p.DefaultRule, nil)
	if err != nil {
		return fmt.Errorf("error while parsing default rule: %w", err)
	}

	p.defaultRuleTpl = defaultRuleTpl
	return nil
}

// Provide allows the consul catalog provider to provide configurations to traefik using the given configuration channel.
func (p *Provider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	pool.GoCtx(func(routineCtx context.Context) {
		ctxLog := log.With(routineCtx, log.Str(log.ProviderName, "consulcatalog"))
		logger := log.FromContext(ctxLog)

		operation := func() error {
			var err error

			p.client, err = createClient(p.Endpoint)
			if err != nil {
				return fmt.Errorf("error create consul client, %w", err)
			}

			ticker := time.NewTicker(time.Duration(p.RefreshInterval))

			for {
				select {
				case <-ticker.C:
					data, err := p.getConsulServicesData(routineCtx)
					if err != nil {
						logger.Errorf("error get consul catalog data, %v", err)
						return err
					}

					configuration := p.buildConfiguration(routineCtx, data)
					configurationChan <- dynamic.Message{
						ProviderName:  "consulcatalog",
						Configuration: configuration,
					}
				case <-routineCtx.Done():
					ticker.Stop()
					return nil
				}
			}
		}

		notify := func(err error, time time.Duration) {
			logger.Errorf("Provider connection error %+v, retrying in %s", err, time)
			configurationChan <- dynamic.Message{
				ProviderName:       "consulcatalog",
				ErrorLoadingConfig: true,
			}
		}

		err := backoff.RetryNotify(safe.OperationWithRecover(operation), backoff.WithContext(job.NewBackOff(backoff.NewExponentialBackOff()), ctxLog), notify)
		if err != nil {
			logger.Errorf("Cannot connect to consul catalog server %+v", err)
			configurationChan <- dynamic.Message{
				ProviderName:       "consulcatalog",
				ErrorLoadingConfig: true,
			}
		}
	})

	return nil
}

func (p *Provider) getConsulServicesData(ctx context.Context) ([]itemData, error) {
	consulServiceNames, err := p.fetchServices(ctx)
	if err != nil {
		return nil, err
	}

	var data []itemData
	for _, name := range consulServiceNames {
		consulServices, healthServices, err := p.fetchService(ctx, name)
		if err != nil {
			return nil, err
		}

		for i, consulService := range consulServices {
			address := consulService.ServiceAddress
			if address == "" {
				address = consulService.Address
			}

			item := itemData{
				ID:      consulService.ServiceID,
				Node:    consulService.Node,
				Name:    consulService.ServiceName,
				Address: address,
				Port:    strconv.Itoa(consulService.ServicePort),
				Labels:  tagsToNeutralLabels(consulService.ServiceTags, p.Prefix),
				Tags:    consulService.ServiceTags,
				Status:  healthServices[i].Checks.AggregatedStatus(),
			}

			extraConf, err := p.getConfiguration(item)
			if err != nil {
				log.FromContext(ctx).Errorf("Skip item %s: %v", item.Name, err)
				continue
			}
			item.ExtraConf = extraConf

			data = append(data, item)
		}
	}
	return data, nil
}

func (p *Provider) fetchService(ctx context.Context, name string) ([]*api.CatalogService, []*api.ServiceEntry, error) {
	var tagFilter string
	if !p.ExposedByDefault {
		tagFilter = p.Prefix + ".enable=true"
	}

	opts := &api.QueryOptions{AllowStale: p.Stale, RequireConsistent: p.RequireConsistent, UseCache: p.Cache}

	consulServices, _, err := p.client.Catalog().Service(name, tagFilter, opts)
	if err != nil {
		return nil, nil, err
	}

	healthServices, _, err := p.client.Health().Service(name, tagFilter, false, opts)
	return consulServices, healthServices, err
}

func (p *Provider) fetchServices(ctx context.Context) ([]string, error) {
	// The query option "Filter" is not supported by /catalog/services.
	// https://www.consul.io/api/catalog.html#list-services
	opts := &api.QueryOptions{AllowStale: p.Stale, RequireConsistent: p.RequireConsistent, UseCache: p.Cache}
	serviceNames, _, err := p.client.Catalog().Services(opts)
	if err != nil {
		return nil, err
	}

	// The keys are the service names, and the array values provide all known tags for a given service.
	// https://www.consul.io/api/catalog.html#list-services
	var filtered []string
	for svcName, tags := range serviceNames {
		logger := log.FromContext(log.With(ctx, log.Str("serviceName", svcName)))

		if !p.ExposedByDefault && !contains(tags, p.Prefix+".enable=true") {
			logger.Debug("Filtering disabled item")
			continue
		}

		if contains(tags, p.Prefix+".enable=false") {
			logger.Debug("Filtering disabled item")
			continue
		}

		matches, err := constraints.MatchTags(tags, p.Constraints)
		if err != nil {
			logger.Errorf("Error matching constraints expression: %v", err)
			continue
		}

		if !matches {
			logger.Debugf("Container pruned by constraint expression: %q", p.Constraints)
			continue
		}

		filtered = append(filtered, svcName)
	}

	return filtered, err
}

func contains(values []string, val string) bool {
	for _, value := range values {
		if strings.EqualFold(value, val) {
			return true
		}
	}
	return false
}

func createClient(cfg *EndpointConfig) (*api.Client, error) {
	config := api.Config{
		Address:    cfg.Address,
		Scheme:     cfg.Scheme,
		Datacenter: cfg.DataCenter,
		WaitTime:   time.Duration(cfg.EndpointWaitTime),
		Token:      cfg.Token,
	}

	if cfg.HTTPAuth != nil {
		config.HttpAuth = &api.HttpBasicAuth{
			Username: cfg.HTTPAuth.Username,
			Password: cfg.HTTPAuth.Password,
		}
	}

	if cfg.TLS != nil {
		config.TLSConfig = api.TLSConfig{
			Address:            cfg.Address,
			CAFile:             cfg.TLS.CA,
			CertFile:           cfg.TLS.Cert,
			KeyFile:            cfg.TLS.Key,
			InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
		}
	}

	return api.NewClient(&config)
}
