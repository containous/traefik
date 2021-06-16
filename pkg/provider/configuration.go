package provider

import (
	"bytes"
	"context"
	"reflect"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/Masterminds/sprig/v3"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
)

// Merge Merges multiple configurations.
func Merge(ctx context.Context, configurations map[string]*dynamic.Configuration) *dynamic.Configuration {
	logger := log.FromContext(ctx)

	configuration := &dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Routers:     make(map[string]*dynamic.Router),
			Middlewares: make(map[string]*dynamic.Middleware),
			Services:    make(map[string]*dynamic.Service),
		},
		TCP: &dynamic.TCPConfiguration{
			Routers:     make(map[string]*dynamic.TCPRouter),
			Services:    make(map[string]*dynamic.TCPService),
			Middlewares: make(map[string]*dynamic.TCPMiddleware),
		},
		UDP: &dynamic.UDPConfiguration{
			Routers:  make(map[string]*dynamic.UDPRouter),
			Services: make(map[string]*dynamic.UDPService),
		},
	}

	servicesToDelete := map[string]struct{}{}
	services := map[string][]string{}

	routersToDelete := map[string]struct{}{}
	routers := map[string][]string{}

	servicesTCPToDelete := map[string]struct{}{}
	servicesTCP := map[string][]string{}

	routersTCPToDelete := map[string]struct{}{}
	routersTCP := map[string][]string{}

	servicesUDPToDelete := map[string]struct{}{}
	servicesUDP := map[string][]string{}

	routersUDPToDelete := map[string]struct{}{}
	routersUDP := map[string][]string{}

	middlewaresToDelete := map[string]struct{}{}
	middlewares := map[string][]string{}

	middlewaresTCPToDelete := map[string]struct{}{}
	middlewaresTCP := map[string][]string{}

	var sortedKeys []string
	for key := range configurations {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	for _, root := range sortedKeys {
		conf := configurations[root]
		for serviceName, service := range conf.HTTP.Services {
			services[serviceName] = append(services[serviceName], root)
			if !AddService(configuration.HTTP, serviceName, service) {
				servicesToDelete[serviceName] = struct{}{}
			}
		}

		for routerName, router := range conf.HTTP.Routers {
			routers[routerName] = append(routers[routerName], root)
			if !AddRouter(configuration.HTTP, routerName, router) {
				routersToDelete[routerName] = struct{}{}
			}
		}

		for serviceName, service := range conf.TCP.Services {
			servicesTCP[serviceName] = append(servicesTCP[serviceName], root)
			if !AddServiceTCP(configuration.TCP, serviceName, service) {
				servicesTCPToDelete[serviceName] = struct{}{}
			}
		}

		for routerName, router := range conf.TCP.Routers {
			routersTCP[routerName] = append(routersTCP[routerName], root)
			if !AddRouterTCP(configuration.TCP, routerName, router) {
				routersTCPToDelete[routerName] = struct{}{}
			}
		}

		for serviceName, service := range conf.UDP.Services {
			servicesUDP[serviceName] = append(servicesUDP[serviceName], root)
			if !AddServiceUDP(configuration.UDP, serviceName, service) {
				servicesUDPToDelete[serviceName] = struct{}{}
			}
		}

		for routerName, router := range conf.UDP.Routers {
			routersUDP[routerName] = append(routersUDP[routerName], root)
			if !AddRouterUDP(configuration.UDP, routerName, router) {
				routersUDPToDelete[routerName] = struct{}{}
			}
		}

		for middlewareName, middleware := range conf.HTTP.Middlewares {
			middlewares[middlewareName] = append(middlewares[middlewareName], root)
			if !AddMiddleware(configuration.HTTP, middlewareName, middleware) {
				middlewaresToDelete[middlewareName] = struct{}{}
			}
		}

		for middlewareName, middleware := range conf.TCP.Middlewares {
			middlewaresTCP[middlewareName] = append(middlewaresTCP[middlewareName], root)
			if !AddMiddlewareTCP(configuration.TCP, middlewareName, middleware) {
				middlewaresTCPToDelete[middlewareName] = struct{}{}
			}
		}
	}

	for serviceName := range servicesToDelete {
		logger.WithField(log.ServiceName, serviceName).
			Errorf("Service defined multiple times with different configurations in %v", services[serviceName])
		delete(configuration.HTTP.Services, serviceName)
	}

	for routerName := range routersToDelete {
		logger.WithField(log.RouterName, routerName).
			Errorf("Router defined multiple times with different configurations in %v", routers[routerName])
		delete(configuration.HTTP.Routers, routerName)
	}

	for serviceName := range servicesTCPToDelete {
		logger.WithField(log.ServiceName, serviceName).
			Errorf("Service TCP defined multiple times with different configurations in %v", servicesTCP[serviceName])
		delete(configuration.TCP.Services, serviceName)
	}

	for routerName := range routersTCPToDelete {
		logger.WithField(log.RouterName, routerName).
			Errorf("Router TCP defined multiple times with different configurations in %v", routersTCP[routerName])
		delete(configuration.TCP.Routers, routerName)
	}

	for serviceName := range servicesUDPToDelete {
		logger.WithField(log.ServiceName, serviceName).
			Errorf("UDP service defined multiple times with different configurations in %v", servicesUDP[serviceName])
		delete(configuration.UDP.Services, serviceName)
	}

	for routerName := range routersUDPToDelete {
		logger.WithField(log.RouterName, routerName).
			Errorf("UDP router defined multiple times with different configurations in %v", routersUDP[routerName])
		delete(configuration.UDP.Routers, routerName)
	}

	for middlewareName := range middlewaresToDelete {
		logger.WithField(log.MiddlewareName, middlewareName).
			Errorf("Middleware defined multiple times with different configurations in %v", middlewares[middlewareName])
		delete(configuration.HTTP.Middlewares, middlewareName)
	}

	for middlewareName := range middlewaresTCPToDelete {
		logger.WithField(log.MiddlewareName, middlewareName).
			Errorf("TCP Middleware defined multiple times with different configurations in %v", middlewaresTCP[middlewareName])
		delete(configuration.TCP.Middlewares, middlewareName)
	}

	return configuration
}

