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
	"sigs.k8s.io/service-apis/apis/v1alpha1"
)

const (
	providerName            = "kubernetesgateway"
	traefikServiceKind      = "TraefikService"
	traefikServiceGroupName = "traefik.containo.us"
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

		cfg, err := p.createGatewayConf(client, gateway)
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

func (p *Provider) createGatewayConf(client Client, gateway *v1alpha1.Gateway) (*dynamic.Configuration, error) {
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
	listenerStatuses := p.fillGatewayConf(client, gateway, conf, tlsConfigs)

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

func (p *Provider) fillGatewayConf(client Client, gateway *v1alpha1.Gateway, conf *dynamic.Configuration, tlsConfigs map[string]*tls.CertAndStores) []v1alpha1.ListenerStatus {
	listenerStatuses := make([]v1alpha1.ListenerStatus, len(gateway.Spec.Listeners))

	for i, listener := range gateway.Spec.Listeners {
		listenerStatuses[i] = v1alpha1.ListenerStatus{
			Port:       listener.Port,
			Conditions: []metav1.Condition{},
		}

		// Supported Protocol
		if listener.Protocol != v1alpha1.HTTPProtocolType && listener.Protocol != v1alpha1.HTTPSProtocolType {
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

		if listener.Protocol == v1alpha1.HTTPSProtocolType {
			if listener.TLS == nil {
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

		// Supported Route types
		if listener.Routes.Kind != "HTTPRoute" {
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

		// TODO: support RouteNamespaces
		httpRoutes, err := client.GetHTTPRoutes(gateway.Namespace, labels.SelectorFromSet(listener.Routes.Selector.MatchLabels))
		if err != nil {
			// update "ResolvedRefs" status true with "InvalidRoutesRef" reason
			listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, metav1.Condition{
				Type:               string(v1alpha1.ListenerConditionResolvedRefs),
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             string(v1alpha1.ListenerReasonInvalidRoutesRef),
				Message:            fmt.Sprintf("Cannot fetch HTTPRoutes for namespace %q and matchLabels %v", gateway.Namespace, listener.Routes.Selector.MatchLabels),
			})
			continue
		}

		for _, httpRoute := range httpRoutes {
			// Should never happen
			if httpRoute == nil {
				continue
			}

			hostRule := hostRule(httpRoute.Spec)

			for _, routeRule := range httpRoute.Spec.Rules {
				rule, err := extractRule(routeRule, hostRule)
				if err != nil {
					// update "ResolvedRefs" status true with "DroppedRoutes" reason
					listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, metav1.Condition{
						Type:               string(v1alpha1.ListenerConditionResolvedRefs),
						Status:             metav1.ConditionFalse,
						LastTransitionTime: metav1.Now(),
						Reason:             string(v1alpha1.ListenerReasonDegradedRoutes),
						Message:            fmt.Sprintf("Skipping HTTPRoute %s: cannot generate rule: %v", httpRoute.Name, err),
					})
					continue
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
					listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, metav1.Condition{
						Type:               string(v1alpha1.ListenerConditionResolvedRefs),
						Status:             metav1.ConditionFalse,
						LastTransitionTime: metav1.Now(),
						Reason:             string(v1alpha1.ListenerReasonDegradedRoutes),
						Message:            fmt.Sprintf("Skipping HTTPRoute %s: cannot make router's key with rule %s: %v", httpRoute.Name, router.Rule, err),
					})

					// TODO update the RouteStatus condition / deduplicate conditions on listener
					continue
				}

				if routeRule.ForwardTo != nil {
					// Traefik internal service can be used only if there is only one ForwardTo service reference.
					if len(routeRule.ForwardTo) == 1 && isInternalService(routeRule.ForwardTo[0]) {
						router.Service = routeRule.ForwardTo[0].BackendRef.Name
					} else {
						wrrService, subServices, err := loadServices(client, gateway.Namespace, routeRule.ForwardTo)
						if err != nil {
							// update "ResolvedRefs" status true with "DroppedRoutes" reason
							listenerStatuses[i].Conditions = append(listenerStatuses[i].Conditions, metav1.Condition{
								Type:               string(v1alpha1.ListenerConditionResolvedRefs),
								Status:             metav1.ConditionFalse,
								LastTransitionTime: metav1.Now(),
								Reason:             string(v1alpha1.ListenerReasonDegradedRoutes),
								Message:            fmt.Sprintf("Cannot load service from HTTPRoute %s/%s : %v", gateway.Namespace, httpRoute.Name, err),
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
				}

				if router.Service != "" {
					routerKey = provider.Normalize(routerKey)

					conf.HTTP.Routers[routerKey] = &router
				}
			}
		}
	}

	return listenerStatuses
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

func hostRule(httpRouteSpec v1alpha1.HTTPRouteSpec) string {
	hostRule := ""
	for i, hostname := range httpRouteSpec.Hostnames {
		if i > 0 && len(hostname) > 0 {
			hostRule += "`, `"
		}
		hostRule += string(hostname)
	}

	if hostRule != "" {
		return "Host(`" + hostRule + "`)"
	}

	return ""
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

		if len(service.Spec.Ports) > 1 && forwardTo.Port == 0 {
			// If the port is unspecified and the backend is a Service
			// object consisting of multiple port definitions, the route
			// must be dropped from the Gateway. The controller should
			// raise the "ResolvedRefs" condition on the Gateway with the
			// "DroppedRoutes" reason.  The gateway status for this route
			// should be updated with a condition that describes the error
			// more specifically.
			log.WithoutContext().Errorf("A multiple ports Kubernetes Service cannot be used if unspecified forwardTo.Port")
			continue
		}

		var portName string
		var portSpec corev1.ServicePort
		var match bool

		for _, p := range service.Spec.Ports {
			if forwardTo.Port == 0 || p.Port == int32(forwardTo.Port) {
				portName = p.Name
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
				if portName == p.Name {
					port = p.Port
					break
				}
			}

			if port == 0 {
				return nil, nil, errors.New("cannot define a port")
			}

			protocol := getProtocol(portSpec, portName)

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

func getProtocol(portSpec corev1.ServicePort, portName string) string {
	protocol := "http"
	if portSpec.Port == 443 || strings.HasPrefix(portName, "https") {
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
