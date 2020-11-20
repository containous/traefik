package crd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	"github.com/traefik/traefik/v2/pkg/tls"
	corev1 "k8s.io/api/core/v1"
	knativenetworkingv1alpha1 "knative.dev/networking/pkg/apis/networking/v1alpha1"
)

func (p *Provider) loadKnativeIngressRouteConfiguration(ctx context.Context, client Client, tlsConfigs map[string]*tls.CertAndStores) *dynamic.HTTPConfiguration {
	conf := &dynamic.HTTPConfiguration{
		Routers:     map[string]*dynamic.Router{},
		Middlewares: map[string]*dynamic.Middleware{},
		Services:    map[string]*dynamic.Service{},
	}

	for _, ingressRoute := range client.GetKnativeIngressRoutes() {
		ctxRt := log.With(ctx, log.Str("ingress", ingressRoute.Name), log.Str("namespace", ingressRoute.Namespace))
		logger := log.FromContext(ctxRt)

		// TODO keep the name ingressClass?
		if !shouldProcessIngress(p.IngressClass, ingressRoute.Annotations[annotationKubernetesIngressClass]) {
			continue
		}

		ingressName := ingressRoute.Name
		if len(ingressName) == 0 {
			ingressName = ingressRoute.GenerateName
		}

		// create host router
		for _, route := range ingressRoute.Spec.Rules {
			// no service
			if route.HTTP == nil {
				continue
			}

			hosts := []string{}
			for _, host := range route.Hosts {
				hosts = append(hosts, fmt.Sprintf("Host(`%v`)", host))
			}
			// create path router
			for _, pathroute := range route.HTTP.Paths {
				// AppendHeaders Middleware
				var mdsPrepare []string

				if pathroute.AppendHeaders != nil && len(pathroute.AppendHeaders) > 0 {
					middlewareID := makeID(ingressName, "AppendHeader")
					conf.Middlewares[middlewareID] = &dynamic.Middleware{
						Headers: &dynamic.Headers{
							CustomRequestHeaders: pathroute.AppendHeaders,
						},
					}
					mdsPrepare = append(mdsPrepare, middlewareID)
				}

				// TODO: Timout Middleware needed.

				Match := fmt.Sprintf("(%v)", strings.Join(hosts, " || "))
				if len(pathroute.Path) > 0 {
					Match += fmt.Sprintf(" && PathPrefix(`%v`)", pathroute.Path)
				}

				for _, service := range pathroute.Splits {
					serviceName := makeID(ingressRoute.Namespace, service.ServiceName)
					mds := make([]string, len(mdsPrepare))
					copy(mds, mdsPrepare)
					if service.AppendHeaders != nil && len(service.AppendHeaders) > 0 {
						middlewareID := makeID(serviceName, "AppendHeader")
						conf.Middlewares[middlewareID] = &dynamic.Middleware{
							Headers: &dynamic.Headers{
								CustomRequestHeaders: service.AppendHeaders,
							},
						}
						mds = append(mds, middlewareID)
					}

					// TODO: entrypoint, Priority
					conf.Routers[serviceName] = &dynamic.Router{
						Middlewares: mds,
						// Priority:    0,
						// EntryPoints: ingressRoute.Spec.EntryPoints,
						Rule:    Match,
						Service: serviceName,
					}

					balancerServerHTTP, err := createKnativeLoadBalancerServerHTTP(client, service.ServiceNamespace,
						v1alpha1.Service{LoadBalancerSpec: v1alpha1.LoadBalancerSpec{Name: service.ServiceName, Port: int32(service.ServicePort.IntValue())}})
					if err != nil {
						logger.
							WithField("serviceName", service.ServiceName).
							WithField("servicePort", service.ServicePort).
							Errorf("Cannot create service: %v", err)
						continue
					}
					// If there is only one service defined, we skip the creation of the load balancer of services,
					// i.e. the service on top is directly a load balancer of servers.
					if len(route.HTTP.Paths) == 1 && len(pathroute.Splits) == 1 {
						conf.Services[serviceName] = balancerServerHTTP
						break
					}

					serviceKey := fmt.Sprintf("%s-%s-%d", serviceName, service.ServiceName, int32(service.ServicePort.IntValue()))
					conf.Services[serviceKey] = balancerServerHTTP

					srv := dynamic.WRRService{Name: serviceKey}
					srv.SetDefaults()
					if service.Percent != 0 {
						val := service.Percent
						srv.Weight = &val
					}

					if conf.Services[serviceName] == nil {
						conf.Services[serviceName] = &dynamic.Service{Weighted: &dynamic.WeightedRoundRobin{}}
					}
					conf.Services[serviceName].Weighted.Services = append(conf.Services[serviceName].Weighted.Services, srv)
				}
			}
		}
		if ingressRoute.GetStatus() == nil ||
			!ingressRoute.GetStatus().GetCondition(knativenetworkingv1alpha1.IngressConditionNetworkConfigured).IsTrue() ||
			ingressRoute.GetGeneration() != ingressRoute.GetStatus().ObservedGeneration {
			ingressRoute.Status.MarkLoadBalancerReady(
				// public lbs
				[]knativenetworkingv1alpha1.LoadBalancerIngressStatus{{
					Domain:         p.LoadBalancerDomain,
					DomainInternal: p.LoadBalancerDomainInternal,
					IP:             p.LoadBalancerIP,
				}},
				// private lbs
				[]knativenetworkingv1alpha1.LoadBalancerIngressStatus{{
					Domain:         p.LoadBalancerDomain,
					DomainInternal: p.LoadBalancerDomainInternal,
					IP:             p.LoadBalancerIP,
				}},
			)

			ingressRoute.Status.MarkNetworkConfigured()
			ingressRoute.Status.ObservedGeneration = ingressRoute.GetGeneration()

			err := client.UpdateKnativeIngressStatus(ingressRoute)
			if err != nil {
				logger.Errorf("error %v", err)
			}
		}
	}
	return conf
}

