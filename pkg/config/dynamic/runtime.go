package dynamic

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/containous/traefik/pkg/log"
)

const (
	RuntimeStatusEnabled  = "enabled"
	RuntimeStatusDisabled = "disabled"
	RuntimeStatusWarning  = "warning"
)

// RuntimeConfiguration holds the information about the currently running traefik instance.
type RuntimeConfiguration struct {
	Routers     map[string]*RouterInfo     `json:"routers,omitempty"`
	Middlewares map[string]*MiddlewareInfo `json:"middlewares,omitempty"`
	Services    map[string]*ServiceInfo    `json:"services,omitempty"`
	TCPRouters  map[string]*TCPRouterInfo  `json:"tcpRouters,omitempty"`
	TCPServices map[string]*TCPServiceInfo `json:"tcpServices,omitempty"`
}

// NewRuntimeConfig returns a RuntimeConfiguration initialized with the given conf. It never returns nil.
func NewRuntimeConfig(conf Configuration) *RuntimeConfiguration {
	if conf.HTTP == nil && conf.TCP == nil {
		return &RuntimeConfiguration{}
	}

	runtimeConfig := &RuntimeConfiguration{}

	if conf.HTTP != nil {
		routers := conf.HTTP.Routers
		if len(routers) > 0 {
			runtimeConfig.Routers = make(map[string]*RouterInfo, len(routers))
			for k, v := range routers {
				runtimeConfig.Routers[k] = &RouterInfo{Router: v, Status: RuntimeStatusEnabled}
			}
		}

		services := conf.HTTP.Services
		if len(services) > 0 {
			runtimeConfig.Services = make(map[string]*ServiceInfo, len(services))
			for k, v := range services {
				runtimeConfig.Services[k] = &ServiceInfo{Service: v, Status: RuntimeStatusEnabled}
			}
		}

		middlewares := conf.HTTP.Middlewares
		if len(middlewares) > 0 {
			runtimeConfig.Middlewares = make(map[string]*MiddlewareInfo, len(middlewares))
			for k, v := range middlewares {
				runtimeConfig.Middlewares[k] = &MiddlewareInfo{Middleware: v}
			}
		}
	}

	if conf.TCP != nil {
		if len(conf.TCP.Routers) > 0 {
			runtimeConfig.TCPRouters = make(map[string]*TCPRouterInfo, len(conf.TCP.Routers))
			for k, v := range conf.TCP.Routers {
				runtimeConfig.TCPRouters[k] = &TCPRouterInfo{TCPRouter: v}
			}
		}

		if len(conf.TCP.Services) > 0 {
			runtimeConfig.TCPServices = make(map[string]*TCPServiceInfo, len(conf.TCP.Services))
			for k, v := range conf.TCP.Services {
				runtimeConfig.TCPServices[k] = &TCPServiceInfo{TCPService: v}
			}
		}
	}

	return runtimeConfig
}

