package kubernetes

import (
	"bufio"
	"bytes"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/cenk/backoff"
	"github.com/containous/flaeg"
	"github.com/traefik/traefik/job"
	"github.com/traefik/traefik/log"
	"github.com/traefik/traefik/provider"
	"github.com/traefik/traefik/provider/label"
	"github.com/traefik/traefik/safe"
	"github.com/traefik/traefik/tls"
	"github.com/traefik/traefik/types"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ provider.Provider = (*Provider)(nil)

const (
	ruleTypePath               = "Path"
	ruleTypePathPrefix         = "PathPrefix"
	ruleTypePathStrip          = "PathStrip"
	ruleTypePathPrefixStrip    = "PathPrefixStrip"
	ruleTypeAddPrefix          = "AddPrefix"
	ruleTypeReplacePath        = "ReplacePath"
	ruleTypeReplacePathRegex   = "ReplacePathRegex"
	traefikDefaultRealm        = "traefik"
	traefikDefaultIngressClass = "traefik"
	defaultBackendName         = "global-default-backend"
	defaultFrontendName        = "global-default-frontend"
	defaultFrontendRule        = "PathPrefix:/"
	allowedProtocolHTTPS       = "https"
	allowedProtocolH2C         = "h2c"
)

// IngressEndpoint holds the endpoint information for the Kubernetes provider
type IngressEndpoint struct {
	IP               string `description:"IP used for Kubernetes Ingress endpoints"`
	Hostname         string `description:"Hostname used for Kubernetes Ingress endpoints"`
	PublishedService string `description:"Published Kubernetes Service to copy status from"`
}

// Provider holds configurations of the provider.
type Provider struct {
	provider.BaseProvider  `mapstructure:",squash" export:"true"`
	Endpoint               string           `description:"Kubernetes server endpoint (required for external cluster client)"`
	Token                  string           `description:"Kubernetes bearer token (not needed for in-cluster client)"`
	CertAuthFilePath       string           `description:"Kubernetes certificate authority file path (not needed for in-cluster client)"`
	DisablePassHostHeaders bool             `description:"Kubernetes disable PassHost Headers" export:"true"`
	EnablePassTLSCert      bool             `description:"Kubernetes enable Pass TLS Client Certs" export:"true"` // Deprecated
	Namespaces             Namespaces       `description:"Kubernetes namespaces" export:"true"`
	LabelSelector          string           `description:"Kubernetes Ingress label selector to use" export:"true"`
	IngressClass           string           `description:"Value of kubernetes.io/ingress.class annotation to watch for" export:"true"`
	IngressEndpoint        *IngressEndpoint `description:"Kubernetes Ingress Endpoint"`
	ThrottleDuration       flaeg.Duration   `description:"Ingress refresh throttle duration"`
	lastConfiguration      safe.Safe
}

func (p *Provider) newK8sClient(ingressLabelSelector string) (Client, error) {
	ingLabelSel, err := labels.Parse(ingressLabelSelector)
	if err != nil {
		return nil, fmt.Errorf("invalid ingress label selector: %q", ingressLabelSelector)
	}
	log.Infof("ingress label selector is: %q", ingLabelSel)

	withEndpoint := ""
	if p.Endpoint != "" {
		withEndpoint = fmt.Sprintf(" with endpoint %v", p.Endpoint)
	}

	var cl *clientImpl
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" && os.Getenv("KUBERNETES_SERVICE_PORT") != "" {
		log.Infof("Creating in-cluster Provider client%s", withEndpoint)
		cl, err = newInClusterClient(p.Endpoint)
	} else {
		log.Infof("Creating cluster-external Provider client%s", withEndpoint)
		cl, err = newExternalClusterClient(p.Endpoint, p.Token, p.CertAuthFilePath)
	}

	if err == nil {
		cl.ingressLabelSelector = ingLabelSel
	}

	return cl, err
}

// Init the provider
func (p *Provider) Init(constraints types.Constraints) error {
	return p.BaseProvider.Init(constraints)
}

// Provide allows the k8s provider to provide configurations to traefik
// using the given configuration channel.
func (p *Provider) Provide(configurationChan chan<- types.ConfigMessage, pool *safe.Pool) error {
	// Tell glog (used by client-go) to log into STDERR. Otherwise, we risk
	// certain kinds of API errors getting logged into a directory not
	// available in a `FROM scratch` Docker container, causing glog to abort
	// hard with an exit code > 0.
	err := flag.Set("logtostderr", "true")
	if err != nil {
		return err
	}

	log.Debugf("Using Ingress label selector: %q", p.LabelSelector)
	k8sClient, err := p.newK8sClient(p.LabelSelector)
	if err != nil {
		return err
	}

	pool.Go(func(stop chan bool) {
		operation := func() error {
			stopWatch := make(chan struct{}, 1)
			defer close(stopWatch)
			eventsChan, err := k8sClient.WatchAll(p.Namespaces, stopWatch)
			if err != nil {
				log.Errorf("Error watching kubernetes events: %v", err)
				timer := time.NewTimer(1 * time.Second)
				select {
				case <-timer.C:
					return err
				case <-stop:
					return nil
				}
			}

			throttleDuration := time.Duration(p.ThrottleDuration)
			throttledChan := throttleEvents(throttleDuration, stop, eventsChan)
			if throttledChan != nil {
				eventsChan = throttledChan
			}

			for {
				select {
				case <-stop:
					return nil
				case event := <-eventsChan:
					// Note that event is the *first* event that came in during this
					// throttling interval -- if we're hitting our throttle, we may have
					// dropped events. This is fine, because we don't treat different
					// event types differently. But if we do in the future, we'll need to
					// track more information about the dropped events.
					log.Debugf("Received Kubernetes event kind %T", event)
					templateObjects, err := p.loadIngresses(k8sClient)
					if err != nil {
						return err
					}

					if reflect.DeepEqual(p.lastConfiguration.Get(), templateObjects) {
						log.Debugf("Skipping Kubernetes event kind %T", event)
					} else {
						p.lastConfiguration.Set(templateObjects)
						configurationChan <- types.ConfigMessage{
							ProviderName:  "kubernetes",
							Configuration: p.loadConfig(*templateObjects),
						}
					}

					// If we're throttling, we sleep here for the throttle duration to
					// enforce that we don't refresh faster than our throttle. time.Sleep
					// returns immediately if p.ThrottleDuration is 0 (no throttle).
					time.Sleep(throttleDuration)
				}
			}
		}

		notify := func(err error, time time.Duration) {
			log.Errorf("Provider connection error: %s; retrying in %s", err, time)
		}
		err := backoff.RetryNotify(safe.OperationWithRecover(operation), job.NewBackOff(backoff.NewExponentialBackOff()), notify)
		if err != nil {
			log.Errorf("Cannot connect to Provider: %s", err)
		}
	})

	return nil
}