// AddServiceTCP Adds a service to a configurations.
func AddServiceTCP(configuration *dynamic.TCPConfiguration, serviceName string, service *dynamic.TCPService) bool {
	if _, ok := configuration.Services[serviceName]; !ok {
		configuration.Services[serviceName] = service
		return true
	}

	if !configuration.Services[serviceName].LoadBalancer.Mergeable(service.LoadBalancer) {
		return false
	}

	uniq := map[string]struct{}{}
	for _, server := range configuration.Services[serviceName].LoadBalancer.Servers {
		uniq[server.Address] = struct{}{}
	}

	for _, server := range service.LoadBalancer.Servers {
		if _, ok := uniq[server.Address]; !ok {
			configuration.Services[serviceName].LoadBalancer.Servers = append(configuration.Services[serviceName].LoadBalancer.Servers, server)
		}
	}

	return true
}

// AddRouterTCP Adds a router to a configurations.
func AddRouterTCP(configuration *dynamic.TCPConfiguration, routerName string, router *dynamic.TCPRouter) bool {
	if _, ok := configuration.Routers[routerName]; !ok {
		configuration.Routers[routerName] = router
		return true
	}

	return reflect.DeepEqual(configuration.Routers[routerName], router)
}

// AddMiddlewareTCP Adds a middleware to a configurations.
func AddMiddlewareTCP(configuration *dynamic.TCPConfiguration, middlewareName string, middleware *dynamic.TCPMiddleware) bool {
	if _, ok := configuration.Middlewares[middlewareName]; !ok {
		configuration.Middlewares[middlewareName] = middleware
		return true
	}

	return reflect.DeepEqual(configuration.Middlewares[middlewareName], middleware)
}

// AddServiceUDP adds a service to a configuration.
func AddServiceUDP(configuration *dynamic.UDPConfiguration, serviceName string, service *dynamic.UDPService) bool {
	if _, ok := configuration.Services[serviceName]; !ok {
		configuration.Services[serviceName] = service
		return true
	}

	if !configuration.Services[serviceName].LoadBalancer.Mergeable(service.LoadBalancer) {
		return false
	}

	uniq := map[string]struct{}{}
	for _, server := range configuration.Services[serviceName].LoadBalancer.Servers {
		uniq[server.Address] = struct{}{}
	}

	for _, server := range service.LoadBalancer.Servers {
		if _, ok := uniq[server.Address]; !ok {
			configuration.Services[serviceName].LoadBalancer.Servers = append(configuration.Services[serviceName].LoadBalancer.Servers, server)
		}
	}

	return true
}

// AddRouterUDP adds a router to a configuration.
func AddRouterUDP(configuration *dynamic.UDPConfiguration, routerName string, router *dynamic.UDPRouter) bool {
	if _, ok := configuration.Routers[routerName]; !ok {
		configuration.Routers[routerName] = router
		return true
	}

	return reflect.DeepEqual(configuration.Routers[routerName], router)
}

