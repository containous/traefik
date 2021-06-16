package crd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	"github.com/traefik/traefik/v2/pkg/tls"
	corev1 "k8s.io/api/core/v1"
)

func (p *Provider) loadIngressRouteTCPConfiguration(ctx context.Context, client Client, tlsConfigs map[string]*tls.CertAndStores) *dynamic.TCPConfiguration {
	conf := &dynamic.TCPConfiguration{
		Routers:     map[string]*dynamic.TCPRouter{},
		Middlewares: map[string]*dynamic.TCPMiddleware{},
		Services:    map[string]*dynamic.TCPService{},
	}

	for _, ingressRouteTCP := range client.GetIngressRouteTCPs() {
		logger := log.FromContext(log.With(ctx, log.Str("ingress", ingressRouteTCP.Name), log.Str("namespace", ingressRouteTCP.Namespace)))

		if !shouldProcessIngress(p.IngressClass, ingressRouteTCP.Annotations[annotationKubernetesIngressClass]) {
			continue
		}

		if ingressRouteTCP.Spec.TLS != nil && !ingressRouteTCP.Spec.TLS.Passthrough {
			err := getTLSTCP(ctx, ingressRouteTCP, client, tlsConfigs)
			if err != nil {
				logger.Errorf("Error configuring TLS: %v", err)
			}
		}

		ingressName := ingressRouteTCP.Name
		if len(ingressName) == 0 {
			ingressName = ingressRouteTCP.GenerateName
		}

		for _, route := range ingressRouteTCP.Spec.Routes {
			if len(route.Match) == 0 {
				logger.Errorf("Empty match rule")
				continue
			}

			key, err := makeServiceKey(route.Match, ingressName)
			if err != nil {
				logger.Error(err)
				continue
			}

			mds, err := p.makeMiddlewareTCPKeys(ctx, ingressRouteTCP.Namespace, route.Middlewares)
			if err != nil {
				logger.Errorf("Failed to create middleware keys: %v", err)
				continue
			}

			serviceName := makeID(ingressRouteTCP.Namespace, key)

			for _, service := range route.Services {
				balancerServerTCP, err := p.createLoadBalancerServerTCP(client, ingressRouteTCP.Namespace, service)
				if err != nil {
					logger.
						WithField("serviceName", service.Name).
						WithField("servicePort", service.Port).
						Errorf("Cannot create service: %v", err)
					continue
				}

				// If there is only one service defined, we skip the creation of the load balancer of services,
				// i.e. the service on top is directly a load balancer of servers.
				if len(route.Services) == 1 {
					conf.Services[serviceName] = balancerServerTCP
					break
				}

				serviceKey := fmt.Sprintf("%s-%s-%s", serviceName, service.Name, &service.Port)
				conf.Services[serviceKey] = balancerServerTCP

				srv := dynamic.TCPWRRService{Name: serviceKey}
				srv.SetDefaults()
				if service.Weight != nil {
					srv.Weight = service.Weight
				}

				if conf.Services[serviceName] == nil {
					conf.Services[serviceName] = &dynamic.TCPService{Weighted: &dynamic.TCPWeightedRoundRobin{}}
				}
				conf.Services[serviceName].Weighted.Services = append(conf.Services[serviceName].Weighted.Services, srv)
			}

			conf.Routers[serviceName] = &dynamic.TCPRouter{
				EntryPoints: ingressRouteTCP.Spec.EntryPoints,
				Middlewares: mds,
				Rule:        route.Match,
				Service:     serviceName,
			}

			if ingressRouteTCP.Spec.TLS != nil {
				conf.Routers[serviceName].TLS = &dynamic.RouterTCPTLSConfig{
					Passthrough:  ingressRouteTCP.Spec.TLS.Passthrough,
					CertResolver: ingressRouteTCP.Spec.TLS.CertResolver,
					Domains:      ingressRouteTCP.Spec.TLS.Domains,
				}

				if ingressRouteTCP.Spec.TLS.Options == nil || len(ingressRouteTCP.Spec.TLS.Options.Name) == 0 {
					continue
				}

				tlsOptionsName := ingressRouteTCP.Spec.TLS.Options.Name
				// Is a Kubernetes CRD reference (i.e. not a cross-provider reference)
				ns := ingressRouteTCP.Spec.TLS.Options.Namespace
				if !strings.Contains(tlsOptionsName, "@") {
					if len(ns) == 0 {
						ns = ingressRouteTCP.Namespace
					}
					tlsOptionsName = makeID(ns, tlsOptionsName)
				} else if len(ns) > 0 {
					logger.
						WithField("TLSoptions", ingressRouteTCP.Spec.TLS.Options.Name).
						Warnf("namespace %q is ignored in cross-provider context", ns)
				}

				conf.Routers[serviceName].TLS.Options = tlsOptionsName
			}
		}
	}

	return conf
}