func (p *Provider) loadIngresses(k8sClient Client) (*types.Configuration, error) {
	ingresses := k8sClient.GetIngresses()

	templateObjects := &types.Configuration{
		Backends:  map[string]*types.Backend{},
		Frontends: map[string]*types.Frontend{},
	}

	tlsConfigs := map[string]*tls.Configuration{}

	for _, i := range ingresses {
		ingressClass, err := getStringSafeValue(i.Annotations, annotationKubernetesIngressClass, "")
		if err != nil {
			log.Errorf("Misconfigured ingress class for ingress %s/%s: %v", i.Namespace, i.Name, err)
			continue
		}

		if !p.shouldProcessIngress(ingressClass) {
			continue
		}

		if err = getTLS(i, k8sClient, tlsConfigs); err != nil {
			log.Errorf("Error configuring TLS for ingress %s/%s: %v", i.Namespace, i.Name, err)
			continue
		}

		if i.Spec.Backend != nil {
			err := p.addGlobalBackend(k8sClient, i, templateObjects)
			if err != nil {
				log.Errorf("Error creating global backend for ingress %s/%s: %v", i.Namespace, i.Name, err)
				continue
			}
		}

		var weightAllocator weightAllocator = &defaultWeightAllocator{}
		annotationPercentageWeights := getAnnotationName(i.Annotations, annotationKubernetesServiceWeights)
		if _, ok := i.Annotations[annotationPercentageWeights]; ok {
			fractionalAllocator, err := newFractionalWeightAllocator(i, k8sClient)
			if err != nil {
				log.Errorf("failed to create fractional weight allocator for ingress %s/%s: %v", i.Namespace, i.Name, err)
				continue
			}
			log.Debugf("Created custom weight allocator for %s/%s: %s", i.Namespace, i.Name, fractionalAllocator)
			weightAllocator = fractionalAllocator
		}

		for _, r := range i.Spec.Rules {
			if r.HTTP == nil {
				log.Warn("Error in ingress: HTTP is nil")
				continue
			}

			for _, pa := range r.HTTP.Paths {
				priority := getIntValue(i.Annotations, annotationKubernetesPriority, 0)

				err := templateSafeString(r.Host)
				if err != nil {
					log.Errorf("failed to validate host %q for ingress %s/%s: %v", r.Host, i.Namespace, i.Name, err)
					continue
				}

				err = templateSafeString(pa.Path)
				if err != nil {
					log.Errorf("failed to validate path %q for ingress %s/%s: %v", pa.Path, i.Namespace, i.Name, err)
					continue
				}

				baseName := r.Host + pa.Path

				if len(baseName) == 0 {
					baseName = pa.Backend.ServiceName
				}

				entryPoints := getSliceStringValue(i.Annotations, annotationKubernetesFrontendEntryPoints)
				if len(entryPoints) > 0 {
					baseName = strings.Join(entryPoints, "-") + "_" + baseName
				}

				if priority > 0 {
					baseName = strconv.Itoa(priority) + "-" + baseName
				}

				if _, exists := templateObjects.Backends[baseName]; !exists {
					templateObjects.Backends[baseName] = &types.Backend{
						Servers: make(map[string]types.Server),
						LoadBalancer: &types.LoadBalancer{
							Method: "wrr",
						},
					}
				}

				annotationAuthRealm := getAnnotationName(i.Annotations, annotationKubernetesAuthRealm)
				if realm := i.Annotations[annotationAuthRealm]; realm != "" && realm != traefikDefaultRealm {
					log.Errorf("Value for annotation %q on ingress %s/%s invalid: no realm customization supported", annotationAuthRealm, i.Namespace, i.Name)
					delete(templateObjects.Backends, baseName)
					continue
				}

				var frontend *types.Frontend
				if fe, exists := templateObjects.Frontends[baseName]; exists {
					frontend = fe
				} else {
					auth, err := getAuthConfig(i, k8sClient)
					if err != nil {
						log.Errorf("Failed to retrieve auth configuration for ingress %s/%s: %s", i.Namespace, i.Name, err)
						continue
					}

					passHostHeader := getBoolValue(i.Annotations, annotationKubernetesPreserveHost, !p.DisablePassHostHeaders)
					passTLSCert := getBoolValue(i.Annotations, annotationKubernetesPassTLSCert, p.EnablePassTLSCert) // Deprecated

					frontend = &types.Frontend{
						Backend:           baseName,
						PassHostHeader:    passHostHeader,
						PassTLSCert:       passTLSCert,
						PassTLSClientCert: getPassTLSClientCert(i),
						Routes:            make(map[string]types.Route),
						Priority:          priority,
						WhiteList:         getWhiteList(i),
						Redirect:          getFrontendRedirect(i, baseName, pa.Path),
						EntryPoints:       entryPoints,
						Headers:           getHeader(i),
						Errors:            getErrorPages(i),
						RateLimit:         getRateLimit(i),
						Auth:              auth,
					}
				}

				service, exists, err := k8sClient.GetService(i.Namespace, pa.Backend.ServiceName)
				if err != nil {
					log.Errorf("Error while retrieving service information from k8s API %s/%s: %v", i.Namespace, pa.Backend.ServiceName, err)
					return nil, err
				}

				if !exists {
					log.Errorf("Service not found for %s/%s", i.Namespace, pa.Backend.ServiceName)
					continue
				}

				rule, err := getRuleForPath(pa, i)
				if err != nil {
					log.Errorf("Failed to get rule for ingress %s/%s: %s", i.Namespace, i.Name, err)
					continue
				}

				if rule != "" {
					frontend.Routes[pa.Path] = types.Route{
						Rule: rule,
					}
				}

				if len(r.Host) > 0 {
					if _, exists := frontend.Routes[r.Host]; !exists {
						frontend.Routes[r.Host] = types.Route{
							Rule: getRuleForHost(r.Host),
						}
					}
				}

				if len(frontend.Routes) == 0 {
					frontend.Routes["/"] = types.Route{
						Rule: defaultFrontendRule,
					}
				}

				templateObjects.Frontends[baseName] = frontend
				templateObjects.Backends[baseName].CircuitBreaker = getCircuitBreaker(service)
				templateObjects.Backends[baseName].LoadBalancer = getLoadBalancer(service)
				templateObjects.Backends[baseName].MaxConn = getMaxConn(service)
				templateObjects.Backends[baseName].Buffering = getBuffering(service)
				templateObjects.Backends[baseName].ResponseForwarding = getResponseForwarding(service)

				protocol := label.DefaultProtocol

				for _, port := range service.Spec.Ports {
					if equalPorts(port, pa.Backend.ServicePort) {
						if port.Port == 443 || strings.HasPrefix(port.Name, "https") {
							protocol = "https"
						}

						protocol = getStringValue(i.Annotations, annotationKubernetesProtocol, protocol)
						switch protocol {
						case allowedProtocolHTTPS:
						case allowedProtocolH2C:
						case label.DefaultProtocol:
						default:
							log.Errorf("Invalid protocol %s/%s specified for Ingress %s - skipping", annotationKubernetesProtocol, i.Namespace, i.Name)
							continue
						}

						// We have to treat external-name service differently here b/c it doesn't have any endpoints
						if service.Spec.Type == corev1.ServiceTypeExternalName {
							url := protocol + "://" + service.Spec.ExternalName
							if port.Port != 443 && port.Port != 80 {
								url = fmt.Sprintf("%s:%d", url, port.Port)
							}
							externalNameServiceWeight := weightAllocator.getWeight(r.Host, pa.Path, pa.Backend.ServiceName)
							templateObjects.Backends[baseName].Servers[url] = types.Server{
								URL:    url,
								Weight: externalNameServiceWeight,
							}
						} else {
							endpoints, exists, err := k8sClient.GetEndpoints(service.Namespace, service.Name)
							if err != nil {
								log.Errorf("Error retrieving endpoints %s/%s: %v", service.Namespace, service.Name, err)
								return nil, err
							}

							if !exists {
								log.Warnf("Endpoints not found for %s/%s", service.Namespace, service.Name)
								break
							}

							if len(endpoints.Subsets) == 0 {
								log.Warnf("Endpoints not available for %s/%s", service.Namespace, service.Name)
								break
							}

							for _, subset := range endpoints.Subsets {
								endpointPort := endpointPortNumber(port, subset.Ports)
								if endpointPort == 0 {
									// endpoint port does not match service.
									continue
								}
								for _, address := range subset.Addresses {
									url := protocol + "://" + net.JoinHostPort(address.IP, strconv.FormatInt(int64(endpointPort), 10))
									name := url
									if address.TargetRef != nil && address.TargetRef.Name != "" {
										name = address.TargetRef.Name
									}

									templateObjects.Backends[baseName].Servers[name] = types.Server{
										URL:    url,
										Weight: weightAllocator.getWeight(r.Host, pa.Path, pa.Backend.ServiceName),
									}
								}
							}
						}
						break
					}
				}
			}
		}

		err = p.updateIngressStatus(i, k8sClient)
		if err != nil {
			log.Errorf("Cannot update Ingress %s/%s due to error: %v", i.Namespace, i.Name, err)
		}
	}

	templateObjects.TLS = getTLSConfig(tlsConfigs)

	return templateObjects, nil
}

