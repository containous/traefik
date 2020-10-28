package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/provider"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/unrolled/render"
)

var _ provider.Provider = (*Provider)(nil)

// Provider is a provider.Provider implementation that provides a Rest API.
type Provider struct {
	Insecure          bool `description:"Activate REST Provider directly on the entryPoint named traefik." json:"insecure,omitempty" toml:"insecure,omitempty" yaml:"insecure,omitempty" export:"true"`
	configurationChan chan<- dynamic.Message
}

// SetDefaults sets the default values.
func (p *Provider) SetDefaults() {}

var templatesRenderer = render.New(render.Options{Directory: "nowhere"})

// Init the provider.
func (p *Provider) Init() error {
	return nil
}

// CreateRouter creates a router for the Rest API.
func (p *Provider) CreateRouter() *mux.Router {
	router := mux.NewRouter()
	router.Methods(http.MethodPut).Path("/api/providers/{provider}").Handler(p)
	return router
}

func (p *Provider) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	if vars["provider"] != "rest" {
		p.configurationChan <- dynamic.Message{ProviderName: "rest", ErrorLoadingConfig: true}
		http.Error(rw, "Only 'rest' provider can be updated through the REST API", http.StatusBadRequest)
		return
	}

	configuration := new(dynamic.Configuration)

	if err := json.NewDecoder(req.Body).Decode(configuration); err != nil {
		log.WithoutContext().Errorf("Error parsing configuration %+v", err)
		p.configurationChan <- dynamic.Message{ProviderName: "rest", ErrorLoadingConfig: true}
		http.Error(rw, fmt.Sprintf("%+v", err), http.StatusBadRequest)
		return
	}

	p.configurationChan <- dynamic.Message{ProviderName: "rest", Configuration: configuration}
	if err := templatesRenderer.JSON(rw, http.StatusOK, configuration); err != nil {
		log.WithoutContext().Error(err)
	}
}

// Provide allows the provider to provide configurations to traefik
// using the given configuration channel.
func (p *Provider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	p.configurationChan = configurationChan
	return nil
}