func createKnativeLoadBalancerServerHTTP(client Client, namespace string, service v1alpha1.Service) (*dynamic.Service, error) {
	servers, err := loadKnativeServers(client, namespace, service)
	if err != nil {
		return nil, err
	}

	// TODO: support other strategies.
	lb := &dynamic.ServersLoadBalancer{}
	lb.SetDefaults()

	lb.Servers = servers

	lb.PassHostHeader = service.PassHostHeader
	if lb.PassHostHeader == nil {
		passHostHeader := true
		lb.PassHostHeader = &passHostHeader
	}
	lb.ResponseForwarding = service.ResponseForwarding

	return &dynamic.Service{
		LoadBalancer: lb,
	}, nil
}

func loadKnativeServers(client Client, namespace string, svc v1alpha1.Service) ([]dynamic.Server, error) {
	strategy := ""
	if strategy == "" {
		strategy = "RoundRobin"
	}
	if strategy != "RoundRobin" {
		return nil, fmt.Errorf("load balancing strategy %v is not supported", strategy)
	}

	serverlessservice, exists, err := client.GetServerlessService(namespace, svc.Name)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("serverless service not found %s/%s", namespace, svc.Name)
	}

	service, exists, err := client.GetService(namespace, serverlessservice.Status.ServiceName)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("service not found %s/%s", namespace, svc.Name)
	}

	var portSpec *corev1.ServicePort
	for _, p := range service.Spec.Ports {
		if svc.Port == p.Port {
			portSpec = p.DeepCopy()
			break
		}
	}

	if portSpec == nil {
		return nil, errors.New("service port not found")
	}

	var servers []dynamic.Server
	if service.Spec.ClusterIP != "" {
		if svc.Port == 80 {
			servers = append(servers, dynamic.Server{
				URL: fmt.Sprintf("%s://%s:%d", "http", service.Spec.ClusterIP, portSpec.Port),
			})
		} else if svc.Port == 443 {
			servers = append(servers, dynamic.Server{
				URL: fmt.Sprintf("%s://%s:%d", "https", service.Spec.ClusterIP, portSpec.Port),
			})
		}
	}
	return servers, nil
}

func makeID(namespace, name string) string {
	if namespace == "" {
		return name
	}

	return namespace + "-" + name
}

func shouldProcessIngress(ingressClass, ingressClassAnnotation string) bool {
	return ingressClass == ingressClassAnnotation ||
		(len(ingressClass) == 0 && ingressClassAnnotation == traefikDefaultIngressClass)
}