func (p *Provider) updateIngressStatus(i *extensionsv1beta1.Ingress, k8sClient Client) error {
	// Only process if an IngressEndpoint has been configured
	if p.IngressEndpoint == nil {
		return nil
	}

	if len(p.IngressEndpoint.PublishedService) == 0 {
		if len(p.IngressEndpoint.IP) == 0 && len(p.IngressEndpoint.Hostname) == 0 {
			return errors.New("publishedService or ip or hostname must be defined")
		}

		return k8sClient.UpdateIngressStatus(i.Namespace, i.Name, p.IngressEndpoint.IP, p.IngressEndpoint.Hostname)
	}

	serviceInfo := strings.Split(p.IngressEndpoint.PublishedService, "/")
	if len(serviceInfo) != 2 {
		return fmt.Errorf("invalid publishedService format (expected 'namespace/service' format): %s", p.IngressEndpoint.PublishedService)
	}
	serviceNamespace, serviceName := serviceInfo[0], serviceInfo[1]

	service, exists, err := k8sClient.GetService(serviceNamespace, serviceName)
	if err != nil {
		return fmt.Errorf("cannot get service %s, received error: %s", p.IngressEndpoint.PublishedService, err)
	}

	if exists && service.Status.LoadBalancer.Ingress == nil {
		// service exists, but has no Load Balancer status
		log.Debugf("Skipping updating Ingress %s/%s due to service %s having no status set", i.Namespace, i.Name, p.IngressEndpoint.PublishedService)
		return nil
	}

	if !exists {
		return fmt.Errorf("missing service: %s", p.IngressEndpoint.PublishedService)
	}

	return k8sClient.UpdateIngressStatus(i.Namespace, i.Name, service.Status.LoadBalancer.Ingress[0].IP, service.Status.LoadBalancer.Ingress[0].Hostname)
}

func (p *Provider) loadConfig(templateObjects types.Configuration) *types.Configuration {
	var FuncMap = template.FuncMap{}
	configuration, err := p.GetConfiguration("templates/kubernetes.tmpl", FuncMap, templateObjects)
	if err != nil {
		log.Error(err)
	}
	return configuration
}

