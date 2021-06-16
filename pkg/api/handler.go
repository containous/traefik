package api

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/config/runtime"
	"github.com/traefik/traefik/v2/pkg/config/static"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/version"
)

type apiError struct {
	Message string `json:"message"`
}

func writeError(rw http.ResponseWriter, msg string, code int) {
	data, err := json.Marshal(apiError{Message: msg})
	if err != nil {
		http.Error(rw, msg, code)
		return
	}

	http.Error(rw, string(data), code)
}

type serviceInfoRepresentation struct {
	*runtime.ServiceInfo
	ServerStatus map[string]string `json:"serverStatus,omitempty"`
}

// RunTimeRepresentation is the configuration information exposed by the API handler.
type RunTimeRepresentation struct {
	Routers        map[string]*runtime.RouterInfo        `json:"routers,omitempty"`
	Middlewares    map[string]*runtime.MiddlewareInfo    `json:"middlewares,omitempty"`
	Services       map[string]*serviceInfoRepresentation `json:"services,omitempty"`
	TCPRouters     map[string]*runtime.TCPRouterInfo     `json:"tcpRouters,omitempty"`
	TCPMiddlewares map[string]*runtime.TCPMiddlewareInfo `json:"tcpMiddlewares,omitempty"`
	TCPServices    map[string]*runtime.TCPServiceInfo    `json:"tcpServices,omitempty"`
	UDPRouters     map[string]*runtime.UDPRouterInfo     `json:"udpRouters,omitempty"`
	UDPServices    map[string]*runtime.UDPServiceInfo    `json:"udpServices,omitempty"`
}

// Handler serves the configuration and status of Traefik on API endpoints.
type Handler struct {
	dashboard       bool
	debug           bool
	staticConfig    static.Configuration
	dashboardAssets *assetfs.AssetFS

	// runtimeConfiguration is the data set used to create all the data representations exposed by the API.
	runtimeConfiguration *runtime.Configuration
}

// NewBuilder returns a http.Handler builder based on runtime.Configuration.
func NewBuilder(staticConfig static.Configuration) func(*runtime.Configuration) http.Handler {
	return func(configuration *runtime.Configuration) http.Handler {
		return New(staticConfig, configuration).createRouter()
	}
}

// New returns a Handler defined by staticConfig, and if provided, by runtimeConfig.
// It finishes populating the information provided in the runtimeConfig.
func New(staticConfig static.Configuration, runtimeConfig *runtime.Configuration) *Handler {
	rConfig := runtimeConfig
	if rConfig == nil {
		rConfig = &runtime.Configuration{}
	}

	return &Handler{
		dashboard:            staticConfig.API.Dashboard,
		dashboardAssets:      staticConfig.API.DashboardAssets,
		runtimeConfiguration: rConfig,
		staticConfig:         staticConfig,
		debug:                staticConfig.API.Debug,
	}
}

// createRouter creates API routes and router.
func (h Handler) createRouter() *mux.Router {
	router := mux.NewRouter()

	if h.debug {
		DebugHandler{}.Append(router)
	}

	router.Methods(http.MethodGet).Path("/api/rawdata").HandlerFunc(h.getRuntimeConfiguration)

	// Experimental endpoint
	router.Methods(http.MethodGet).Path("/api/overview").HandlerFunc(h.getOverview)

	router.Methods(http.MethodGet).Path("/api/entrypoints").HandlerFunc(h.getEntryPoints)
	router.Methods(http.MethodGet).Path("/api/entrypoints/{entryPointID}").HandlerFunc(h.getEntryPoint)

	router.Methods(http.MethodGet).Path("/api/http/routers").HandlerFunc(h.getRouters)
	router.Methods(http.MethodGet).Path("/api/http/routers/{routerID}").HandlerFunc(h.getRouter)
	router.Methods(http.MethodGet).Path("/api/http/services").HandlerFunc(h.getServices)
	router.Methods(http.MethodGet).Path("/api/http/services/{serviceID}").HandlerFunc(h.getService)
	router.Methods(http.MethodGet).Path("/api/http/middlewares").HandlerFunc(h.getMiddlewares)
	router.Methods(http.MethodGet).Path("/api/http/middlewares/{middlewareID}").HandlerFunc(h.getMiddleware)

	router.Methods(http.MethodGet).Path("/api/tcp/routers").HandlerFunc(h.getTCPRouters)
	router.Methods(http.MethodGet).Path("/api/tcp/routers/{routerID}").HandlerFunc(h.getTCPRouter)
	router.Methods(http.MethodGet).Path("/api/tcp/services").HandlerFunc(h.getTCPServices)
	router.Methods(http.MethodGet).Path("/api/tcp/services/{serviceID}").HandlerFunc(h.getTCPService)
	router.Methods(http.MethodGet).Path("/api/tcp/middlewares").HandlerFunc(h.getTCPMiddlewares)
	router.Methods(http.MethodGet).Path("/api/tcp/middlewares/{middlewareID}").HandlerFunc(h.getTCPMiddleware)

	router.Methods(http.MethodGet).Path("/api/udp/routers").HandlerFunc(h.getUDPRouters)
	router.Methods(http.MethodGet).Path("/api/udp/routers/{routerID}").HandlerFunc(h.getUDPRouter)
	router.Methods(http.MethodGet).Path("/api/udp/services").HandlerFunc(h.getUDPServices)
	router.Methods(http.MethodGet).Path("/api/udp/services/{serviceID}").HandlerFunc(h.getUDPService)

	version.Handler{}.Append(router)

	if h.dashboard {
		DashboardHandler{Assets: h.dashboardAssets}.Append(router)
	}

	return router
}

func (h Handler) getRuntimeConfiguration(rw http.ResponseWriter, request *http.Request) {
	siRepr := make(map[string]*serviceInfoRepresentation, len(h.runtimeConfiguration.Services))
	for k, v := range h.runtimeConfiguration.Services {
		siRepr[k] = &serviceInfoRepresentation{
			ServiceInfo:  v,
			ServerStatus: v.GetAllStatus(),
		}
	}

	result := RunTimeRepresentation{
		Routers:        h.runtimeConfiguration.Routers,
		Middlewares:    h.runtimeConfiguration.Middlewares,
		Services:       siRepr,
		TCPRouters:     h.runtimeConfiguration.TCPRouters,
		TCPMiddlewares: h.runtimeConfiguration.TCPMiddlewares,
		TCPServices:    h.runtimeConfiguration.TCPServices,
		UDPRouters:     h.runtimeConfiguration.UDPRouters,
		UDPServices:    h.runtimeConfiguration.UDPServices,
	}

	rw.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(rw).Encode(result)
	if err != nil {
		log.FromContext(request.Context()).Error(err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

func getProviderName(id string) string {
	return strings.SplitN(id, "@", 2)[1]
}

func extractType(element interface{}) string {
	v := reflect.ValueOf(element).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		if field.Kind() == reflect.Map && field.Type().Elem() == reflect.TypeOf(dynamic.PluginConf{}) {
			if keys := field.MapKeys(); len(keys) == 1 {
				return keys[0].String()
			}
		}

		if field.Kind() == reflect.Ptr && field.Elem().Kind() == reflect.Struct {
			if !field.IsNil() {
				return v.Type().Field(i).Name
			}
		}
	}
	return ""
}
