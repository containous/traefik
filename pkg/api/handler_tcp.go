package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/traefik/traefik/v2/pkg/config/runtime"
	"github.com/traefik/traefik/v2/pkg/log"
)

type tcpRouterRepresentation struct {
	*runtime.TCPRouterInfo
	Name     string `json:"name,omitempty"`
	Provider string `json:"provider,omitempty"`
}

func newTCPRouterRepresentation(name string, rt *runtime.TCPRouterInfo) tcpRouterRepresentation {
	return tcpRouterRepresentation{
		TCPRouterInfo: rt,
		Name:          name,
		Provider:      getProviderName(name),
	}
}

type tcpServiceRepresentation struct {
	*runtime.TCPServiceInfo
	Name     string `json:"name,omitempty"`
	Provider string `json:"provider,omitempty"`
	Type     string `json:"type,omitempty"`
}

func newTCPServiceRepresentation(name string, si *runtime.TCPServiceInfo) tcpServiceRepresentation {
	return tcpServiceRepresentation{
		TCPServiceInfo: si,
		Name:           name,
		Provider:       getProviderName(name),
		Type:           strings.ToLower(extractType(si.TCPService)),
	}
}

type tcpMiddlewareRepresentation struct {
	*runtime.TCPMiddlewareInfo
	Name     string `json:"name,omitempty"`
	Provider string `json:"provider,omitempty"`
	Type     string `json:"type,omitempty"`
}

func newTCPMiddlewareRepresentation(name string, mi *runtime.TCPMiddlewareInfo) tcpMiddlewareRepresentation {
	return tcpMiddlewareRepresentation{
		TCPMiddlewareInfo: mi,
		Name:              name,
		Provider:          getProviderName(name),
		Type:              strings.ToLower(extractType(mi.TCPMiddleware)),
	}
}

func (h Handler) getTCPRouters(rw http.ResponseWriter, request *http.Request) {
	results := make([]tcpRouterRepresentation, 0, len(h.runtimeConfiguration.TCPRouters))

	criterion := newSearchCriterion(request.URL.Query())

	for name, rt := range h.runtimeConfiguration.TCPRouters {
		if keepTCPRouter(name, rt, criterion) {
			results = append(results, newTCPRouterRepresentation(name, rt))
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	rw.Header().Set("Content-Type", "application/json")

	pageInfo, err := pagination(request, len(results))
	if err != nil {
		writeError(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.Header().Set(nextPageHeader, strconv.Itoa(pageInfo.nextPage))

	err = json.NewEncoder(rw).Encode(results[pageInfo.startIndex:pageInfo.endIndex])
	if err != nil {
		log.FromContext(request.Context()).Error(err)
		writeError(rw, err.Error(), http.StatusInternalServerError)
	}
}

func (h Handler) getTCPRouter(rw http.ResponseWriter, request *http.Request) {
	routerID := mux.Vars(request)["routerID"]

	rw.Header().Set("Content-Type", "application/json")

	router, ok := h.runtimeConfiguration.TCPRouters[routerID]
	if !ok {
		writeError(rw, fmt.Sprintf("router not found: %s", routerID), http.StatusNotFound)
		return
	}

	result := newTCPRouterRepresentation(routerID, router)

	err := json.NewEncoder(rw).Encode(result)
	if err != nil {
		log.FromContext(request.Context()).Error(err)
		writeError(rw, err.Error(), http.StatusInternalServerError)
	}
}

func (h Handler) getTCPServices(rw http.ResponseWriter, request *http.Request) {
	results := make([]tcpServiceRepresentation, 0, len(h.runtimeConfiguration.TCPServices))

	criterion := newSearchCriterion(request.URL.Query())

	for name, si := range h.runtimeConfiguration.TCPServices {
		if keepTCPService(name, si, criterion) {
			results = append(results, newTCPServiceRepresentation(name, si))
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	rw.Header().Set("Content-Type", "application/json")

	pageInfo, err := pagination(request, len(results))
	if err != nil {
		writeError(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.Header().Set(nextPageHeader, strconv.Itoa(pageInfo.nextPage))

	err = json.NewEncoder(rw).Encode(results[pageInfo.startIndex:pageInfo.endIndex])
	if err != nil {
		log.FromContext(request.Context()).Error(err)
		writeError(rw, err.Error(), http.StatusInternalServerError)
	}
}

func (h Handler) getTCPService(rw http.ResponseWriter, request *http.Request) {
	serviceID := mux.Vars(request)["serviceID"]

	rw.Header().Set("Content-Type", "application/json")

	service, ok := h.runtimeConfiguration.TCPServices[serviceID]
	if !ok {
		writeError(rw, fmt.Sprintf("service not found: %s", serviceID), http.StatusNotFound)
		return
	}

	result := newTCPServiceRepresentation(serviceID, service)

	err := json.NewEncoder(rw).Encode(result)
	if err != nil {
		log.FromContext(request.Context()).Error(err)
		writeError(rw, err.Error(), http.StatusInternalServerError)
	}
}

func (h Handler) getTCPMiddlewares(rw http.ResponseWriter, request *http.Request) {
	results := make([]tcpMiddlewareRepresentation, 0, len(h.runtimeConfiguration.Middlewares))

	criterion := newSearchCriterion(request.URL.Query())

	for name, mi := range h.runtimeConfiguration.TCPMiddlewares {
		if keepTCPMiddleware(name, mi, criterion) {
			results = append(results, newTCPMiddlewareRepresentation(name, mi))
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	rw.Header().Set("Content-Type", "application/json")

	pageInfo, err := pagination(request, len(results))
	if err != nil {
		writeError(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.Header().Set(nextPageHeader, strconv.Itoa(pageInfo.nextPage))

	err = json.NewEncoder(rw).Encode(results[pageInfo.startIndex:pageInfo.endIndex])
	if err != nil {
		log.FromContext(request.Context()).Error(err)
		writeError(rw, err.Error(), http.StatusInternalServerError)
	}
}

func (h Handler) getTCPMiddleware(rw http.ResponseWriter, request *http.Request) {
	middlewareID := mux.Vars(request)["middlewareID"]

	rw.Header().Set("Content-Type", "application/json")

	middleware, ok := h.runtimeConfiguration.TCPMiddlewares[middlewareID]
	if !ok {
		writeError(rw, fmt.Sprintf("middleware not found: %s", middlewareID), http.StatusNotFound)
		return
	}

	result := newTCPMiddlewareRepresentation(middlewareID, middleware)

	err := json.NewEncoder(rw).Encode(result)
	if err != nil {
		log.FromContext(request.Context()).Error(err)
		writeError(rw, err.Error(), http.StatusInternalServerError)
	}
}

func keepTCPRouter(name string, item *runtime.TCPRouterInfo, criterion *searchCriterion) bool {
	if criterion == nil {
		return true
	}

	return criterion.withStatus(item.Status) && criterion.searchIn(item.Rule, name)
}

func keepTCPService(name string, item *runtime.TCPServiceInfo, criterion *searchCriterion) bool {
	if criterion == nil {
		return true
	}

	return criterion.withStatus(item.Status) && criterion.searchIn(name)
}

func keepTCPMiddleware(name string, item *runtime.TCPMiddlewareInfo, criterion *searchCriterion) bool {
	if criterion == nil {
		return true
	}

	return criterion.withStatus(item.Status) && criterion.searchIn(name)
}