func (p *Provider) addGlobalBackend(cl Client, i *extensionsv1beta1.Ingress, templateObjects *types.Configuration) error {
	// Ensure that we are not duplicating the frontend
	if _, exists := templateObjects.Frontends[defaultFrontendName]; exists {
		return errors.New("duplicate frontend: " + defaultFrontendName)
	}

	// Ensure we are not duplicating the backend
	if _, exists := templateObjects.Backends[defaultBackendName]; exists {
		return errors.New("duplicate backend: " + defaultBackendName)
	}

	templateObjects.Backends[defaultBackendName] = &types.Backend{
		Servers: make(map[string]types.Server),
		LoadBalancer: &types.LoadBalancer{
			Method: "wrr",
		},
	}

	service, exists, err := cl.GetService(i.Namespace, i.Spec.Backend.ServiceName)
	if err != nil {
		return fmt.Errorf("error while retrieving service information from k8s API %s/%s: %v", i.Namespace, i.Spec.Backend.ServiceName, err)
	}
	if !exists {
		return fmt.Errorf("service not found for %s/%s", i.Namespace, i.Spec.Backend.ServiceName)
	}

	templateObjects.Backends[defaultBackendName].CircuitBreaker = getCircuitBreaker(service)
	templateObjects.Backends[defaultBackendName].LoadBalancer = getLoadBalancer(service)
	templateObjects.Backends[defaultBackendName].MaxConn = getMaxConn(service)
	templateObjects.Backends[defaultBackendName].Buffering = getBuffering(service)
	templateObjects.Backends[defaultBackendName].ResponseForwarding = getResponseForwarding(service)

	for _, port := range service.Spec.Ports {

		if !equalPorts(port, i.Spec.Backend.ServicePort) {
			continue
		}

		// We have to treat external-name service differently here b/c it doesn't have any endpoints
		if service.Spec.Type == corev1.ServiceTypeExternalName {

			protocol := "http"
			if port.Port == 443 || strings.HasPrefix(port.Name, "https") {
				protocol = "https"
			}

			url := protocol + "://" + service.Spec.ExternalName
			if port.Port != 443 && port.Port != 80 {
				url = fmt.Sprintf("%s:%d", url, port.Port)
			}

			templateObjects.Backends[defaultBackendName].Servers[url] = types.Server{
				URL:    url,
				Weight: label.DefaultWeight,
			}

		} else {

			endpoints, exists, err := cl.GetEndpoints(service.Namespace, service.Name)
			if err != nil {
				return fmt.Errorf("error retrieving endpoint information from k8s API %s/%s: %v", service.Namespace, service.Name, err)
			}
			if !exists {
				return fmt.Errorf("endpoints not found for %s/%s", service.Namespace, service.Name)
			}
			if len(endpoints.Subsets) == 0 {
				return fmt.Errorf("endpoints not available for %s/%s", service.Namespace, service.Name)
			}

			for _, subset := range endpoints.Subsets {

				endpointPort := endpointPortNumber(port, subset.Ports)
				if endpointPort == 0 {
					// endpoint port does not match service.
					continue
				}

				protocol := "http"
				for _, address := range subset.Addresses {
					if endpointPort == 443 || strings.HasPrefix(i.Spec.Backend.ServicePort.String(), "https") {
						protocol = "https"
					}

					url := fmt.Sprintf("%s://%s", protocol, net.JoinHostPort(address.IP, strconv.FormatInt(int64(endpointPort), 10)))
					name := url
					if address.TargetRef != nil && address.TargetRef.Name != "" {
						name = address.TargetRef.Name
					}

					templateObjects.Backends[defaultBackendName].Servers[name] = types.Server{
						URL:    url,
						Weight: label.DefaultWeight,
					}
				}
			}
		}
	}

	passHostHeader := getBoolValue(i.Annotations, annotationKubernetesPreserveHost, !p.DisablePassHostHeaders)
	passTLSCert := getBoolValue(i.Annotations, annotationKubernetesPassTLSCert, p.EnablePassTLSCert) // Deprecated
	priority := getIntValue(i.Annotations, annotationKubernetesPriority, 0)
	entryPoints := getSliceStringValue(i.Annotations, annotationKubernetesFrontendEntryPoints)

	templateObjects.Frontends[defaultFrontendName] = &types.Frontend{
		Backend:           defaultBackendName,
		PassHostHeader:    passHostHeader,
		PassTLSCert:       passTLSCert,
		PassTLSClientCert: getPassTLSClientCert(i),
		Routes:            make(map[string]types.Route),
		Priority:          priority,
		WhiteList:         getWhiteList(i),
		Redirect:          getFrontendRedirect(i, defaultFrontendName, "/"),
		EntryPoints:       entryPoints,
		Headers:           getHeader(i),
		Errors:            getErrorPages(i),
		RateLimit:         getRateLimit(i),
	}

	templateObjects.Frontends[defaultFrontendName].Routes["/"] = types.Route{
		Rule: defaultFrontendRule,
	}

	return nil
}

func throttleEvents(throttleDuration time.Duration, stop chan bool, eventsChan <-chan interface{}) chan interface{} {
	if throttleDuration == 0 {
		return nil
	}
	// Create a buffered channel to hold the pending event (if we're delaying processing the event due to throttling)
	eventsChanBuffered := make(chan interface{}, 1)

	// Run a goroutine that reads events from eventChan and does a
	// non-blocking write to pendingEvent. This guarantees that writing to
	// eventChan will never block, and that pendingEvent will have
	// something in it if there's been an event since we read from that channel.
	go func() {
		for {
			select {
			case <-stop:
				return
			case nextEvent := <-eventsChan:
				select {
				case eventsChanBuffered <- nextEvent:
				default:
					// We already have an event in eventsChanBuffered, so we'll
					// do a refresh as soon as our throttle allows us to. It's fine
					// to drop the event and keep whatever's in the buffer -- we
					// don't do different things for different events
					log.Debugf("Dropping event kind %T due to throttling", nextEvent)
				}
			}
		}
	}()

	return eventsChanBuffered
}