// PopulateUsedBy populates all the UsedBy lists of the underlying fields of r,
// based on the relations between the included services, routers, and middlewares.
func (r *RuntimeConfiguration) PopulateUsedBy() {
	if r == nil {
		return
	}

	logger := log.WithoutContext()

	for routerName, routerInfo := range r.Routers {
		// lazily initialize Status in case caller forgot to do it
		if routerInfo.Status == "" {
			routerInfo.Status = RuntimeStatusEnabled
		}

		providerName := getProviderName(routerName)
		if providerName == "" {
			logger.WithField(log.RouterName, routerName).Error("router name is not fully qualified")
			continue
		}

		for _, midName := range routerInfo.Router.Middlewares {
			fullMidName := getQualifiedName(providerName, midName)
			if _, ok := r.Middlewares[fullMidName]; !ok {
				continue
			}
			r.Middlewares[fullMidName].UsedBy = append(r.Middlewares[fullMidName].UsedBy, routerName)
		}

		serviceName := getQualifiedName(providerName, routerInfo.Router.Service)
		if _, ok := r.Services[serviceName]; !ok {
			continue
		}
		r.Services[serviceName].UsedBy = append(r.Services[serviceName].UsedBy, routerName)
	}

	for k, serviceInfo := range r.Services {
		// lazily initialize Status in case caller forgot to do it
		if serviceInfo.Status == "" {
			serviceInfo.Status = RuntimeStatusEnabled
		}

		sort.Strings(r.Services[k].UsedBy)
	}

	for k := range r.Middlewares {
		sort.Strings(r.Middlewares[k].UsedBy)
	}

	for routerName, routerInfo := range r.TCPRouters {
		providerName := getProviderName(routerName)
		if providerName == "" {
			logger.WithField(log.RouterName, routerName).Error("tcp router name is not fully qualified")
			continue
		}

		serviceName := getQualifiedName(providerName, routerInfo.TCPRouter.Service)
		if _, ok := r.TCPServices[serviceName]; !ok {
			continue
		}
		r.TCPServices[serviceName].UsedBy = append(r.TCPServices[serviceName].UsedBy, routerName)
	}

	for k := range r.TCPServices {
		sort.Strings(r.TCPServices[k].UsedBy)
	}
}

func contains(entryPoints []string, entryPointName string) bool {
	for _, name := range entryPoints {
		if name == entryPointName {
			return true
		}
	}
	return false
}

// GetRoutersByEntryPoints returns all the http routers by entry points name and routers name
func (r *RuntimeConfiguration) GetRoutersByEntryPoints(ctx context.Context, entryPoints []string, tls bool) map[string]map[string]*RouterInfo {
	entryPointsRouters := make(map[string]map[string]*RouterInfo)

	for rtName, rt := range r.Routers {
		if (tls && rt.TLS == nil) || (!tls && rt.TLS != nil) {
			continue
		}

		eps := rt.EntryPoints
		if len(eps) == 0 {
			eps = entryPoints
		}
		for _, entryPointName := range eps {
			if !contains(entryPoints, entryPointName) {
				log.FromContext(log.With(ctx, log.Str(log.EntryPointName, entryPointName))).
					Errorf("entryPoint %q doesn't exist", entryPointName)
				continue
			}

			if _, ok := entryPointsRouters[entryPointName]; !ok {
				entryPointsRouters[entryPointName] = make(map[string]*RouterInfo)
			}

			entryPointsRouters[entryPointName][rtName] = rt
		}
	}

	return entryPointsRouters
}

// GetTCPRoutersByEntryPoints returns all the tcp routers by entry points name and routers name
func (r *RuntimeConfiguration) GetTCPRoutersByEntryPoints(ctx context.Context, entryPoints []string) map[string]map[string]*TCPRouterInfo {
	entryPointsRouters := make(map[string]map[string]*TCPRouterInfo)

	for rtName, rt := range r.TCPRouters {
		eps := rt.EntryPoints
		if len(eps) == 0 {
			eps = entryPoints
		}

		for _, entryPointName := range eps {
			if !contains(entryPoints, entryPointName) {
				log.FromContext(log.With(ctx, log.Str(log.EntryPointName, entryPointName))).
					Errorf("entryPoint %q doesn't exist", entryPointName)
				continue
			}

			if _, ok := entryPointsRouters[entryPointName]; !ok {
				entryPointsRouters[entryPointName] = make(map[string]*TCPRouterInfo)
			}

			entryPointsRouters[entryPointName][rtName] = rt
		}
	}

	return entryPointsRouters
}

// RouterInfo holds information about a currently running HTTP router
type RouterInfo struct {
	*Router // dynamic configuration
	// Err contains all the errors that occurred during router's creation.
	Err []string `json:"error,omitempty"`
	// Status reports whether the router is disabled, in a warning state, or all good (enabled).
	// If not in "enabled" state, the reason for it should be in the list of Err.
	// It is the caller's responsibility to set the initial status.
	Status string `json:"status,omitempty"`
}

