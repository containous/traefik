package ingress

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/mitchellh/hashstructure"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/job"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/provider"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/traefik/traefik/v2/pkg/tls"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	annotationKubernetesIngressClass     = "kubernetes.io/ingress.class"
	traefikDefaultIngressClass           = "traefik"
	traefikDefaultIngressClassController = "traefik.io/ingress-controller"
	defaultPathMatcher                   = "PathPrefix"
)

// Provider holds configurations of the provider.
type Provider struct {
	Endpoint           string           `description:"Kubernetes server endpoint (required for external cluster client)." json:"endpoint,omitempty" toml:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Token              string           `description:"Kubernetes bearer token (not needed for in-cluster client)." json:"token,omitempty" toml:"token,omitempty" yaml:"token,omitempty"`
	CertAuthFilePath   string           `description:"Kubernetes certificate authority file path (not needed for in-cluster client)." json:"certAuthFilePath,omitempty" toml:"certAuthFilePath,omitempty" yaml:"certAuthFilePath,omitempty"`
	Namespaces         []string         `description:"Kubernetes namespaces." json:"namespaces,omitempty" toml:"namespaces,omitempty" yaml:"namespaces,omitempty" export:"true"`
	LabelSelector      string           `description:"Kubernetes Ingress label selector to use." json:"labelSelector,omitempty" toml:"labelSelector,omitempty" yaml:"labelSelector,omitempty" export:"true"`
	IngressClass       string           `description:"Value of kubernetes.io/ingress.class annotation or IngressClass name to watch for." json:"ingressClass,omitempty" toml:"ingressClass,omitempty" yaml:"ingressClass,omitempty" export:"true"`
	IngressEndpoint    *EndpointIngress `description:"Kubernetes Ingress Endpoint." json:"ingressEndpoint,omitempty" toml:"ingressEndpoint,omitempty" yaml:"ingressEndpoint,omitempty" export:"true"`
	ThrottleDuration   ptypes.Duration  `description:"Ingress refresh throttle duration" json:"throttleDuration,omitempty" toml:"throttleDuration,omitempty" yaml:"throttleDuration,omitempty" export:"true"`
	AllowEmptyServices bool             `description:"Allow creation of services without endpoints." json:"allowEmptyServices,omitempty" toml:"allowEmptyServices,omitempty" yaml:"allowEmptyServices,omitempty" export:"true"`
	lastConfiguration  safe.Safe
}

// EndpointIngress holds the endpoint information for the Kubernetes provider.
type EndpointIngress struct {
	IP               string `description:"IP used for Kubernetes Ingress endpoints." json:"ip,omitempty" toml:"ip,omitempty" yaml:"ip,omitempty"`
	Hostname         string `description:"Hostname used for Kubernetes Ingress endpoints." json:"hostname,omitempty" toml:"hostname,omitempty" yaml:"hostname,omitempty"`
	PublishedService string `description:"Published Kubernetes Service to copy status from." json:"publishedService,omitempty" toml:"publishedService,omitempty" yaml:"publishedService,omitempty"`
}

func (p *Provider) newK8sClient(ctx context.Context) (*clientWrapper, error) {
	_, err := labels.Parse(p.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("invalid ingress label selector: %q", p.LabelSelector)
	}

	logger := log.FromContext(ctx)

	logger.Infof("ingress label selector is: %q", p.LabelSelector)

	withEndpoint := ""
	if p.Endpoint != "" {
		withEndpoint = fmt.Sprintf(" with endpoint %v", p.Endpoint)
	}

	var cl *clientWrapper
	switch {
	case os.Getenv("KUBERNETES_SERVICE_HOST") != "" && os.Getenv("KUBERNETES_SERVICE_PORT") != "":
		logger.Infof("Creating in-cluster Provider client%s", withEndpoint)
		cl, err = newInClusterClient(p.Endpoint)
	case os.Getenv("KUBECONFIG") != "":
		logger.Infof("Creating cluster-external Provider client from KUBECONFIG %s", os.Getenv("KUBECONFIG"))
		cl, err = newExternalClusterClientFromFile(os.Getenv("KUBECONFIG"))
	default:
		logger.Infof("Creating cluster-external Provider client%s", withEndpoint)
		cl, err = newExternalClusterClient(p.Endpoint, p.Token, p.CertAuthFilePath)
	}

	if err != nil {
		return nil, err
	}

	cl.ingressLabelSelector = p.LabelSelector
	return cl, nil
}

// Init the provider.
func (p *Provider) Init() error {
	return nil
}

// Provide allows the k8s provider to provide configurations to traefik
// using the given configuration channel.
func (p *Provider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	ctxLog := log.With(context.Background(), log.Str(log.ProviderName, "kubernetes"))
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
					// Note that event is the *first* event that came in during this
					// throttling interval -- if we're hitting our throttle, we may have
					// dropped events. This is fine, because we don't treat different
					// event types differently. But if we do in the future, we'll need to
					// track more information about the dropped events.
					conf := p.loadConfigurationFromIngresses(ctxLog, k8sClient)

					confHash, err := hashstructure.Hash(conf, nil)
					switch {
					case err != nil:
						logger.Error("Unable to hash the configuration")
					case p.lastConfiguration.Get() == confHash:
						logger.Debugf("Skipping Kubernetes event kind %T", event)
					default:
						p.lastConfiguration.Set(confHash)
						configurationChan <- dynamic.Message{
							ProviderName:  "kubernetes",
							Configuration: conf,
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
			logger.Errorf("Provider connection error: %s; retrying in %s", err, time)
		}

		err := backoff.RetryNotify(safe.OperationWithRecover(operation), backoff.WithContext(job.NewBackOff(backoff.NewExponentialBackOff()), ctxPool), notify)
		if err != nil {
			logger.Errorf("Cannot connect to Provider: %s", err)
		}
	})

	return nil
}

func (p *Provider) loadConfigurationFromIngresses(ctx context.Context, client Client) *dynamic.Configuration {
	conf := &dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Routers:     map[string]*dynamic.Router{},
			Middlewares: map[string]*dynamic.Middleware{},
			Services:    map[string]*dynamic.Service{},
		},
		TCP: &dynamic.TCPConfiguration{},
	}

	serverVersion, err := client.GetServerVersion()
	if err != nil {
		log.FromContext(ctx).Errorf("Failed to get server version: %v", err)
		return conf
	}

	var ingressClasses []*networkingv1.IngressClass

	if supportsIngressClass(serverVersion) {
		ics, err := client.GetIngressClasses()
		if err != nil {
			log.FromContext(ctx).Warnf("Failed to list ingress classes: %v", err)
		}

		if p.IngressClass != "" {
			ingressClasses = filterIngressClassByName(p.IngressClass, ics)
		} else {
			ingressClasses = ics
		}
	}

	ingresses := client.GetIngresses()

	certConfigs := make(map[string]*tls.CertAndStores)
	for _, ingress := range ingresses {
		ctx = log.With(ctx, log.Str("ingress", ingress.Name), log.Str("namespace", ingress.Namespace))

		if !p.shouldProcessIngress(ingress, ingressClasses) {
			continue
		}

		rtConfig, err := parseRouterConfig(ingress.Annotations)
		if err != nil {
			log.FromContext(ctx).Errorf("Failed to parse annotations: %v", err)
			continue
		}

		err = getCertificates(ctx, ingress, client, certConfigs)
		if err != nil {
			log.FromContext(ctx).Errorf("Error configuring TLS: %v", err)
		}

		if len(ingress.Spec.Rules) == 0 && ingress.Spec.DefaultBackend != nil {
			if _, ok := conf.HTTP.Services["default-backend"]; ok {
				log.FromContext(ctx).Error("The default backend already exists.")
				continue
			}

			service, err := loadService(client, ingress.Namespace, *ingress.Spec.DefaultBackend)
			if err != nil {
				log.FromContext(ctx).
					WithField("serviceName", ingress.Spec.DefaultBackend.Service.Name).
					WithField("servicePort", ingress.Spec.DefaultBackend.Service.Port.String()).
					Errorf("Cannot create service: %v", err)
				continue
			}

			if len(service.LoadBalancer.Servers) == 0 && !p.AllowEmptyServices {
				log.FromContext(ctx).
					WithField("serviceName", ingress.Spec.DefaultBackend.Service.Name).
					WithField("servicePort", ingress.Spec.DefaultBackend.Service.Port.String()).
					Errorf("Skipping service: no endpoints found")
				continue
			}

			rt := &dynamic.Router{
				Rule:     "PathPrefix(`/`)",
				Priority: math.MinInt32,
				Service:  "default-backend",
			}

			if rtConfig != nil && rtConfig.Router != nil {
				rt.EntryPoints = rtConfig.Router.EntryPoints
				rt.Middlewares = rtConfig.Router.Middlewares
				rt.TLS = rtConfig.Router.TLS
			}

			conf.HTTP.Routers["default-router"] = rt
			conf.HTTP.Services["default-backend"] = service
		}

		routers := map[string][]*dynamic.Router{}

		for _, rule := range ingress.Spec.Rules {
			if err := p.updateIngressStatus(ingress, client); err != nil {
				log.FromContext(ctx).Errorf("Error while updating ingress status: %v", err)
			}

			if rule.HTTP == nil {
				continue
			}

			for _, pa := range rule.HTTP.Paths {
				service, err := loadService(client, ingress.Namespace, pa.Backend)
				if err != nil {
					log.FromContext(ctx).
						WithField("serviceName", pa.Backend.Service.Name).
						WithField("servicePort", pa.Backend.Service.Port.String()).
						Errorf("Cannot create service: %v", err)
					continue
				}

				if len(service.LoadBalancer.Servers) == 0 && !p.AllowEmptyServices {
					log.FromContext(ctx).
						WithField("serviceName", pa.Backend.Service.Name).
						WithField("servicePort", pa.Backend.Service.Port.String()).
						Errorf("Skipping service: no endpoints found")
					continue
				}

				portString := pa.Backend.Service.Port.Name

				if len(pa.Backend.Service.Port.Name) == 0 {
					portString = fmt.Sprint(pa.Backend.Service.Port.Number)
				}

				serviceName := provider.Normalize(ingress.Namespace + "-" + pa.Backend.Service.Name + "-" + portString)
				conf.HTTP.Services[serviceName] = service

				routerKey := strings.TrimPrefix(provider.Normalize(ingress.Name+"-"+ingress.Namespace+"-"+rule.Host+pa.Path), "-")
				routers[routerKey] = append(routers[routerKey], loadRouter(rule, pa, rtConfig, serviceName))
			}
		}

		for routerKey, conflictingRouters := range routers {
			if len(conflictingRouters) == 1 {
				conf.HTTP.Routers[routerKey] = conflictingRouters[0]
				continue
			}

			log.FromContext(ctx).Debugf("Multiple routers are defined with the same key %q, generating hashes to avoid conflicts", routerKey)

			for _, router := range conflictingRouters {
				key, err := makeRouterKeyWithHash(routerKey, router.Rule)
				if err != nil {
					log.FromContext(ctx).Error(err)
					continue
				}

				conf.HTTP.Routers[key] = router
			}
		}
	}

	certs := getTLSConfig(certConfigs)
	if len(certs) > 0 {
		conf.TLS = &dynamic.TLSConfiguration{
			Certificates: certs,
		}
	}

	return conf
}

func (p *Provider) updateIngressStatus(ing *networkingv1.Ingress, k8sClient Client) error {
	// Only process if an EndpointIngress has been configured.
	if p.IngressEndpoint == nil {
		return nil
	}

	if len(p.IngressEndpoint.PublishedService) == 0 {
		if len(p.IngressEndpoint.IP) == 0 && len(p.IngressEndpoint.Hostname) == 0 {
			return errors.New("publishedService or ip or hostname must be defined")
		}

		return k8sClient.UpdateIngressStatus(ing, []corev1.LoadBalancerIngress{{IP: p.IngressEndpoint.IP, Hostname: p.IngressEndpoint.Hostname}})
	}

	serviceInfo := strings.Split(p.IngressEndpoint.PublishedService, "/")
	if len(serviceInfo) != 2 {
		return fmt.Errorf("invalid publishedService format (expected 'namespace/service' format): %s", p.IngressEndpoint.PublishedService)
	}

	serviceNamespace, serviceName := serviceInfo[0], serviceInfo[1]

	service, exists, err := k8sClient.GetService(serviceNamespace, serviceName)
	if err != nil {
		return fmt.Errorf("cannot get service %s, received error: %w", p.IngressEndpoint.PublishedService, err)
	}

	if exists && service.Status.LoadBalancer.Ingress == nil {
		// service exists, but has no Load Balancer status
		log.Debugf("Skipping updating Ingress %s/%s due to service %s having no status set", ing.Namespace, ing.Name, p.IngressEndpoint.PublishedService)
		return nil
	}

	if !exists {
		return fmt.Errorf("missing service: %s", p.IngressEndpoint.PublishedService)
	}

	return k8sClient.UpdateIngressStatus(ing, service.Status.LoadBalancer.Ingress)
}

func (p *Provider) shouldProcessIngress(ingress *networkingv1.Ingress, ingressClasses []*networkingv1.IngressClass) bool {
	// configuration through the new kubernetes ingressClass
	if ingress.Spec.IngressClassName != nil {
		for _, ic := range ingressClasses {
			if *ingress.Spec.IngressClassName == ic.ObjectMeta.Name {
				return true
			}
		}

		return false
	}

	return p.IngressClass == ingress.Annotations[annotationKubernetesIngressClass] ||
		len(p.IngressClass) == 0 && ingress.Annotations[annotationKubernetesIngressClass] == traefikDefaultIngressClass
}

func buildHostRule(host string) string {
	if strings.HasPrefix(host, "*.") {
		return "HostRegexp(`" + strings.Replace(host, "*.", "{subdomain:[a-zA-Z0-9-]+}.", 1) + "`)"
	}

	return "Host(`" + host + "`)"
}

func getCertificates(ctx context.Context, ingress *networkingv1.Ingress, k8sClient Client, tlsConfigs map[string]*tls.CertAndStores) error {
	for _, t := range ingress.Spec.TLS {
		if t.SecretName == "" {
			log.FromContext(ctx).Debugf("Skipping TLS sub-section: No secret name provided")
			continue
		}

		configKey := ingress.Namespace + "-" + t.SecretName
		if _, tlsExists := tlsConfigs[configKey]; !tlsExists {
			secret, exists, err := k8sClient.GetSecret(ingress.Namespace, t.SecretName)
			if err != nil {
				return fmt.Errorf("failed to fetch secret %s/%s: %w", ingress.Namespace, t.SecretName, err)
			}
			if !exists {
				return fmt.Errorf("secret %s/%s does not exist", ingress.Namespace, t.SecretName)
			}

			cert, key, err := getCertificateBlocks(secret, ingress.Namespace, t.SecretName)
			if err != nil {
				return err
			}

			tlsConfigs[configKey] = &tls.CertAndStores{
				Certificate: tls.Certificate{
					CertFile: tls.FileOrContent(cert),
					KeyFile:  tls.FileOrContent(key),
				},
			}
		}
	}

	return nil
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

func loadService(client Client, namespace string, backend networkingv1.IngressBackend) (*dynamic.Service, error) {
	service, exists, err := client.GetService(namespace, backend.Service.Name)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New("service not found")
	}

	var portName string
	var portSpec corev1.ServicePort
	var match bool
	for _, p := range service.Spec.Ports {
		if backend.Service.Port.Number == p.Port || (backend.Service.Port.Name == p.Name && len(p.Name) > 0) {
			portName = p.Name
			portSpec = p
			match = true
			break
		}
	}

	if !match {
		return nil, errors.New("service port not found")
	}

	svc := &dynamic.Service{
		LoadBalancer: &dynamic.ServersLoadBalancer{
			PassHostHeader: func(v bool) *bool { return &v }(true),
		},
	}

	svcConfig, err := parseServiceConfig(service.Annotations)
	if err != nil {
		return nil, err
	}

	if svcConfig != nil && svcConfig.Service != nil {
		svc.LoadBalancer.Sticky = svcConfig.Service.Sticky

		if svcConfig.Service.PassHostHeader != nil {
			svc.LoadBalancer.PassHostHeader = svcConfig.Service.PassHostHeader
		}

		if svcConfig.Service.ServersTransport != "" {
			svc.LoadBalancer.ServersTransport = svcConfig.Service.ServersTransport
		}
	}

	if service.Spec.Type == corev1.ServiceTypeExternalName {
		protocol := getProtocol(portSpec, portSpec.Name, svcConfig)
		hostPort := net.JoinHostPort(service.Spec.ExternalName, strconv.Itoa(int(portSpec.Port)))

		svc.LoadBalancer.Servers = []dynamic.Server{
			{URL: fmt.Sprintf("%s://%s", protocol, hostPort)},
		}

		return svc, nil
	}

	endpoints, endpointsExists, endpointsErr := client.GetEndpoints(namespace, backend.Service.Name)
	if endpointsErr != nil {
		return nil, endpointsErr
	}

	if !endpointsExists {
		return nil, errors.New("endpoints not found")
	}

	var port int32
	for _, subset := range endpoints.Subsets {
		for _, p := range subset.Ports {
			if portName == p.Name {
				port = p.Port
				break
			}
		}

		if port == 0 {
			continue
		}

		protocol := getProtocol(portSpec, portName, svcConfig)

		for _, addr := range subset.Addresses {
			hostPort := net.JoinHostPort(addr.IP, strconv.Itoa(int(port)))

			svc.LoadBalancer.Servers = append(svc.LoadBalancer.Servers, dynamic.Server{
				URL: fmt.Sprintf("%s://%s", protocol, hostPort),
			})
		}
	}

	return svc, nil
}

func getProtocol(portSpec corev1.ServicePort, portName string, svcConfig *ServiceConfig) string {
	if svcConfig != nil && svcConfig.Service != nil && svcConfig.Service.ServersScheme != "" {
		return svcConfig.Service.ServersScheme
	}

	protocol := "http"
	if portSpec.Port == 443 || strings.HasPrefix(portName, "https") {
		protocol = "https"
	}

	return protocol
}

func makeRouterKeyWithHash(key, rule string) (string, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(rule)); err != nil {
		return "", err
	}

	dupKey := fmt.Sprintf("%s-%.10x", key, h.Sum(nil))

	return dupKey, nil
}

func loadRouter(rule networkingv1.IngressRule, pa networkingv1.HTTPIngressPath, rtConfig *RouterConfig, serviceName string) *dynamic.Router {
	var rules []string
	if len(rule.Host) > 0 {
		rules = []string{buildHostRule(rule.Host)}
	}

	if len(pa.Path) > 0 {
		matcher := defaultPathMatcher

		if pa.PathType == nil || *pa.PathType == "" || *pa.PathType == networkingv1.PathTypeImplementationSpecific {
			if rtConfig != nil && rtConfig.Router != nil && rtConfig.Router.PathMatcher != "" {
				matcher = rtConfig.Router.PathMatcher
			}
		} else if *pa.PathType == networkingv1.PathTypeExact {
			matcher = "Path"
		}

		rules = append(rules, fmt.Sprintf("%s(`%s`)", matcher, pa.Path))
	}

	rt := &dynamic.Router{
		Rule:    strings.Join(rules, " && "),
		Service: serviceName,
	}

	if rtConfig != nil && rtConfig.Router != nil {
		rt.Priority = rtConfig.Router.Priority
		rt.EntryPoints = rtConfig.Router.EntryPoints
		rt.Middlewares = rtConfig.Router.Middlewares

		if rtConfig.Router.TLS != nil {
			rt.TLS = rtConfig.Router.TLS
		}
	}

	return rt
}

func throttleEvents(ctx context.Context, throttleDuration time.Duration, pool *safe.Pool, eventsChan <-chan interface{}) chan interface{} {
	if throttleDuration == 0 {
		return nil
	}

	// Create a buffered channel to hold the pending event (if we're delaying processing the event due to throttling).
	eventsChanBuffered := make(chan interface{}, 1)

	// Run a goroutine that reads events from eventChan and does a
	// non-blocking write to pendingEvent. This guarantees that writing to
	// eventChan will never block, and that pendingEvent will have
	// something in it if there's been an event since we read from that channel.
	pool.GoCtx(func(ctxPool context.Context) {
		for {
			select {
			case <-ctxPool.Done():
				return
			case nextEvent := <-eventsChan:
				select {
				case eventsChanBuffered <- nextEvent:
				default:
					// We already have an event in eventsChanBuffered, so we'll
					// do a refresh as soon as our throttle allows us to. It's fine
					// to drop the event and keep whatever's in the buffer -- we
					// don't do different things for different events.
					log.FromContext(ctx).Debugf("Dropping event kind %T due to throttling", nextEvent)
				}
			}
		}
	})

	return eventsChanBuffered
}