func getRuleForPath(pa extensionsv1beta1.HTTPIngressPath, i *extensionsv1beta1.Ingress) (string, error) {
	if len(pa.Path) == 0 {
		return "", nil
	}

	ruleType := getStringValue(i.Annotations, annotationKubernetesRuleType, ruleTypePathPrefix)

	switch ruleType {
	case ruleTypePath, ruleTypePathPrefix, ruleTypePathStrip, ruleTypePathPrefixStrip:
	case ruleTypeReplacePath:
		log.Warnf("Using %s as %s will be deprecated in the future. Please use the %s annotation instead", ruleType, annotationKubernetesRuleType, annotationKubernetesRequestModifier)
	default:
		return "", fmt.Errorf("cannot use non-matcher rule: %q", ruleType)
	}

	rules := []string{ruleType + ":" + pa.Path}

	if rewriteTarget := getStringValue(i.Annotations, annotationKubernetesRewriteTarget, ""); rewriteTarget != "" {
		if ruleType == ruleTypeReplacePath {
			return "", fmt.Errorf("rewrite-target must not be used together with annotation %q", annotationKubernetesRuleType)
		}
		rewriteTargetRule := fmt.Sprintf("ReplacePathRegex: ^%s(.*) %s$1", pa.Path, strings.TrimRight(rewriteTarget, "/"))
		if pa.Path == "/" {
			// If path = /, then just append the cap group, as if we don't cap the path as part of the regex,
			// then when we strip the right / from the rewrite target, it ends up being missed, as removed but never returned
			// this only happens when path = / because it is the only case where TrimRight will catch a leading /.
			rewriteTargetRule = fmt.Sprintf("ReplacePathRegex: ^(.*) %s$1", strings.TrimRight(rewriteTarget, "/"))
		}
		rules = append(rules, rewriteTargetRule)
	}

	if requestModifier := getStringValue(i.Annotations, annotationKubernetesRequestModifier, ""); requestModifier != "" {
		rule, err := parseRequestModifier(requestModifier, ruleType)
		if err != nil {
			return "", err
		}

		rules = append(rules, rule)
	}

	return strings.Join(rules, ";"), nil
}

func parseRequestModifier(requestModifier, ruleType string) (string, error) {
	trimmedRequestModifier := strings.TrimRight(requestModifier, " :")
	if trimmedRequestModifier == "" {
		return "", fmt.Errorf("rule %q is empty", requestModifier)
	}

	// Split annotation to determine modifier type
	modifierParts := strings.Split(trimmedRequestModifier, ":")
	if len(modifierParts) < 2 {
		return "", fmt.Errorf("rule %q is missing type or value", requestModifier)
	}

	modifier := strings.TrimSpace(modifierParts[0])
	value := strings.TrimSpace(modifierParts[1])

	switch modifier {
	case ruleTypeAddPrefix, ruleTypeReplacePath, ruleTypeReplacePathRegex:
		if ruleType == ruleTypeReplacePath {
			return "", fmt.Errorf("cannot use '%s: %s' and '%s: %s', as this leads to rule duplication, and unintended behavior",
				annotationKubernetesRuleType, ruleTypeReplacePath, annotationKubernetesRequestModifier, modifier)
		}
	case "":
		return "", errors.New("cannot use empty rule")
	default:
		return "", fmt.Errorf("cannot use non-modifier rule: %q", modifier)
	}

	return modifier + ":" + value, nil
}

func getRuleForHost(host string) string {
	if strings.Contains(host, "*") {
		return "HostRegexp:" + strings.Replace(host, "*", "{subdomain:[A-Za-z0-9-_]+}", 1)
	}
	return "Host:" + host
}

func getTLS(ingress *extensionsv1beta1.Ingress, k8sClient Client, tlsConfigs map[string]*tls.Configuration) error {
	for _, t := range ingress.Spec.TLS {
		if t.SecretName == "" {
			log.Debugf("Skipping TLS sub-section for ingress %s/%s: No secret name provided", ingress.Namespace, ingress.Name)
			continue
		}

		newEntryPoints := getSliceStringValue(ingress.Annotations, annotationKubernetesFrontendEntryPoints)

		configKey := ingress.Namespace + "/" + t.SecretName
		if tlsConfig, tlsExists := tlsConfigs[configKey]; tlsExists {
			for _, entryPoint := range newEntryPoints {
				tlsConfig.EntryPoints = mergeEntryPoint(tlsConfig.EntryPoints, entryPoint)
			}
		} else {
			secret, exists, err := k8sClient.GetSecret(ingress.Namespace, t.SecretName)
			if err != nil {
				return fmt.Errorf("failed to fetch secret %s/%s: %v", ingress.Namespace, t.SecretName, err)
			}
			if !exists {
				return fmt.Errorf("secret %s/%s does not exist", ingress.Namespace, t.SecretName)
			}

			cert, key, err := getCertificateBlocks(secret, ingress.Namespace, t.SecretName)
			if err != nil {
				return err
			}

			sort.Strings(newEntryPoints)

			tlsConfig = &tls.Configuration{
				EntryPoints: newEntryPoints,
				Certificate: &tls.Certificate{
					CertFile: tls.FileOrContent(cert),
					KeyFile:  tls.FileOrContent(key),
				},
			}
			tlsConfigs[configKey] = tlsConfig
		}
	}

	return nil
}

func getTLSConfig(tlsConfigs map[string]*tls.Configuration) []*tls.Configuration {
	var secretNames []string
	for secretName := range tlsConfigs {
		secretNames = append(secretNames, secretName)
	}
	sort.Strings(secretNames)

	var configs []*tls.Configuration
	for _, secretName := range secretNames {
		configs = append(configs, tlsConfigs[secretName])
	}

	return configs
}