func (p *Provider) makeMiddlewareTCPKeys(ctx context.Context, ingRouteTCPNamespace string, middlewares []v1alpha1.ObjectReference) ([]string, error) {
	var mds []string

	for _, mi := range middlewares {
		if strings.Contains(mi.Name, providerNamespaceSeparator) {
			if len(mi.Namespace) > 0 {
				log.FromContext(ctx).
					WithField(log.MiddlewareName, mi.Name).
					Warnf("namespace %q is ignored in cross-provider context", mi.Namespace)
			}
			mds = append(mds, mi.Name)
			continue
		}

		ns := ingRouteTCPNamespace
		if len(mi.Namespace) > 0 {
			if !isNamespaceAllowed(p.AllowCrossNamespace, ingRouteTCPNamespace, mi.Namespace) {
				return nil, fmt.Errorf("middleware %s/%s is not in the IngressRouteTCP namespace %s", mi.Namespace, mi.Name, ingRouteTCPNamespace)
			}

			ns = mi.Namespace
		}

		mds = append(mds, makeID(ns, mi.Name))
	}

	return mds, nil
}

func (p *Provider) createLoadBalancerServerTCP(client Client, parentNamespace string, service v1alpha1.ServiceTCP) (*dynamic.TCPService, error) {
	ns := parentNamespace
	if len(service.Namespace) > 0 {
		if !isNamespaceAllowed(p.AllowCrossNamespace, parentNamespace, service.Namespace) {
			return nil, fmt.Errorf("tcp service %s/%s is not in the parent resource namespace %s", service.Namespace, service.Name, parentNamespace)
		}

		ns = service.Namespace
	}

	servers, err := loadTCPServers(client, ns, service)
	if err != nil {
		return nil, err
	}

	tcpService := &dynamic.TCPService{
		LoadBalancer: &dynamic.TCPServersLoadBalancer{
			Servers: servers,
		},
	}

	if service.ProxyProtocol != nil {
		tcpService.LoadBalancer.ProxyProtocol = &dynamic.ProxyProtocol{}
		tcpService.LoadBalancer.ProxyProtocol.SetDefaults()

		if service.ProxyProtocol.Version != 0 {
			tcpService.LoadBalancer.ProxyProtocol.Version = service.ProxyProtocol.Version
		}
	}

	if service.TerminationDelay != nil {
		tcpService.LoadBalancer.TerminationDelay = service.TerminationDelay
	}

	return tcpService, nil
}

func loadTCPServers(client Client, namespace string, svc v1alpha1.ServiceTCP) ([]dynamic.TCPServer, error) {
	service, exists, err := client.GetService(namespace, svc.Name)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New("service not found")
	}

	svcPort, err := getServicePort(service, svc.Port)
	if err != nil {
		return nil, err
	}

	var servers []dynamic.TCPServer
	if service.Spec.Type == corev1.ServiceTypeExternalName {
		servers = append(servers, dynamic.TCPServer{
			Address: net.JoinHostPort(service.Spec.ExternalName, strconv.Itoa(int(svcPort.Port))),
		})
	} else {
		endpoints, endpointsExists, endpointsErr := client.GetEndpoints(namespace, svc.Name)
		if endpointsErr != nil {
			return nil, endpointsErr
		}

		if !endpointsExists {
			return nil, errors.New("endpoints not found")
		}

		if len(endpoints.Subsets) == 0 {
			return nil, errors.New("subset not found")
		}

		var port int32
		for _, subset := range endpoints.Subsets {
			for _, p := range subset.Ports {
				if svcPort.Name == p.Name {
					port = p.Port
					break
				}
			}

			if port == 0 {
				return nil, errors.New("cannot define a port")
			}

			for _, addr := range subset.Addresses {
				servers = append(servers, dynamic.TCPServer{
					Address: net.JoinHostPort(addr.IP, strconv.Itoa(int(port))),
				})
			}
		}
	}

	return servers, nil
}

func getTLSTCP(ctx context.Context, ingressRoute *v1alpha1.IngressRouteTCP, k8sClient Client, tlsConfigs map[string]*tls.CertAndStores) error {
	if ingressRoute.Spec.TLS == nil {
		return nil
	}
	if ingressRoute.Spec.TLS.SecretName == "" {
		log.FromContext(ctx).Debugf("No secret name provided")
		return nil
	}

	configKey := ingressRoute.Namespace + "/" + ingressRoute.Spec.TLS.SecretName
	if _, tlsExists := tlsConfigs[configKey]; !tlsExists {
		tlsConf, err := getTLS(k8sClient, ingressRoute.Spec.TLS.SecretName, ingressRoute.Namespace)
		if err != nil {
			return err
		}

		tlsConfigs[configKey] = tlsConf
	}

	return nil
}