// AddService Adds a service to a configurations.
func AddService(configuration *dynamic.HTTPConfiguration, serviceName string, service *dynamic.Service) bool {
	if _, ok := configuration.Services[serviceName]; !ok {
		configuration.Services[serviceName] = service
		return true
	}

	if !configuration.Services[serviceName].LoadBalancer.Mergeable(service.LoadBalancer) {
		return false
	}

	uniq := map[string]struct{}{}
	for _, server := range configuration.Services[serviceName].LoadBalancer.Servers {
		uniq[server.URL] = struct{}{}
	}

	for _, server := range service.LoadBalancer.Servers {
		if _, ok := uniq[server.URL]; !ok {
			configuration.Services[serviceName].LoadBalancer.Servers = append(configuration.Services[serviceName].LoadBalancer.Servers, server)
		}
	}

	return true
}

// AddRouter Adds a router to a configurations.
func AddRouter(configuration *dynamic.HTTPConfiguration, routerName string, router *dynamic.Router) bool {
	if _, ok := configuration.Routers[routerName]; !ok {
		configuration.Routers[routerName] = router
		return true
	}

	return reflect.DeepEqual(configuration.Routers[routerName], router)
}

// AddMiddleware Adds a middleware to a configurations.
func AddMiddleware(configuration *dynamic.HTTPConfiguration, middlewareName string, middleware *dynamic.Middleware) bool {
	if _, ok := configuration.Middlewares[middlewareName]; !ok {
		configuration.Middlewares[middlewareName] = middleware
		return true
	}

	return reflect.DeepEqual(configuration.Middlewares[middlewareName], middleware)
}

// MakeDefaultRuleTemplate Creates the default rule template.
func MakeDefaultRuleTemplate(defaultRule string, funcMap template.FuncMap) (*template.Template, error) {
	defaultFuncMap := sprig.TxtFuncMap()
	defaultFuncMap["normalize"] = Normalize

	for k, fn := range funcMap {
		defaultFuncMap[k] = fn
	}

	return template.New("defaultRule").Funcs(defaultFuncMap).Parse(defaultRule)
}

// BuildTCPRouterConfiguration Builds a router configuration.
func BuildTCPRouterConfiguration(ctx context.Context, configuration *dynamic.TCPConfiguration) {
	for routerName, router := range configuration.Routers {
		loggerRouter := log.FromContext(ctx).WithField(log.RouterName, routerName)
		if len(router.Rule) == 0 {
			delete(configuration.Routers, routerName)
			loggerRouter.Errorf("Empty rule")
			continue
		}

		if len(router.Service) == 0 {
			if len(configuration.Services) > 1 {
				delete(configuration.Routers, routerName)
				loggerRouter.
					Error("Could not define the service name for the router: too many services")
				continue
			}

			for serviceName := range configuration.Services {
				router.Service = serviceName
			}
		}
	}
}

// BuildUDPRouterConfiguration Builds a router configuration.
func BuildUDPRouterConfiguration(ctx context.Context, configuration *dynamic.UDPConfiguration) {
	for routerName, router := range configuration.Routers {
		loggerRouter := log.FromContext(ctx).WithField(log.RouterName, routerName)
		if len(router.Service) > 0 {
			continue
		}

		if len(configuration.Services) > 1 {
			delete(configuration.Routers, routerName)
			loggerRouter.
				Error("Could not define the service name for the router: too many services")
			continue
		}

		for serviceName := range configuration.Services {
			router.Service = serviceName
			break
		}
	}
}

// BuildRouterConfiguration Builds a router configuration.
func BuildRouterConfiguration(ctx context.Context, configuration *dynamic.HTTPConfiguration, defaultRouterName string, defaultRuleTpl *template.Template, model interface{}) {
	if len(configuration.Routers) == 0 {
		if len(configuration.Services) > 1 {
			log.FromContext(ctx).Info("Could not create a router for the container: too many services")
		} else {
			configuration.Routers = make(map[string]*dynamic.Router)
			configuration.Routers[defaultRouterName] = &dynamic.Router{}
		}
	}

	for routerName, router := range configuration.Routers {
		loggerRouter := log.FromContext(ctx).WithField(log.RouterName, routerName)
		if len(router.Rule) == 0 {
			writer := &bytes.Buffer{}
			if err := defaultRuleTpl.Execute(writer, model); err != nil {
				loggerRouter.Errorf("Error while parsing default rule: %v", err)
				delete(configuration.Routers, routerName)
				continue
			}

			router.Rule = writer.String()
			if len(router.Rule) == 0 {
				loggerRouter.Error("Undefined rule")
				delete(configuration.Routers, routerName)
				continue
			}
		}

		if len(router.Service) == 0 {
			if len(configuration.Services) > 1 {
				delete(configuration.Routers, routerName)
				loggerRouter.
					Error("Could not define the service name for the router: too many services")
				continue
			}

			for serviceName := range configuration.Services {
				router.Service = serviceName
			}
		}
	}
}

// Normalize Replace all special chars with `-`.
func Normalize(name string) string {
	fargs := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}
	// get function
	return strings.Join(strings.FieldsFunc(name, fargs), "-")
}