func mergeEntryPoint(entryPoints []string, newEntryPoint string) []string {
	for _, ep := range entryPoints {
		if ep == newEntryPoint {
			return entryPoints
		}
	}
	entryPoints = append(entryPoints, newEntryPoint)
	sort.Strings(entryPoints)
	return entryPoints
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

	cert := string(tlsCrtData[:])
	if cert == "" {
		missingEntries = append(missingEntries, "tls.crt")
	}

	key := string(tlsKeyData[:])
	if key == "" {
		missingEntries = append(missingEntries, "tls.key")
	}

	if len(missingEntries) > 0 {
		return "", "", fmt.Errorf("secret %s/%s contains the following empty TLS data entries: %s",
			namespace, secretName, strings.Join(missingEntries, ", "))
	}

	if !isPem(tlsCrtData) {
		missingEntries = append(missingEntries, "tls.crt")
	}

	if !isPem(tlsKeyData) {
		missingEntries = append(missingEntries, "tls.key")
	}

	if len(missingEntries) > 0 {
		return "", "", fmt.Errorf("secret %s/%s does not contain PEM formatted TLS data entries: %s",
			namespace, secretName, strings.Join(missingEntries, ","))
	}

	return cert, key, nil
}

// endpointPortNumber returns the port to be used for this endpoint. It is zero
// if the endpoint does not match the given service port.
func endpointPortNumber(servicePort corev1.ServicePort, endpointPorts []corev1.EndpointPort) int32 {
	// Is this reasonable to assume?
	if len(endpointPorts) == 0 {
		return servicePort.Port
	}

	for _, endpointPort := range endpointPorts {
		// For matching endpoints, the port names must correspond, either by
		// being empty or non-empty. Multi-port services mandate non-empty
		// names and allow us to filter for the right addresses.
		if servicePort.Name == endpointPort.Name {
			return endpointPort.Port
		}
	}
	return 0
}

func equalPorts(servicePort corev1.ServicePort, ingressPort intstr.IntOrString) bool {
	if int(servicePort.Port) == ingressPort.IntValue() {
		return true
	}
	if servicePort.Name != "" && servicePort.Name == ingressPort.String() {
		return true
	}
	return false
}

func (p *Provider) shouldProcessIngress(annotationIngressClass string) bool {
	if len(p.IngressClass) == 0 {
		return len(annotationIngressClass) == 0 || annotationIngressClass == traefikDefaultIngressClass
	}
	return annotationIngressClass == p.IngressClass
}

func getAuthConfig(i *extensionsv1beta1.Ingress, k8sClient Client) (*types.Auth, error) {
	authType := getStringValue(i.Annotations, annotationKubernetesAuthType, "")
	if len(authType) == 0 {
		return nil, nil
	}

	auth := &types.Auth{
		HeaderField: getStringValue(i.Annotations, annotationKubernetesAuthHeaderField, ""),
	}

	switch strings.ToLower(authType) {
	case "basic":
		basic, err := getBasicAuthConfig(i, k8sClient)
		if err != nil {
			return nil, err
		}

		auth.Basic = basic
	case "digest":
		digest, err := getDigestAuthConfig(i, k8sClient)
		if err != nil {
			return nil, err
		}

		auth.Digest = digest
	case "forward":
		forward, err := getForwardAuthConfig(i, k8sClient)
		if err != nil {
			return nil, err
		}

		auth.Forward = forward
	default:
		return nil, fmt.Errorf("unsupported auth-type on annotation %s: %s", annotationKubernetesAuthType, authType)
	}

	return auth, nil
}

func getBasicAuthConfig(i *extensionsv1beta1.Ingress, k8sClient Client) (*types.Basic, error) {
	credentials, err := getAuthCredentials(i, k8sClient)
	if err != nil {
		return nil, err
	}

	return &types.Basic{
		Users:        credentials,
		RemoveHeader: getBoolValue(i.Annotations, annotationKubernetesAuthRemoveHeader, false),
	}, nil
}

func getDigestAuthConfig(i *extensionsv1beta1.Ingress, k8sClient Client) (*types.Digest, error) {
	credentials, err := getAuthCredentials(i, k8sClient)
	if err != nil {
		return nil, err
	}

	return &types.Digest{Users: credentials,
		RemoveHeader: getBoolValue(i.Annotations, annotationKubernetesAuthRemoveHeader, false),
	}, nil
}

func getAuthCredentials(i *extensionsv1beta1.Ingress, k8sClient Client) ([]string, error) {
	authSecret := getStringValue(i.Annotations, annotationKubernetesAuthSecret, "")
	if authSecret == "" {
		return nil, fmt.Errorf("auth-secret annotation %s must be set", annotationKubernetesAuthSecret)
	}

	auth, err := loadAuthCredentials(i.Namespace, authSecret, k8sClient)
	if err != nil {
		return nil, fmt.Errorf("failed to load auth credentials: %s", err)
	}

	return auth, nil
}

func loadAuthCredentials(namespace, secretName string, k8sClient Client) ([]string, error) {
	secret, ok, err := k8sClient.GetSecret(namespace, secretName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch secret %q/%q: %s", namespace, secretName, err)
	}
	if !ok {
		return nil, fmt.Errorf("secret %q/%q not found", namespace, secretName)
	}
	if secret == nil {
		return nil, fmt.Errorf("data for secret %q/%q must not be nil", namespace, secretName)
	}
	if len(secret.Data) != 1 {
		return nil, fmt.Errorf("found %d elements for secret %q/%q, must be single element exactly", len(secret.Data), namespace, secretName)
	}

	var firstSecret []byte
	for _, v := range secret.Data {
		firstSecret = v
		break
	}

	var credentials []string
	scanner := bufio.NewScanner(bytes.NewReader(firstSecret))
	for scanner.Scan() {
		if cred := scanner.Text(); len(cred) > 0 {
			credentials = append(credentials, cred)
		}
	}

	if len(credentials) == 0 {
		return nil, fmt.Errorf("secret %q/%q does not contain any credentials", namespace, secretName)
	}

	return credentials, nil
}