// AddError adds err to r.Err, if it does not already exist.
// If critical is set, r is marked as disabled.
func (r *RouterInfo) AddError(err error, critical bool) {
	for _, value := range r.Err {
		if value == err.Error() {
			return
		}
	}

	r.Err = append(r.Err, err.Error())
	if critical {
		r.Status = RuntimeStatusDisabled
		return
	}

	// only set it to "warning" if not already in a worse state
	if r.Status != RuntimeStatusDisabled {
		r.Status = RuntimeStatusWarning
	}
}

// TCPRouterInfo holds information about a currently running TCP router
type TCPRouterInfo struct {
	*TCPRouter        // dynamic configuration
	Err        string `json:"error,omitempty"` // initialization error
}

// MiddlewareInfo holds information about a currently running middleware
type MiddlewareInfo struct {
	*Middleware // dynamic configuration
	// Err contains all the errors that occurred during service creation.
	Err    []string `json:"error,omitempty"`
	UsedBy []string `json:"usedBy,omitempty"` // list of routers and services using that middleware
}

// AddError adds err to s.Err, if it does not already exist.
// If critical is set, m is marked as disabled.
func (m *MiddlewareInfo) AddError(err error) {
	for _, value := range m.Err {
		if value == err.Error() {
			return
		}
	}

	m.Err = append(m.Err, err.Error())
}

// ServiceInfo holds information about a currently running service
type ServiceInfo struct {
	*Service // dynamic configuration
	// Err contains all the errors that occurred during service creation.
	Err []string `json:"error,omitempty"`
	// Status reports whether the service is disabled, in a warning state, or all good (enabled).
	// If not in "enabled" state, the reason for it should be in the list of Err.
	// It is the caller's responsibility to set the initial status.
	Status string   `json:"status,omitempty"`
	UsedBy []string `json:"usedBy,omitempty"` // list of routers using that service

	serverStatusMu sync.RWMutex
	serverStatus   map[string]string // keyed by server URL
}

// AddError adds err to s.Err, if it does not already exist.
// If critical is set, s is marked as disabled.
func (s *ServiceInfo) AddError(err error, critical bool) {
	for _, value := range s.Err {
		if value == err.Error() {
			return
		}
	}

	s.Err = append(s.Err, err.Error())
	if critical {
		s.Status = RuntimeStatusDisabled
		return
	}

	// only set it to "warning" if not already in a worse state
	if s.Status != RuntimeStatusDisabled {
		s.Status = RuntimeStatusWarning
	}
}

// UpdateServerStatus sets the status of the server in the ServiceInfo.
// It is the responsibility of the caller to check that s is not nil.
func (s *ServiceInfo) UpdateServerStatus(server string, status string) {
	s.serverStatusMu.Lock()
	defer s.serverStatusMu.Unlock()

	if s.serverStatus == nil {
		s.serverStatus = make(map[string]string)
	}
	s.serverStatus[server] = status
}

// GetAllStatus returns all the statuses of all the servers in ServiceInfo.
// It is the responsibility of the caller to check that s is not nil
func (s *ServiceInfo) GetAllStatus() map[string]string {
	s.serverStatusMu.RLock()
	defer s.serverStatusMu.RUnlock()

	if len(s.serverStatus) == 0 {
		return nil
	}

	allStatus := make(map[string]string, len(s.serverStatus))
	for k, v := range s.serverStatus {
		allStatus[k] = v
	}
	return allStatus
}

// TCPServiceInfo holds information about a currently running TCP service
type TCPServiceInfo struct {
	*TCPService          // dynamic configuration
	Err         error    `json:"error,omitempty"`  // initialization error
	UsedBy      []string `json:"usedBy,omitempty"` // list of routers using that service
}

func getProviderName(elementName string) string {
	parts := strings.Split(elementName, "@")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

func getQualifiedName(provider, elementName string) string {
	parts := strings.Split(elementName, "@")
	if len(parts) == 1 {
		return elementName + "@" + provider
	}
	return elementName
}
