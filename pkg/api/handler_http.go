package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/containous/traefik/v2/pkg/config/runtime"
	"github.com/containous/traefik/v2/pkg/log"
	"github.com/gorilla/mux"
)

type routerRepresentation struct {
	*runtime.RouterInfo
	Name     string `json:"name,omitempty"`
	Provider string `json:"provider,omitempty"`
}

func newRouterRepresentation(name string, rt *runtime.RouterInfo) routerRepresentation {
	return routerRepresentation{
		RouterInfo: rt,
		Name:       name,
		Provider:   getProviderName(name),
	}
}

type serviceRepresentation struct {
	*runtime.ServiceInfo
	ServerStatus map[string]string `json:"serverStatus,omitempty"`
	Name         string            `json:"name,omitempty"`
	Provider     string            `json:"provider,omitempty"`
	Type         string            `json:"type,omitempty"`
}

func newServiceRepresentation(name string, si *runtime.ServiceInfo) serviceRepresentation {
	return serviceRepresentation{
		ServiceInfo:  si,
		Name:         name,
		Provider:     getProviderName(name),
		ServerStatus: si.GetAllStatus(),
		Type:         strings.ToLower(extractType(si.Service)),
	}
}

type middlewareRepresentation struct {
	*runtime.MiddlewareInfo
	Name     string `json:"name,omitempty"`
	Provider string `json:"provider,omitempty"`
	Type     string `json:"type,omitempty"`
}

func newMiddlewareRepresentation(name string, mi *runtime.MiddlewareInfo) middlewareRepresentation {
	return middlewareRepresentation{
		MiddlewareInfo: mi,
		Name:           name,
		Provider:       getProviderName(name),
		Type:           strings.ToLower(extractType(mi.Middleware)),
	}
}

func (h Handler) getRouters(rw http.ResponseWriter, request *http.Request) {
	results := make([]routerRepresentation, 0, len(h.runtimeConfiguration.Routers))

	criterion := newSearchCriterion(request.URL.Query())

	for name, rt := range h.runtimeConfiguration.Routers {
		if keepRouter(name, rt, criterion) {
			results = append(results, newRouterRepresentation(name, rt))
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	pageInfo, err := pagination(request, len(results))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set(nextPageHeader, strconv.Itoa(pageInfo.nextPage))

	err = json.NewEncoder(rw).Encode(results[pageInfo.startIndex:pageInfo.endIndex])
	if err != nil {
		log.FromContext(request.Context()).Error(err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

func (h Handler) getRouter(rw http.ResponseWriter, request *http.Request) {
	routerID := mux.Vars(request)["routerID"]

	router, ok := h.runtimeConfiguration.Routers[routerID]
	if !ok {
		http.NotFound(rw, request)
		return
	}

	result := newRouterRepresentation(routerID, router)

	rw.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(rw).Encode(result)
	if err != nil {
		log.FromContext(request.Context()).Error(err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

func (h Handler) getServices(rw http.ResponseWriter, request *http.Request) {
	results := make([]serviceRepresentation, 0, len(h.runtimeConfiguration.Services))

	criterion := newSearchCriterion(request.URL.Query())

	for name, si := range h.runtimeConfiguration.Services {
		if keepService(name, si, criterion) {
			results = append(results, newServiceRepresentation(name, si))
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	pageInfo, err := pagination(request, len(results))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set(nextPageHeader, strconv.Itoa(pageInfo.nextPage))

	err = json.NewEncoder(rw).Encode(results[pageInfo.startIndex:pageInfo.endIndex])
	if err != nil {
		log.FromContext(request.Context()).Error(err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

func (h Handler) getService(rw http.ResponseWriter, request *http.Request) {
	serviceID := mux.Vars(request)["serviceID"]

	service, ok := h.runtimeConfiguration.Services[serviceID]
	if !ok {
		http.NotFound(rw, request)
		return
	}

	result := newServiceRepresentation(serviceID, service)

	rw.Header().Add("Content-Type", "application/json")

	err := json.NewEncoder(rw).Encode(result)
	if err != nil {
		log.FromContext(request.Context()).Error(err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

func (h Handler) getMiddlewares(rw http.ResponseWriter, request *http.Request) {
	results := make([]middlewareRepresentation, 0, len(h.runtimeConfiguration.Middlewares))

	criterion := newSearchCriterion(request.URL.Query())

	for name, mi := range h.runtimeConfiguration.Middlewares {
		if keepMiddleware(name, mi, criterion) {
			results = append(results, newMiddlewareRepresentation(name, mi))
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	pageInfo, err := pagination(request, len(results))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set(nextPageHeader, strconv.Itoa(pageInfo.nextPage))

	err = json.NewEncoder(rw).Encode(results[pageInfo.startIndex:pageInfo.endIndex])
	if err != nil {
		log.FromContext(request.Context()).Error(err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

func (h Handler) getMiddleware(rw http.ResponseWriter, request *http.Request) {
	middlewareID := mux.Vars(request)["middlewareID"]

	middleware, ok := h.runtimeConfiguration.Middlewares[middlewareID]
	if !ok {
		http.NotFound(rw, request)
		return
	}

	result := newMiddlewareRepresentation(middlewareID, middleware)

	rw.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(rw).Encode(result)
	if err != nil {
		log.FromContext(request.Context()).Error(err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

func keepRouter(name string, item *runtime.RouterInfo, criterion *searchCriterion) bool {
	if criterion == nil {
		return true
	}

	return criterion.withStatus(item.Status) && criterion.searchIn(item.Rule, name)
}

func keepService(name string, item *runtime.ServiceInfo, criterion *searchCriterion) bool {
	if criterion == nil {
		return true
	}

	return criterion.withStatus(item.Status) && criterion.searchIn(name)
}

func keepMiddleware(name string, item *runtime.MiddlewareInfo, criterion *searchCriterion) bool {
	if criterion == nil {
		return true
	}

	return criterion.withStatus(item.Status) && criterion.searchIn(name)
}