func getForwardAuthConfig(i *extensionsv1beta1.Ingress, k8sClient Client) (*types.Forward, error) {
	authURL := getStringValue(i.Annotations, annotationKubernetesAuthForwardURL, "")
	if len(authURL) == 0 {
		return nil, fmt.Errorf("forward authentication requires a url")
	}

	forwardAuth := &types.Forward{
		Address:             authURL,
		TrustForwardHeader:  getBoolValue(i.Annotations, annotationKubernetesAuthForwardTrustHeaders, false),
		AuthResponseHeaders: getSliceStringValue(i.Annotations, annotationKubernetesAuthForwardResponseHeaders),
	}

	authSecretName := getStringValue(i.Annotations, annotationKubernetesAuthForwardTLSSecret, "")
	authSecretCert, authSecretKey, err := loadAuthTLSSecret(i.Namespace, authSecretName, k8sClient)
	if err != nil {
		return nil, fmt.Errorf("failed to load auth secret: %s", err)
	}

	if authSecretCert != "" || authSecretKey != "" {
		forwardAuth.TLS = &types.ClientTLS{
			Cert:               authSecretCert,
			Key:                authSecretKey,
			InsecureSkipVerify: getBoolValue(i.Annotations, annotationKubernetesAuthForwardTLSInsecure, false),
		}
	}

	if forwardAuth.TLS == nil && label.Has(i.Annotations, getAnnotationName(i.Annotations, annotationKubernetesAuthForwardTLSInsecure)) {
		forwardAuth.TLS = &types.ClientTLS{
			InsecureSkipVerify: getBoolValue(i.Annotations, annotationKubernetesAuthForwardTLSInsecure, false),
		}
	}

	return forwardAuth, nil
}

func loadAuthTLSSecret(namespace, secretName string, k8sClient Client) (string, string, error) {
	if len(secretName) == 0 {
		return "", "", nil
	}

	secret, exists, err := k8sClient.GetSecret(namespace, secretName)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch secret %q/%q: %s", namespace, secretName, err)
	}
	if !exists {
		return "", "", fmt.Errorf("secret %q/%q does not exist", namespace, secretName)
	}
	if secret == nil {
		return "", "", fmt.Errorf("data for secret %q/%q must not be nil", namespace, secretName)
	}
	if len(secret.Data) != 2 {
		return "", "", fmt.Errorf("found %d elements for secret %q/%q, must be two elements exactly", len(secret.Data), namespace, secretName)
	}

	return getCertificateBlocks(secret, namespace, secretName)
}

func getFrontendRedirect(i *extensionsv1beta1.Ingress, baseName, path string) *types.Redirect {
	permanent := getBoolValue(i.Annotations, annotationKubernetesRedirectPermanent, false)

	if appRoot := getStringValue(i.Annotations, annotationKubernetesAppRoot, ""); appRoot != "" && (path == "/" || path == "") {
		regex := fmt.Sprintf("%s$", baseName)
		if path == "" {
			regex = fmt.Sprintf("%s/$", baseName)
		}
		return &types.Redirect{
			Regex:       regex,
			Replacement: fmt.Sprintf("%s/%s", strings.TrimRight(baseName, "/"), strings.TrimLeft(appRoot, "/")),
			Permanent:   permanent,
		}
	}

	redirectEntryPoint := getStringValue(i.Annotations, annotationKubernetesRedirectEntryPoint, "")
	if len(redirectEntryPoint) > 0 {
		return &types.Redirect{
			EntryPoint: redirectEntryPoint,
			Permanent:  permanent,
		}
	}

	redirectRegex, err := getStringSafeValue(i.Annotations, annotationKubernetesRedirectRegex, "")
	if err != nil {
		log.Debugf("Skipping Redirect on Ingress %s/%s due to invalid regex: %s", i.Namespace, i.Name, redirectRegex)
		return nil
	}

	redirectReplacement, err := getStringSafeValue(i.Annotations, annotationKubernetesRedirectReplacement, "")
	if err != nil {
		log.Debugf("Skipping Redirect on Ingress %s/%s due to invalid replacement: %q", i.Namespace, i.Name, redirectRegex)
		return nil
	}

	if len(redirectRegex) > 0 && len(redirectReplacement) > 0 {
		return &types.Redirect{
			Regex:       redirectRegex,
			Replacement: redirectReplacement,
			Permanent:   permanent,
		}
	}

	return nil
}

func getWhiteList(i *extensionsv1beta1.Ingress) *types.WhiteList {
	ranges := getSliceStringValue(i.Annotations, annotationKubernetesWhiteListSourceRange)
	if len(ranges) <= 0 {
		return nil
	}

	return &types.WhiteList{
		SourceRange:      ranges,
		UseXForwardedFor: getBoolValue(i.Annotations, annotationKubernetesWhiteListUseXForwardedFor, false),
	}
}

func getResponseForwarding(service *corev1.Service) *types.ResponseForwarding {
	flushIntervalValue := getStringValue(service.Annotations, annotationKubernetesResponseForwardingFlushInterval, "")
	if len(flushIntervalValue) == 0 {
		return nil
	}

	return &types.ResponseForwarding{
		FlushInterval: flushIntervalValue,
	}
}

func getBuffering(service *corev1.Service) *types.Buffering {
	var buffering *types.Buffering

	bufferingRaw := getStringValue(service.Annotations, annotationKubernetesBuffering, "")

	if len(bufferingRaw) > 0 {
		buffering = &types.Buffering{}
		err := yaml.Unmarshal([]byte(bufferingRaw), buffering)
		if err != nil {
			log.Error(err)
			return nil
		}
	}

	return buffering
}

func getLoadBalancer(service *corev1.Service) *types.LoadBalancer {
	loadBalancer := &types.LoadBalancer{
		Method: "wrr",
	}

	if getStringValue(service.Annotations, annotationKubernetesLoadBalancerMethod, "") == "drr" {
		loadBalancer.Method = "drr"
	}

	if sticky := service.Annotations[label.TraefikBackendLoadBalancerSticky]; len(sticky) > 0 {
		log.Warnf("Deprecated configuration found: %s. Please use %s.", label.TraefikBackendLoadBalancerSticky, annotationKubernetesAffinity)
		loadBalancer.Sticky = strings.EqualFold(strings.TrimSpace(sticky), "true")
	}

	if stickiness := getStickiness(service); stickiness != nil {
		loadBalancer.Stickiness = stickiness
	}

	return loadBalancer
}

