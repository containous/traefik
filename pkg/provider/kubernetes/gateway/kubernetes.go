package gateway

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/go-multierror"
	"github.com/mitchellh/hashstructure"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/job"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/provider"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/traefik/traefik/v2/pkg/tls"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/gateway-api/apis/v1alpha1"
)

const (
	providerName            = "kubernetesgateway"
	traefikServiceKind      = "TraefikService"
	traefikServiceGroupName = "traefik.containo.us"
	routeHTTPKind           = "HTTPRoute"
	routeTCPKind            = "TCPRoute"
	routeTLSKind            = "TLSRoute"
)

// Provider holds configurations of the provider.
type Provider struct {
	Endpoint         string                `description:"Kubernetes server endpoint (required for external cluster client)." json:"endpoint,omitempty" toml:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Token            string                `description:"Kubernetes bearer token (not needed for in-cluster client)." json:"token,omitempty" toml:"token,omitempty" yaml:"token,omitempty"`
	CertAuthFilePath string                `description:"Kubernetes certificate authority file path (not needed for in-cluster client)." json:"certAuthFilePath,omitempty" toml:"certAuthFilePath,omitempty" yaml:"certAuthFilePath,omitempty"`
	Namespaces       []string              `description:"Kubernetes namespaces." json:"namespaces,omitempty" toml:"namespaces,omitempty" yaml:"namespaces,omitempty" export:"true"`
	LabelSelector    string                `description:"Kubernetes label selector to select specific GatewayClasses." json:"labelSelector,omitempty" toml:"labelSelector,omitempty" yaml:"labelSelector,omitempty" export:"true"`
	ThrottleDuration ptypes.Duration       `description:"Kubernetes refresh throttle duration" json:"throttleDuration,omitempty" toml:"throttleDuration,omitempty" yaml:"throttleDuration,omitempty" export:"true"`
	EntryPoints      map[string]Entrypoint `json:"-" toml:"-" yaml:"-" label:"-" file:"-"`

	lastConfiguration safe.Safe
}

// Entrypoint defines the available entry points.
type Entrypoint struct {
	Address        string
	HasHTTPTLSConf bool
}

func (p *Provider) newK8sClient(ctx context.Context) (*clientWrapper, error) {
	// Label selector validation
	_, err := labels.Parse(p.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("invalid label selector: %q", p.LabelSelector)
	}
	log.FromContext(ctx).Infof("label selector is: %q", p.LabelSelector)

	withEndpoint := ""
	if p.Endpoint != "" {
		withEndpoint = fmt.Sprintf(" with endpoint %s", p.Endpoint)
	}

	var client *clientWrapper
	switch {
	case os.Getenv("KUBERNETES_SERVICE_HOST") != "" && os.Getenv("KUBERNETES_SERVICE_PORT") != "":
		log.FromContext(ctx).Infof("Creating in-cluster Provider client%s", withEndpoint)
		client, err = newInClusterClient(p.Endpoint)
	case os.Getenv("KUBECONFIG") != "":
		log.FromContext(ctx).Infof("Creating cluster-external Provider client from KUBECONFIG %s", os.Getenv("KUBECONFIG"))
		client, err = newExternalClusterClientFromFile(os.Getenv("KUBECONFIG"))
	default:
		log.FromContext(ctx).Infof("Creating cluster-external Provider client%s", withEndpoint)
		client, err = newExternalClusterClient(p.Endpoint, p.Token, p.CertAuthFilePath)
	}

	if err != nil {
		return nil, err
	}
	client.labelSelector = p.LabelSelector

	return client, nil
}

// Init the provider.
func (p *Provider) Init() error {
	return nil
}

// Provide allows the k8s provider to provide configurations to traefik
// using the given configuration channel.
func (p *Provider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	ctxLog := log.With(context.Background(), log.Str(log.ProviderName, providerName))
	logger := log.FromContext(ctxLog)

	k8sClient, err := p.newK8sClient(ctxLog)
	if err != nil {
		return err
	}

	pool.GoCtx(func(ctxPool context.Context) {
		operation := func() error {
			eventsChan, err := k8sClient.WatchAll(p.Namespaces, ctxPool.Done())
			if err != nil {
				logger.Errorf("Error watching kubernetes events: %v", err)
				timer := time.NewTimer(1 * time.Second)
				select {
				case <-timer.C:
					return err
				case <-ctxPool.Done():
					return nil
				}
			}

			throttleDuration := time.Duration(p.ThrottleDuration)
			throttledChan := throttleEvents(ctxLog, throttleDuration, pool, eventsChan)
			if throttledChan != nil {
				eventsChan = throttledChan
			}

			for {
				select {
				case <-ctxPool.Done():
					return nil
				case event := <-eventsChan:
					// Note that event is the *first* event that came in during this throttling interval -- if we're hitting our throttle, we may have dropped events.
					// This is fine, because we don't treat different event types differently.
					// But if we do in the future, we'll need to track more information about the dropped events.
					conf := p.loadConfigurationFromGateway(ctxLog, k8sClient)

					confHash, err := hashstructure.Hash(conf, nil)
					switch {
					case err != nil:
						logger.Error("Unable to hash the configuration")
					case p.lastConfiguration.Get() == confHash:
						logger.Debugf("Skipping Kubernetes event kind %T", event)
					default:
						p.lastConfiguration.Set(confHash)
						configurationChan <- dynamic.Message{
							ProviderName:  providerName,
							Configuration: conf,
						}
					}

					// If we're throttling,
					// we sleep here for the throttle duration to enforce that we don't refresh faster than our throttle.
					// time.Sleep returns immediately if p.ThrottleDuration is 0 (no throttle).
					time.Sleep(throttleDuration)
				}
			}
		}

		notify := func(err error, time time.Duration) {
			logger.Errorf("Provider connection error: %v; retrying in %s", err, time)
		}
		err := backoff.RetryNotify(safe.OperationWithRecover(operation), backoff.WithContext(job.NewBackOff(backoff.NewExponentialBackOff()), ctxPool), notify)
		if err != nil {
			logger.Errorf("Cannot connect to Provider: %v", err)
		}
	})

	return nil
}

// TODO Handle errors and update resources statuses (gatewayClass, gateway).
func (p *Provider) loadConfigurationFromGateway(ctx context.Context, client Client) *dynamic.Configuration {
	logger := log.FromContext(ctx)

	gatewayClassNames := map[string]struct{}{}

	gatewayClasses, err := client.GetGatewayClasses()
	if err != nil {
		logger.Errorf("Cannot find GatewayClasses: %v", err)
		return &dynamic.Configuration{
			UDP: &dynamic.UDPConfiguration{
				Routers:  map[string]*dynamic.UDPRouter{},
				Services: map[string]*dynamic.UDPService{},
			},
			TCP: &dynamic.TCPConfiguration{
				Routers:  map[string]*dynamic.TCPRouter{},
				Services: map[string]*dynamic.TCPService{},
			},
			HTTP: &dynamic.HTTPConfiguration{
				Routers:     map[string]*dynamic.Router{},
				Middlewares: map[string]*dynamic.Middleware{},
				Services:    map[string]*dynamic.Service{},
			},
			TLS: &dynamic.TLSConfiguration{},
		}
	}

	for _, gatewayClass := range gatewayClasses {
		if gatewayClass.Spec.Controller == "traefik.io/gateway-controller" {
			gatewayClassNames[gatewayClass.Name] = struct{}{}

			err := client.UpdateGatewayClassStatus(gatewayClass, metav1.Condition{
				Type:               string(v1alpha1.GatewayClassConditionStatusAdmitted),
				Status:             metav1.ConditionTrue,
				Reason:             "Handled",
				Message:            "Handled by Traefik controller",
				LastTransitionTime: metav1.Now(),
			})
			if err != nil {
				logger.Errorf("Failed to update %s condition: %v", v1alpha1.GatewayClassConditionStatusAdmitted, err)
			}
		}
	}

	cfgs := map[string]*dynamic.Configuration{}

	// TODO check if we can only use the default filtering mechanism
	for _, gateway := range client.GetGateways() {
		ctxLog := log.With(ctx, log.Str("gateway", gateway.Name), log.Str("namespace", gateway.Namespace))
		logger := log.FromContext(ctxLog)

		if _, ok := gatewayClassNames[gateway.Spec.GatewayClassName]; !ok {
			continue
		}

		cfg, err := p.createGatewayConf(ctxLog, client, gateway)
		if err != nil {
			logger.Error(err)
			continue
		}

		cfgs[gateway.Name+gateway.Namespace] = cfg
	}

	conf := provider.Merge(ctx, cfgs)

	conf.TLS = &dynamic.TLSConfiguration{}

	for _, cfg := range cfgs {
		if conf.TLS == nil {
			conf.TLS = &dynamic.TLSConfiguration{}
		}

		conf.TLS.Certificates = append(conf.TLS.Certificates, cfg.TLS.Certificates...)

		for name, options := range cfg.TLS.Options {
			if conf.TLS.Options == nil {
				conf.TLS.Options = map[string]tls.Options{}
			}

			conf.TLS.Options[name] = options
		}

		for name, store := range cfg.TLS.Stores {
			if conf.TLS.Stores == nil {
				conf.TLS.Stores = map[string]tls.Store{}
			}

			conf.TLS.Stores[name] = store
		}
	}

	return conf
}

func (p *Provider) createGatewayConf(ctx context.Context, client Client, gateway *v1alpha1.Gateway) (*dynamic.Configuration, error) {
	conf := &dynamic.Configuration{
		UDP: &dynamic.UDPConfiguration{
			Routers:  map[string]*dynamic.UDPRouter{},
			Services: map[string]*dynamic.UDPService{},
		},
		TCP: &dynamic.TCPConfiguration{
			Routers:  map[string]*dynamic.TCPRouter{},
			Services: map[string]*dynamic.TCPService{},
		},
		HTTP: &dynamic.HTTPConfiguration{
			Routers:     map[string]*dynamic.Router{},
			Middlewares: map[string]*dynamic.Middleware{},
			Services:    map[string]*dynamic.Service{},
		},
		TLS: &dynamic.TLSConfiguration{},
	}

	tlsConfigs := make(map[string]*tls.CertAndStores)

	// GatewayReasonListenersNotValid is used when one or more
	// Listeners have an invalid or unsupported configuration
	// and cannot be configured on the Gateway.
	listenerStatuses := p.fillGatewayConf(ctx, client, gateway, conf, tlsConfigs)

	gatewayStatus, errG := p.makeGatewayStatus(listenerStatuses)

	err := client.UpdateGatewayStatus(gateway, gatewayStatus)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while updating gateway status: %w", err)
	}

	if errG != nil {
		return nil, fmt.Errorf("an error occurred while creating gateway status: %w", errG)
	}

	if len(tlsConfigs) > 0 {
		conf.TLS.Certificates = append(conf.TLS.Certificates, getTLSConfig(tlsConfigs)...)
	}

	return conf, nil
}

func (p *Provider) fillGatewayConf(ctx context.Context, client Client, gateway *v1alpha1.Gateway, conf *dynamic.Configuration, tlsConfigs map[string]*tls.CertAndStores) []v1alpha1.ListenerStatus {
	listenerStatuses := make([]v1alpha1.ListenerStatus, len(gateway.Spec.Listeners))
	logger := log.FromContext(ctx)
	allocatedPort := map[v1alpha1.PortNumber]v1alpha1.ProtocolType{}

	for i, listener := range gateway.Spec.Listeners {
		listenerStatuses[i] = v1alpha1.ListenerStatus{
			Port:       listener.Port,
			Conditions: []metav1.Condition{},
		}

		// Supported Protocol
		if listener.Protocol != v1alpha1.HTTPProtocolType && listener.Protocol != v1alpha1.HTTPSProtocolType &&
			listener.Protocol != v1alpha1.TCPProtocolType && listener.Protocol != v1alpha1.TLSProtocolType {
			// update "Detached" status true with "UnsupportedProtocol" reason
			listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, metav1.Condition{
				Type:               string(v1alpha1.ListenerConditionDetached),
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             string(v1alpha1.ListenerReasonUnsupportedProtocol),
				Message:            fmt.Sprintf("Unsupported listener protocol %q", listener.Protocol),
			})

			continue
		}

		// Supported Route types
		if listener.Routes.Kind != routeHTTPKind && listener.Routes.Kind != routeTCPKind && listener.Routes.Kind != routeTLSKind {
			// update "ResolvedRefs" status true with "InvalidRoutesRef" reason
			listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, metav1.Condition{
				Type:               string(v1alpha1.ListenerConditionResolvedRefs),
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             string(v1alpha1.ListenerReasonInvalidRoutesRef),
				Message:            fmt.Sprintf("Unsupported Route Kind %q", listener.Routes.Kind),
			})

			continue
		}

		// Protocol compliant with route type
		if listener.Protocol == v1alpha1.HTTPProtocolType && listener.Routes.Kind != routeHTTPKind ||
			listener.Protocol == v1alpha1.HTTPSProtocolType && listener.Routes.Kind != routeHTTPKind ||
			listener.Protocol == v1alpha1.TCPProtocolType && listener.Routes.Kind != routeTCPKind ||
			listener.Protocol == v1alpha1.TLSProtocolType && listener.Routes.Kind != routeTLSKind && listener.Routes.Kind != routeTCPKind {
			// update "Detached" status true with "UnsupportedProtocol" reason
			listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, metav1.Condition{
				Type:               string(v1alpha1.ListenerConditionDetached),
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             string(v1alpha1.ListenerReasonUnsupportedProtocol),
				Message:            fmt.Sprintf("listener protocol %q not supported with route kind %q", listener.Protocol, listener.Routes.Kind),
			})

			continue
		}

		if _, ok := allocatedPort[listener.Port]; ok {
			listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, metav1.Condition{
				Type:               string(v1alpha1.ListenerConditionDetached),
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             string(v1alpha1.ListenerReasonPortUnavailable),
				Message:            fmt.Sprintf("port %d unavailable", listener.Port),
			})

			continue
		}

		allocatedPort[listener.Port] = listener.Protocol
		ep, err := p.entryPointName(listener.Port, listener.Protocol)
		if err != nil {
			// update "Detached" status with "PortUnavailable" reason
			listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, metav1.Condition{
				Type:               string(v1alpha1.ListenerConditionDetached),
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             string(v1alpha1.ListenerReasonPortUnavailable),
				Message:            fmt.Sprintf("Cannot find entryPoint for Gateway: %v", err),
			})

			continue
		}

		// TLS
		if listener.Protocol == v1alpha1.HTTPSProtocolType || listener.Protocol == v1alpha1.TLSProtocolType {
			if listener.TLS == nil || (listener.TLS.CertificateRef == nil && listener.TLS.Mode != v1alpha1.TLSModePassthrough) {
				// update "Detached" status with "UnsupportedProtocol" reason
				listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, metav1.Condition{
					Type:               string(v1alpha1.ListenerConditionDetached),
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             string(v1alpha1.ListenerReasonUnsupportedProtocol),
					Message:            fmt.Sprintf("No TLS configuration for Gateway Listener port %d and protocol %q", listener.Port, listener.Protocol),
				})

				continue
			}

			if listener.TLS.Mode == v1alpha1.TLSModePassthrough && listener.TLS.CertificateRef != nil {
				// https://gateway-api.sigs.k8s.io/guides/tls/
				logger.Warnf("In case of Passthrough TLS mode, no TLS settings take effect as the TLS session from the client is NOT terminated at the Gateway")
			}

			isTLSPassthrough := listener.TLS.Mode == v1alpha1.TLSModePassthrough

			// Allowed configurations:
			// Protocol TLS -> Passthrough -> TLSRoute
			// Protocol TLS -> Terminate -> TCPRoute
			// Protocol HTTPS -> Terminate -> HTTPRoute
			if !(listener.Protocol == v1alpha1.TLSProtocolType && isTLSPassthrough && listener.Routes.Kind == routeTLSKind ||
				listener.Protocol == v1alpha1.TLSProtocolType && !isTLSPassthrough && listener.Routes.Kind == routeTCPKind ||
				listener.Protocol == v1alpha1.HTTPSProtocolType && !isTLSPassthrough && listener.Routes.Kind == routeHTTPKind) {
				// update "ConditionDetached" status true with "ReasonUnsupportedProtocol" reason
				listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, metav1.Condition{
					Type:               string(v1alpha1.ListenerConditionDetached),
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             string(v1alpha1.ListenerReasonUnsupportedProtocol),
					Message: fmt.Sprintf("Unsupported route kind %q with %q",
						listener.Routes.Kind, listener.TLS.Mode),
				})

				continue
			}

			if !isTLSPassthrough {
				if listener.TLS.CertificateRef.Kind != "Secret" || listener.TLS.CertificateRef.Group != "core" {
					// update "ResolvedRefs" status true with "InvalidCertificateRef" reason
					listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, metav1.Condition{
						Type:               string(v1alpha1.ListenerConditionResolvedRefs),
						Status:             metav1.ConditionFalse,
						LastTransitionTime: metav1.Now(),
						Reason:             string(v1alpha1.ListenerReasonInvalidCertificateRef),
						Message:            fmt.Sprintf("Unsupported TLS CertificateRef group/kind : %v/%v", listener.TLS.CertificateRef.Group, listener.TLS.CertificateRef.Kind),
					})

					continue
				}

				configKey := gateway.Namespace + "/" + listener.TLS.CertificateRef.Name
				if _, tlsExists := tlsConfigs[configKey]; !tlsExists {
					tlsConf, err := getTLS(client, listener.TLS.CertificateRef.Name, gateway.Namespace)
					if err != nil {
						// update "ResolvedRefs" status true with "InvalidCertificateRef" reason
						listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, metav1.Condition{
							Type:               string(v1alpha1.ListenerConditionResolvedRefs),
							Status:             metav1.ConditionFalse,
							LastTransitionTime: metav1.Now(),
							Reason:             string(v1alpha1.ListenerReasonInvalidCertificateRef),
							Message:            fmt.Sprintf("Error while retrieving certificate: %v", err),
						})

						continue
					}

					tlsConfigs[configKey] = tlsConf
				}
			}
		}

		switch listener.Routes.Kind {
		case routeHTTPKind:
			listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, gatewayHTTPRouteToHTTPConf(ctx, ep, listener, gateway, client, conf)...)
		case routeTCPKind:
			listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, gatewayTCPRouteToTCPConf(ctx, ep, listener, gateway, client, conf)...)
		case routeTLSKind:
			listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, gatewayTLSRouteToTCPConf(ctx, ep, listener, gateway, client, conf)...)
		}
	}

	return listenerStatuses
}

func gatewayHTTPRouteToHTTPConf(ctx context.Context, ep string, listener v1alpha1.Listener, gateway *v1alpha1.Gateway, client Client, conf *dynamic.Configuration) []metav1.Condition {
	// TODO: support RouteNamespaces
	selector := labels.SelectorFromSet(listener.Routes.Selector.MatchLabels)
	httpRoutes, err := client.GetHTTPRoutes(gateway.Namespace, selector)
	if err != nil {
		// update "ResolvedRefs" status true with "InvalidRoutesRef" reason
		return []metav1.Condition{{
			Type:               string(v1alpha1.ListenerConditionResolvedRefs),
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             string(v1alpha1.ListenerReasonInvalidRoutesRef),
			Message:            fmt.Sprintf("Cannot fetch %ss for namespace %q and matchLabels %v", listener.Routes.Kind, gateway.Namespace, listener.Routes.Selector.MatchLabels),
		}}
	}

	if len(httpRoutes) == 0 {
		log.FromContext(ctx).Debugf("No HTTPRoutes found for selector %q", selector)
		return nil
	}

	var conditions []metav1.Condition
	for _, httpRoute := range httpRoutes {
		hostRule, err := hostRule(httpRoute.Spec)
		if err != nil {
			conditions = append(conditions, metav1.Condition{
				Type:               string(v1alpha1.ListenerConditionResolvedRefs),
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             string(v1alpha1.ListenerReasonDegradedRoutes),
				Message:            fmt.Sprintf("Skipping HTTPRoute %s: invalid hostname: %v", httpRoute.Name, err),
			})
			continue
		}

		for _, routeRule := range httpRoute.Spec.Rules {
			rule, err := extractRule(routeRule, hostRule)
			if err != nil {
				// update "ResolvedRefs" status true with "DroppedRoutes" reason
				conditions = append(conditions, metav1.Condition{
					Type:               string(v1alpha1.ListenerConditionResolvedRefs),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(v1alpha1.ListenerReasonDegradedRoutes),
					Message:            fmt.Sprintf("Skipping %s %s: cannot generate rule: %v", listener.Routes.Kind, httpRoute.Name, err),
				})
			}

			router := dynamic.Router{
				Rule:        rule,
				EntryPoints: []string{ep},
			}

			if listener.TLS != nil {
				// TODO support let's encrypt
				router.TLS = &dynamic.RouterTLSConfig{}
			}

			// Adding the gateway name and the entryPoint name prevents overlapping of routers build from the same routes.
			routerName := httpRoute.Name + "-" + gateway.Name + "-" + ep
			routerKey, err := makeRouterKey(router.Rule, makeID(httpRoute.Namespace, routerName))
			if err != nil {
				// update "ResolvedRefs" status true with "DroppedRoutes" reason
				conditions = append(conditions, metav1.Condition{
					Type:               string(v1alpha1.ListenerConditionResolvedRefs),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(v1alpha1.ListenerReasonDegradedRoutes),
					Message:            fmt.Sprintf("Skipping %s %s: cannot make router's key with rule %s: %v", listener.Routes.Kind, httpRoute.Name, router.Rule, err),
				})

				// TODO update the RouteStatus condition / deduplicate conditions on listener
				continue
			}

			if routeRule.ForwardTo == nil {
				continue
			}

			// Traefik internal service can be used only if there is only one ForwardTo service reference.
			if len(routeRule.ForwardTo) == 1 && isInternalService(routeRule.ForwardTo[0]) {
				router.Service = routeRule.ForwardTo[0].BackendRef.Name
			} else {
				wrrService, subServices, err := loadServices(client, gateway.Namespace, routeRule.ForwardTo)
				if err != nil {
					// update "ResolvedRefs" status true with "DroppedRoutes" reason
					conditions = append(conditions, metav1.Condition{
						Type:               string(v1alpha1.ListenerConditionResolvedRefs),
						Status:             metav1.ConditionFalse,
						LastTransitionTime: metav1.Now(),
						Reason:             string(v1alpha1.ListenerReasonDegradedRoutes),
						Message:            fmt.Sprintf("Cannot load service from %s %s/%s : %v", listener.Routes.Kind, gateway.Namespace, httpRoute.Name, err),
					})

					// TODO update the RouteStatus condition / deduplicate conditions on listener
					continue
				}

				for svcName, svc := range subServices {
					conf.HTTP.Services[svcName] = svc
				}

				serviceName := provider.Normalize(routerKey + "-wrr")
				conf.HTTP.Services[serviceName] = wrrService

				router.Service = serviceName
			}

			routerKey = provider.Normalize(routerKey)
			conf.HTTP.Routers[routerKey] = &router
		}
	}

	return conditions
}

func gatewayTCPRouteToTCPConf(ctx context.Context, ep string, listener v1alpha1.Listener, gateway *v1alpha1.Gateway, client Client, conf *dynamic.Configuration) []metav1.Condition {
	// TODO: support RouteNamespaces
	selector := labels.SelectorFromSet(listener.Routes.Selector.MatchLabels)
	tcpRoutes, err := client.GetTCPRoutes(gateway.Namespace, selector)
	if err != nil {
		// update "ResolvedRefs" status true with "InvalidRoutesRef" reason
		return []metav1.Condition{{
			Type:               string(v1alpha1.ListenerConditionResolvedRefs),
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             string(v1alpha1.ListenerReasonInvalidRoutesRef),
			Message:            fmt.Sprintf("Cannot fetch %ss for namespace %q and matchLabels %v", listener.Routes.Kind, gateway.Namespace, listener.Routes.Selector.MatchLabels),
		}}
	}

	if len(tcpRoutes) == 0 {
		log.FromContext(ctx).Debugf("No TCPRoutes found for selector %q", selector)
		return nil
	}

	var conditions []metav1.Condition
	for _, tcpRoute := range tcpRoutes {
		if len(tcpRoute.Spec.Rules) > 1 {
			conditions = append(conditions, metav1.Condition{
				Type:               string(v1alpha1.ListenerConditionResolvedRefs),
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             string(v1alpha1.ListenerReasonDegradedRoutes),
				Message:            fmt.Sprintf("Skipping %s %s: multiple rules are not supported", listener.Routes.Kind, tcpRoute.Name),
			})
			continue
		}

		for _, routeRule := range tcpRoute.Spec.Rules {
			router := dynamic.TCPRouter{
				Rule:        "HostSNI(`*`)", // Gateway listener hostname not available in TCP
				EntryPoints: []string{ep},
			}

			if listener.TLS != nil {
				// TODO support let's encrypt
				router.TLS = &dynamic.RouterTCPTLSConfig{}
			}

			// Adding the gateway name and the entryPoint name prevents overlapping of routers build from the same routes.
			routerName := tcpRoute.Name + "-" + gateway.Name + "-" + ep
			routerKey, err := makeRouterKey("", makeID(tcpRoute.Namespace, routerName))
			if err != nil {
				// update "ResolvedRefs" status true with "DroppedRoutes" reason
				conditions = append(conditions, metav1.Condition{
					Type:               string(v1alpha1.ListenerConditionResolvedRefs),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(v1alpha1.ListenerReasonDegradedRoutes),
					Message:            fmt.Sprintf("Skipping %s %s: cannot make router's key with rule %s: %v", listener.Routes.Kind, tcpRoute.Name, router.Rule, err),
				})

				// TODO update the RouteStatus condition / deduplicate conditions on listener
				continue
			}

			// Should not happen due to validation
			// https://github.com/kubernetes-sigs/gateway-api/blob/af68a622f072811767d246ef5897135d93af0704/apis/v1alpha1/tcproute_types.go#L76
			if routeRule.ForwardTo == nil {
				continue
			}

			wrrService, subServices, err := loadTCPServices(client, gateway.Namespace, routeRule.ForwardTo)
			if err != nil {
				// update "ResolvedRefs" status true with "DroppedRoutes" reason
				conditions = append(conditions, metav1.Condition{
					Type:               string(v1alpha1.ListenerConditionResolvedRefs),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(v1alpha1.ListenerReasonDegradedRoutes),
					Message:            fmt.Sprintf("Cannot load service from %s %s/%s : %v", listener.Routes.Kind, gateway.Namespace, tcpRoute.Name, err),
				})

				// TODO update the RouteStatus condition / deduplicate conditions on listener
				continue
			}

			for svcName, svc := range subServices {
				conf.TCP.Services[svcName] = svc
			}

			serviceName := provider.Normalize(routerKey + "-wrr")
			conf.TCP.Services[serviceName] = wrrService

			router.Service = serviceName

			routerKey = provider.Normalize(routerKey)
			conf.TCP.Routers[routerKey] = &router
		}
	}

	return conditions
}

func gatewayTLSRouteToTCPConf(ctx context.Context, ep string, listener v1alpha1.Listener, gateway *v1alpha1.Gateway, client Client, conf *dynamic.Configuration) []metav1.Condition {
	// TODO: support RouteNamespaces
	selector := labels.SelectorFromSet(listener.Routes.Selector.MatchLabels)
	tlsRoutes, err := client.GetTLSRoutes(gateway.Namespace, selector)
	if err != nil {
		// update "ResolvedRefs" status true with "InvalidRoutesRef" reason
		return []metav1.Condition{{
			Type:               string(v1alpha1.ListenerConditionResolvedRefs),
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             string(v1alpha1.ListenerReasonInvalidRoutesRef),
			Message:            fmt.Sprintf("Cannot fetch %ss for namespace %q and matchLabels %v", listener.Routes.Kind, gateway.Namespace, listener.Routes.Selector.MatchLabels),
		}}
	}

	if len(tlsRoutes) == 0 {
		log.FromContext(ctx).Debugf("No TLSRoutes found for selector %q", selector)
		return nil
	}

	var conditions []metav1.Condition
	for _, tlsRoute := range tlsRoutes {
		for _, routeRule := range tlsRoute.Spec.Rules {
			rule, err := hostSNIRule(routeRule)
			if err != nil {
				// update "ResolvedRefs" status true with "DroppedRoutes" reason
				conditions = append(conditions, metav1.Condition{
					Type:               string(v1alpha1.ListenerConditionResolvedRefs),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(v1alpha1.ListenerReasonDegradedRoutes),
					Message:            fmt.Sprintf("Skipping %s %s: cannot make route's SNI match: %v", listener.Routes.Kind, tlsRoute.Name, err),
				})
				// TODO update the RouteStatus condition / deduplicate conditions on listener
				continue
			}

			router := dynamic.TCPRouter{
				Rule:        rule,
				EntryPoints: []string{ep},
			}

			if listener.TLS != nil {
				// TODO support let's encrypt
				router.TLS = &dynamic.RouterTCPTLSConfig{
					Passthrough: listener.TLS.Mode == v1alpha1.TLSModePassthrough,
				}
			}

			// Adding the gateway name and the entryPoint name prevents overlapping of routers build from the same routes.
			routerName := tlsRoute.Name + "-" + gateway.Name + "-" + ep
			routerKey, err := makeRouterKey(rule, makeID(tlsRoute.Namespace, routerName))
			if err != nil {
				// update "ResolvedRefs" status true with "DroppedRoutes" reason
				conditions = append(conditions, metav1.Condition{
					Type:               string(v1alpha1.ListenerConditionResolvedRefs),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(v1alpha1.ListenerReasonDegradedRoutes),
					Message:            fmt.Sprintf("Skipping %s %s: cannot make router's key with rule %s: %v", listener.Routes.Kind, tlsRoute.Name, router.Rule, err),
				})

				// TODO update the RouteStatus condition / deduplicate conditions on listener
				continue
			}

			// Should not happen due to validation
			// https://github.com/kubernetes-sigs/gateway-api/blob/af68a622f072811767d246ef5897135d93af0704/apis/v1alpha1/tlsroute_types.go#L79
			if routeRule.ForwardTo == nil {
				continue
			}

			wrrService, subServices, err := loadTCPServices(client, gateway.Namespace, routeRule.ForwardTo)
			if err != nil {
				// update "ResolvedRefs" status true with "DroppedRoutes" reason
				conditions = append(conditions, metav1.Condition{
					Type:               string(v1alpha1.ListenerConditionResolvedRefs),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(v1alpha1.ListenerReasonDegradedRoutes),
					Message:            fmt.Sprintf("Cannot load service from %s %s/%s : %v", listener.Routes.Kind, gateway.Namespace, tlsRoute.Name, err),
				})

				// TODO update the RouteStatus condition / deduplicate conditions on listener
				continue
			}

			for svcName, svc := range subServices {
				conf.TCP.Services[svcName] = svc
			}

			serviceName := provider.Normalize(routerKey + "-wrr")
			conf.TCP.Services[serviceName] = wrrService

			router.Service = serviceName

			routerKey = provider.Normalize(routerKey)
			conf.TCP.Routers[routerKey] = &router
		}
	}

	return conditions
}

func (p *Provider) makeGatewayStatus(listenerStatuses []v1alpha1.ListenerStatus) (v1alpha1.GatewayStatus, error) {
	// As Status.Addresses are not implemented yet, we initialize an empty array to follow the API expectations.
	gatewayStatus := v1alpha1.GatewayStatus{
		Addresses: []v1alpha1.GatewayAddress{},
	}

	var result error
	for i, listener := range listenerStatuses {
		if len(listener.Conditions) == 0 {
			// GatewayConditionReady "Ready", GatewayConditionReason "ListenerReady"
			listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, metav1.Condition{
				Type:               string(v1alpha1.ListenerConditionReady),
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             "ListenerReady",
				Message:            "No error found",
			})

			continue
		}

		for _, condition := range listener.Conditions {
			result = multierror.Append(result, errors.New(condition.Message))
		}
	}

	if result != nil {
		// GatewayConditionReady "Ready", GatewayConditionReason "ListenersNotValid"
		gatewayStatus.Conditions = append(gatewayStatus.Conditions, metav1.Condition{
			Type:               string(v1alpha1.GatewayConditionReady),
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             string(v1alpha1.GatewayReasonListenersNotValid),
			Message:            "All Listeners must be valid",
		})

		return gatewayStatus, result
	}

	gatewayStatus.Listeners = listenerStatuses

	gatewayStatus.Conditions = append(gatewayStatus.Conditions,
		// update "Scheduled" status with "ResourcesAvailable" reason
		metav1.Condition{
			Type:               string(v1alpha1.GatewayConditionScheduled),
			Status:             metav1.ConditionTrue,
			Reason:             "ResourcesAvailable",
			Message:            "Resources available",
			LastTransitionTime: metav1.Now(),
		},
		// update "Ready" status with "ListenersValid" reason
		metav1.Condition{
			Type:               string(v1alpha1.GatewayConditionReady),
			Status:             metav1.ConditionTrue,
			Reason:             "ListenersValid",
			Message:            "Listeners valid",
			LastTransitionTime: metav1.Now(),
		},
	)

	return gatewayStatus, nil
}

func hostRule(httpRouteSpec v1alpha1.HTTPRouteSpec) (string, error) {
	var hostNames []string
	var hostRegexNames []string

	for _, hostname := range httpRouteSpec.Hostnames {
		host := string(hostname)
		// When unspecified, "", or *, all hostnames are matched.
		// This field can be omitted for protocols that don't require hostname based matching.
		// TODO Refactor this when building support for TLS options.
		if host == "*" || host == "" {
			return "", nil
		}

		wildcard := strings.Count(host, "*")
		if wildcard == 0 {
			hostNames = append(hostNames, host)
			continue
		}

		// https://gateway-api.sigs.k8s.io/references/spec/#networking.x-k8s.io/v1alpha1.Hostname
		if !strings.HasPrefix(host, "*.") || wildcard > 1 {
			return "", fmt.Errorf("invalid rule: %q", host)
		}

		hostRegexNames = append(hostRegexNames, strings.Replace(host, "*.", "{subdomain:[a-zA-Z0-9-]+}.", 1))
	}

	var res string
	if len(hostNames) > 0 {
		res = "Host(`" + strings.Join(hostNames, "`, `") + "`)"
	}

	if len(hostRegexNames) == 0 {
		return res, nil
	}

	hostRegexp := "HostRegexp(`" + strings.Join(hostRegexNames, "`, `") + "`)"

	if len(res) > 0 {
		return "(" + res + " || " + hostRegexp + ")", nil
	}

	return hostRegexp, nil
}

func hostSNIRule(rule v1alpha1.TLSRouteRule) (string, error) {
	uniqHostnames := map[string]struct{}{}
	var hostnames []string
	for _, match := range rule.Matches {
		for _, hostname := range match.SNIs {
			if len(hostname) == 0 {
				continue
			}

			h := string(hostname)

			// first naive validation, should be improved
			wildcardNb := strings.Count(h, "*")
			if wildcardNb != 0 && !strings.HasPrefix(h, "*.") || wildcardNb > 1 {
				return "", fmt.Errorf("invalid hostname: %q", h)
			}

			hostname := "`" + h + "`"
			if _, ok := uniqHostnames[hostname]; !ok {
				hostnames = append(hostnames, hostname)
				uniqHostnames[hostname] = struct{}{}
			}
		}
	}

	if len(hostnames) == 0 {
		return "HostSNI(`*`)", nil
	}

	return "HostSNI(" + strings.Join(hostnames, ",") + ")", nil
}

func extractRule(routeRule v1alpha1.HTTPRouteRule, hostRule string) (string, error) {
	var rule string
	var matchesRules []string

	for _, match := range routeRule.Matches {
		if len(match.Path.Type) == 0 && match.Headers == nil {
			continue
		}

		var matchRules []string
		// TODO handle other path types
		if len(match.Path.Type) > 0 {
			switch match.Path.Type {
			case v1alpha1.PathMatchExact:
				matchRules = append(matchRules, "Path(`"+match.Path.Value+"`)")
			case v1alpha1.PathMatchPrefix:
				matchRules = append(matchRules, "PathPrefix(`"+match.Path.Value+"`)")
			default:
				return "", fmt.Errorf("unsupported path match %s", match.Path.Type)
			}
		}

		// TODO handle other headers types
		if match.Headers != nil {
			switch match.Headers.Type {
			case v1alpha1.HeaderMatchExact:
				var headerRules []string

				for headerName, headerValue := range match.Headers.Values {
					headerRules = append(headerRules, "Headers(`"+headerName+"`,`"+headerValue+"`)")
				}
				// to have a consistent order
				sort.Strings(headerRules)
				matchRules = append(matchRules, headerRules...)
			default:
				return "", fmt.Errorf("unsupported header match type %s", match.Headers.Type)
			}
		}

		matchesRules = append(matchesRules, strings.Join(matchRules, " && "))
	}

	// If no matches are specified, the default is a prefix
	// path match on "/", which has the effect of matching every
	// HTTP request.
	if len(routeRule.Matches) == 0 {
		matchesRules = append(matchesRules, "PathPrefix(`/`)")
	}

	if hostRule != "" {
		if len(matchesRules) == 0 {
			return hostRule, nil
		}
		rule += hostRule + " && "
	}

	if len(matchesRules) == 1 {
		return rule + matchesRules[0], nil
	}

	if len(rule) == 0 {
		return strings.Join(matchesRules, " || "), nil
	}

	return rule + "(" + strings.Join(matchesRules, " || ") + ")", nil
}

func (p *Provider) entryPointName(port v1alpha1.PortNumber, protocol v1alpha1.ProtocolType) (string, error) {
	portStr := strconv.FormatInt(int64(port), 10)

	for name, entryPoint := range p.EntryPoints {
		if strings.HasSuffix(entryPoint.Address, ":"+portStr) {
			// if the protocol is HTTP the entryPoint must have no TLS conf
			// Not relevant for v1alpha1.TLSProtocolType && v1alpha1.TCPProtocolType
			if protocol == v1alpha1.HTTPProtocolType && entryPoint.HasHTTPTLSConf {
				continue
			}

			return name, nil
		}
	}

	return "", fmt.Errorf("no matching entryPoint for port %d and protocol %q", port, protocol)
}

func makeRouterKey(rule, name string) (string, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(rule)); err != nil {
		return "", err
	}

	key := fmt.Sprintf("%s-%.10x", name, h.Sum(nil))

	return key, nil
}

func makeID(namespace, name string) string {
	if namespace == "" {
		return name
	}

	return namespace + "-" + name
}

func getTLS(k8sClient Client, secretName, namespace string) (*tls.CertAndStores, error) {
	secret, exists, err := k8sClient.GetSecret(namespace, secretName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch secret %s/%s: %w", namespace, secretName, err)
	}
	if !exists {
		return nil, fmt.Errorf("secret %s/%s does not exist", namespace, secretName)
	}

	cert, key, err := getCertificateBlocks(secret, namespace, secretName)
	if err != nil {
		return nil, err
	}

	return &tls.CertAndStores{
		Certificate: tls.Certificate{
			CertFile: tls.FileOrContent(cert),
			KeyFile:  tls.FileOrContent(key),
		},
	}, nil
}

func getTLSConfig(tlsConfigs map[string]*tls.CertAndStores) []*tls.CertAndStores {
	var secretNames []string
	for secretName := range tlsConfigs {
		secretNames = append(secretNames, secretName)
	}
	sort.Strings(secretNames)

	var configs []*tls.CertAndStores
	for _, secretName := range secretNames {
		configs = append(configs, tlsConfigs[secretName])
	}

	return configs
}

func getCertificateBlocks(secret *corev1.Secret, namespace, secretName string) (string, string, error) {
	var missingEntries []string

	tlsCrtData, tlsCrtExists := secret.Data["tls.crt"]
	if !tlsCrtExists {
		missingEntries = append(missingEntries, "tls.crt")
	}

	tlsKeyData, tlsKeyExists := secret.Data["tls.key"]
	if !tlsKeyExists {
		missingEntries = append(missingEntries, "tls.key")
	}

	if len(missingEntries) > 0 {
		return "", "", fmt.Errorf("secret %s/%s is missing the following TLS data entries: %s",
			namespace, secretName, strings.Join(missingEntries, ", "))
	}

	cert := string(tlsCrtData)
	if cert == "" {
		missingEntries = append(missingEntries, "tls.crt")
	}

	key := string(tlsKeyData)
	if key == "" {
		missingEntries = append(missingEntries, "tls.key")
	}

	if len(missingEntries) > 0 {
		return "", "", fmt.Errorf("secret %s/%s contains the following empty TLS data entries: %s",
			namespace, secretName, strings.Join(missingEntries, ", "))
	}

	return cert, key, nil
}

// loadServices is generating a WRR service, even when there is only one target.
func loadServices(client Client, namespace string, targets []v1alpha1.HTTPRouteForwardTo) (*dynamic.Service, map[string]*dynamic.Service, error) {
	services := map[string]*dynamic.Service{}

	wrrSvc := &dynamic.Service{
		Weighted: &dynamic.WeightedRoundRobin{
			Services: []dynamic.WRRService{},
		},
	}

	for _, forwardTo := range targets {
		weight := int(forwardTo.Weight)

		if forwardTo.ServiceName == nil && forwardTo.BackendRef != nil {
			if !(forwardTo.BackendRef.Group == traefikServiceGroupName && forwardTo.BackendRef.Kind == traefikServiceKind) {
				continue
			}

			if strings.HasSuffix(forwardTo.BackendRef.Name, "@internal") {
				return nil, nil, fmt.Errorf("traefik internal service %s is not allowed in a WRR loadbalancer", forwardTo.BackendRef.Name)
			}

			wrrSvc.Weighted.Services = append(wrrSvc.Weighted.Services, dynamic.WRRService{Name: forwardTo.BackendRef.Name, Weight: &weight})
			continue
		}

		if forwardTo.ServiceName == nil {
			continue
		}

		svc := dynamic.Service{
			LoadBalancer: &dynamic.ServersLoadBalancer{
				PassHostHeader: func(v bool) *bool { return &v }(true),
			},
		}

		service, exists, err := client.GetService(namespace, *forwardTo.ServiceName)
		if err != nil {
			return nil, nil, err
		}

		if !exists {
			return nil, nil, errors.New("service not found")
		}

		if len(service.Spec.Ports) > 1 && forwardTo.Port == nil {
			// If the port is unspecified and the backend is a Service
			// object consisting of multiple port definitions, the route
			// must be dropped from the Gateway. The controller should
			// raise the "ResolvedRefs" condition on the Gateway with the
			// "DroppedRoutes" reason. The gateway status for this route
			// should be updated with a condition that describes the error
			// more specifically.
			log.WithoutContext().Errorf("A multiple ports Kubernetes Service cannot be used if unspecified forwardTo.Port")
			continue
		}

		var portSpec corev1.ServicePort
		var match bool

		for _, p := range service.Spec.Ports {
			if forwardTo.Port == nil || p.Port == int32(*forwardTo.Port) {
				portSpec = p
				match = true
				break
			}
		}

		if !match {
			return nil, nil, errors.New("service port not found")
		}

		endpoints, endpointsExists, endpointsErr := client.GetEndpoints(namespace, *forwardTo.ServiceName)
		if endpointsErr != nil {
			return nil, nil, endpointsErr
		}

		if !endpointsExists {
			return nil, nil, errors.New("endpoints not found")
		}

		if len(endpoints.Subsets) == 0 {
			return nil, nil, errors.New("subset not found")
		}

		var port int32
		var portStr string
		for _, subset := range endpoints.Subsets {
			for _, p := range subset.Ports {
				if portSpec.Name == p.Name {
					port = p.Port
					break
				}
			}

			if port == 0 {
				return nil, nil, errors.New("cannot define a port")
			}

			protocol := getProtocol(portSpec)

			portStr = strconv.FormatInt(int64(port), 10)
			for _, addr := range subset.Addresses {
				svc.LoadBalancer.Servers = append(svc.LoadBalancer.Servers, dynamic.Server{
					URL: fmt.Sprintf("%s://%s", protocol, net.JoinHostPort(addr.IP, portStr)),
				})
			}
		}

		serviceName := provider.Normalize(makeID(service.Namespace, service.Name) + "-" + portStr)
		services[serviceName] = &svc

		wrrSvc.Weighted.Services = append(wrrSvc.Weighted.Services, dynamic.WRRService{Name: serviceName, Weight: &weight})
	}

	if len(wrrSvc.Weighted.Services) == 0 {
		return nil, nil, errors.New("no service has been created")
	}

	return wrrSvc, services, nil
}

// loadTCPServices is generating a WRR service, even when there is only one target.
func loadTCPServices(client Client, namespace string, targets []v1alpha1.RouteForwardTo) (*dynamic.TCPService, map[string]*dynamic.TCPService, error) {
	services := map[string]*dynamic.TCPService{}

	wrrSvc := &dynamic.TCPService{
		Weighted: &dynamic.TCPWeightedRoundRobin{
			Services: []dynamic.TCPWRRService{},
		},
	}

	for _, forwardTo := range targets {
		weight := int(forwardTo.Weight)

		if forwardTo.ServiceName == nil && forwardTo.BackendRef != nil {
			if !(forwardTo.BackendRef.Group == traefikServiceGroupName && forwardTo.BackendRef.Kind == traefikServiceKind) {
				continue
			}

			if strings.HasSuffix(forwardTo.BackendRef.Name, "@internal") {
				return nil, nil, fmt.Errorf("traefik internal service %s is not allowed in a TCP service", forwardTo.BackendRef.Name)
			}

			wrrSvc.Weighted.Services = append(wrrSvc.Weighted.Services, dynamic.TCPWRRService{Name: forwardTo.BackendRef.Name, Weight: &weight})
			continue
		}

		if forwardTo.ServiceName == nil {
			continue
		}

		svc := dynamic.TCPService{
			LoadBalancer: &dynamic.TCPServersLoadBalancer{},
		}

		service, exists, err := client.GetService(namespace, *forwardTo.ServiceName)
		if err != nil {
			return nil, nil, err
		}

		if !exists {
			return nil, nil, errors.New("service not found")
		}

		if len(service.Spec.Ports) > 1 && forwardTo.Port == nil {
			// If the port is unspecified and the backend is a Service
			// object consisting of multiple port definitions, the route
			// must be dropped from the Gateway. The controller should
			// raise the "ResolvedRefs" condition on the Gateway with the
			// "DroppedRoutes" reason. The gateway status for this route
			// should be updated with a condition that describes the error
			// more specifically.
			log.WithoutContext().Errorf("A multiple ports Kubernetes Service cannot be used if unspecified forwardTo.Port")
			continue
		}

		var portSpec corev1.ServicePort
		var match bool

		for _, p := range service.Spec.Ports {
			if forwardTo.Port == nil || p.Port == int32(*forwardTo.Port) {
				portSpec = p
				match = true
				break
			}
		}

		if !match {
			return nil, nil, errors.New("service port not found")
		}

		endpoints, endpointsExists, endpointsErr := client.GetEndpoints(namespace, *forwardTo.ServiceName)
		if endpointsErr != nil {
			return nil, nil, endpointsErr
		}

		if !endpointsExists {
			return nil, nil, errors.New("endpoints not found")
		}

		if len(endpoints.Subsets) == 0 {
			return nil, nil, errors.New("subset not found")
		}

		var port int32
		var portStr string
		for _, subset := range endpoints.Subsets {
			for _, p := range subset.Ports {
				if portSpec.Name == p.Name {
					port = p.Port
					break
				}
			}

			if port == 0 {
				return nil, nil, errors.New("cannot define a port")
			}

			portStr = strconv.FormatInt(int64(port), 10)
			for _, addr := range subset.Addresses {
				svc.LoadBalancer.Servers = append(svc.LoadBalancer.Servers, dynamic.TCPServer{
					Address: net.JoinHostPort(addr.IP, portStr),
				})
			}
		}

		serviceName := provider.Normalize(makeID(service.Namespace, service.Name) + "-" + portStr)
		services[serviceName] = &svc

		wrrSvc.Weighted.Services = append(wrrSvc.Weighted.Services, dynamic.TCPWRRService{Name: serviceName, Weight: &weight})
	}

	if len(wrrSvc.Weighted.Services) == 0 {
		return nil, nil, errors.New("no service has been created")
	}

	return wrrSvc, services, nil
}

func getProtocol(portSpec corev1.ServicePort) string {
	protocol := "http"
	if portSpec.Port == 443 || strings.HasPrefix(portSpec.Name, "https") {
		protocol = "https"
	}

	return protocol
}

func throttleEvents(ctx context.Context, throttleDuration time.Duration, pool *safe.Pool, eventsChan <-chan interface{}) chan interface{} {
	if throttleDuration == 0 {
		return nil
	}
	// Create a buffered channel to hold the pending event (if we're delaying processing the event due to throttling)
	eventsChanBuffered := make(chan interface{}, 1)

	// Run a goroutine that reads events from eventChan and does a non-blocking write to pendingEvent.
	// This guarantees that writing to eventChan will never block,
	// and that pendingEvent will have something in it if there's been an event since we read from that channel.
	pool.GoCtx(func(ctxPool context.Context) {
		for {
			select {
			case <-ctxPool.Done():
				return
			case nextEvent := <-eventsChan:
				select {
				case eventsChanBuffered <- nextEvent:
				default:
					// We already have an event in eventsChanBuffered, so we'll do a refresh as soon as our throttle allows us to.
					// It's fine to drop the event and keep whatever's in the buffer -- we don't do different things for different events
					log.FromContext(ctx).Debugf("Dropping event kind %T due to throttling", nextEvent)
				}
			}
		}
	})

	return eventsChanBuffered
}

func isInternalService(forwardTo v1alpha1.HTTPRouteForwardTo) bool {
	return forwardTo.ServiceName == nil &&
		forwardTo.BackendRef != nil &&
		forwardTo.BackendRef.Kind == traefikServiceKind &&
		forwardTo.BackendRef.Group == traefikServiceGroupName &&
		strings.HasSuffix(forwardTo.BackendRef.Name, "@internal")
}