func getStickiness(service *corev1.Service) *types.Stickiness {
	if !getBoolValue(service.Annotations, annotationKubernetesAffinity, false) {
		return nil
	}

	return &types.Stickiness{
		CookieName: getStringValue(service.Annotations, annotationKubernetesSessionCookieName, ""),
		Secure:     getBoolValue(service.Annotations, annotationKubernetesSessionSecure, false),
		HTTPOnly:   getBoolValue(service.Annotations, annotationKubernetesSessionHTTPOnly, false),
		SameSite:   getStringValue(service.Annotations, annotationKubernetesSessionSameSite, ""),
	}
}

func getHeader(i *extensionsv1beta1.Ingress) *types.Headers {
	headers := &types.Headers{
		CustomRequestHeaders:    getMapValue(i.Annotations, annotationKubernetesCustomRequestHeaders),
		CustomResponseHeaders:   getMapValue(i.Annotations, annotationKubernetesCustomResponseHeaders),
		AllowedHosts:            getSliceStringValue(i.Annotations, annotationKubernetesAllowedHosts),
		HostsProxyHeaders:       getSliceStringValue(i.Annotations, annotationKubernetesProxyHeaders),
		SSLForceHost:            getBoolValue(i.Annotations, annotationKubernetesSSLForceHost, false),
		SSLRedirect:             getBoolValue(i.Annotations, annotationKubernetesSSLRedirect, false),
		SSLTemporaryRedirect:    getBoolValue(i.Annotations, annotationKubernetesSSLTemporaryRedirect, false),
		SSLHost:                 getStringValue(i.Annotations, annotationKubernetesSSLHost, ""),
		SSLProxyHeaders:         getMapValue(i.Annotations, annotationKubernetesSSLProxyHeaders),
		STSSeconds:              getInt64Value(i.Annotations, annotationKubernetesHSTSMaxAge, 0),
		STSIncludeSubdomains:    getBoolValue(i.Annotations, annotationKubernetesHSTSIncludeSubdomains, false),
		STSPreload:              getBoolValue(i.Annotations, annotationKubernetesHSTSPreload, false),
		ForceSTSHeader:          getBoolValue(i.Annotations, annotationKubernetesForceHSTSHeader, false),
		FrameDeny:               getBoolValue(i.Annotations, annotationKubernetesFrameDeny, false),
		CustomFrameOptionsValue: getStringValue(i.Annotations, annotationKubernetesCustomFrameOptionsValue, ""),
		ContentTypeNosniff:      getBoolValue(i.Annotations, annotationKubernetesContentTypeNosniff, false),
		BrowserXSSFilter:        getBoolValue(i.Annotations, annotationKubernetesBrowserXSSFilter, false),
		CustomBrowserXSSValue:   getStringValue(i.Annotations, annotationKubernetesCustomBrowserXSSValue, ""),
		ContentSecurityPolicy:   getStringValue(i.Annotations, annotationKubernetesContentSecurityPolicy, ""),
		PublicKey:               getStringValue(i.Annotations, annotationKubernetesPublicKey, ""),
		ReferrerPolicy:          getStringValue(i.Annotations, annotationKubernetesReferrerPolicy, ""),
		IsDevelopment:           getBoolValue(i.Annotations, annotationKubernetesIsDevelopment, false),
	}

	if !headers.HasSecureHeadersDefined() && !headers.HasCustomHeadersDefined() {
		return nil
	}

	return headers
}

func getMaxConn(service *corev1.Service) *types.MaxConn {
	amount := getInt64Value(service.Annotations, annotationKubernetesMaxConnAmount, -1)
	extractorFunc := getStringValue(service.Annotations, annotationKubernetesMaxConnExtractorFunc, "")
	if amount >= 0 && len(extractorFunc) > 0 {
		return &types.MaxConn{
			ExtractorFunc: extractorFunc,
			Amount:        amount,
		}
	}
	return nil
}

func getCircuitBreaker(service *corev1.Service) *types.CircuitBreaker {
	if expression := getStringValue(service.Annotations, annotationKubernetesCircuitBreakerExpression, ""); expression != "" {
		return &types.CircuitBreaker{
			Expression: expression,
		}
	}
	return nil
}

func getErrorPages(i *extensionsv1beta1.Ingress) map[string]*types.ErrorPage {
	var errorPages map[string]*types.ErrorPage

	pagesRaw := getStringValue(i.Annotations, annotationKubernetesErrorPages, "")
	if len(pagesRaw) > 0 {
		errorPages = make(map[string]*types.ErrorPage)
		err := yaml.Unmarshal([]byte(pagesRaw), errorPages)
		if err != nil {
			log.Error(err)
			return nil
		}
	}

	return errorPages
}

func getRateLimit(i *extensionsv1beta1.Ingress) *types.RateLimit {
	var rateLimit *types.RateLimit

	rateRaw := getStringValue(i.Annotations, annotationKubernetesRateLimit, "")
	if len(rateRaw) > 0 {
		rateLimit = &types.RateLimit{}
		err := yaml.Unmarshal([]byte(rateRaw), rateLimit)
		if err != nil {
			log.Error(err)
			return nil
		}
	}

	return rateLimit
}

func getPassTLSClientCert(i *extensionsv1beta1.Ingress) *types.TLSClientHeaders {
	var passTLSClientCert *types.TLSClientHeaders

	passRaw := getStringValue(i.Annotations, annotationKubernetesPassTLSClientCert, "")
	if len(passRaw) > 0 {
		passTLSClientCert = &types.TLSClientHeaders{}
		err := yaml.Unmarshal([]byte(passRaw), passTLSClientCert)
		if err != nil {
			log.Error(err)
			return nil
		}
	}

	return passTLSClientCert
}

func templateSafeString(value string) error {
	_, err := strconv.Unquote(`"` + value + `"`)
	return err
}

func isPem(data []byte) bool {
	for {
		block, rest := pem.Decode(data)
		if block == nil {
			return false
		}
		if len(rest) == 0 {
			break
		}
		data = rest
	}
	return true
}
