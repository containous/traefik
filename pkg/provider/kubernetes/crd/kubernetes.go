package crd

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/mitchellh/hashstructure"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/job"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/provider"
	"github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/traefik/traefik/v2/pkg/tls"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	annotationKubernetesIngressClass = "kubernetes.io/ingress.class"
	traefikDefaultIngressClass       = "traefik"
)

const (
	providerName               = "kubernetescrd"
	providerNamespaceSeparator = "@"
)

// Provider holds configurations of the provider.
type Provider struct {
	Endpoint               string          `description:"Kubernetes server endpoint (required for external cluster client)." json:"endpoint,omitempty" toml:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Token                  string          `description:"Kubernetes bearer token (not needed for in-cluster client)." json:"token,omitempty" toml:"token,omitempty" yaml:"token,omitempty"`
	CertAuthFilePath       string          `description:"Kubernetes certificate authority file path (not needed for in-cluster client)." json:"certAuthFilePath,omitempty" toml:"certAuthFilePath,omitempty" yaml:"certAuthFilePath,omitempty"`
	DisablePassHostHeaders bool            `description:"Kubernetes disable PassHost Headers." json:"disablePassHostHeaders,omitempty" toml:"disablePassHostHeaders,omitempty" yaml:"disablePassHostHeaders,omitempty" export:"true"`
	Namespaces             []string        `description:"Kubernetes namespaces." json:"namespaces,omitempty" toml:"namespaces,omitempty" yaml:"namespaces,omitempty" export:"true"`
	AllowCrossNamespace    *bool           `description:"Allow cross namespace resource reference." json:"allowCrossNamespace,omitempty" toml:"allowCrossNamespace,omitempty" yaml:"allowCrossNamespace,omitempty" export:"true"`
	LabelSelector          string          `description:"Kubernetes label selector to use." json:"labelSelector,omitempty" toml:"labelSelector,omitempty" yaml:"labelSelector,omitempty" export:"true"`
	IngressClass           string          `description:"Value of kubernetes.io/ingress.class annotation to watch for." json:"ingressClass,omitempty" toml:"ingressClass,omitempty" yaml:"ingressClass,omitempty" export:"true"`
	ThrottleDuration       ptypes.Duration `description:"Ingress refresh throttle duration" json:"throttleDuration,omitempty" toml:"throttleDuration,omitempty" yaml:"throttleDuration,omitempty" export:"true"`
	lastConfiguration      safe.Safe
}

func (p *Provider) newK8sClient(ctx context.Context) (*clientWrapper, error) {
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

// SetDefaults sets the default values.
func (p *Provider) SetDefaults() {
	p.AllowCrossNamespace = func(b bool) *bool { return &b }(true)
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
		configurationChan <- dynamic.Message{
			ProviderName:       providerName,
			ErrorLoadingConfig: true,
		}
		return err
	}

	if p.AllowCrossNamespace == nil || *p.AllowCrossNamespace {
		logger.Warn("Cross-namespace reference between IngressRoutes and resources is enabled, please ensure that this is expected (see AllowCrossNamespace option)")
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
					conf := p.loadConfigurationFromCRD(ctxLog, k8sClient)

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
			configurationChan <- dynamic.Message{
				ProviderName:       providerName,
				ErrorLoadingConfig: true,
			}
		}
		err := backoff.RetryNotify(safe.OperationWithRecover(operation), backoff.WithContext(job.NewBackOff(backoff.NewExponentialBackOff()), ctxPool), notify)
		if err != nil {
			logger.Errorf("Cannot connect to Provider: %v", err)
			configurationChan <- dynamic.Message{
				ProviderName:       providerName,
				ErrorLoadingConfig: true,
			}
		}
	})

	return nil
}

func (p *Provider) loadConfigurationFromCRD(ctx context.Context, client Client) *dynamic.Configuration {
	tlsConfigs := make(map[string]*tls.CertAndStores)
	conf := &dynamic.Configuration{
		HTTP: p.loadIngressRouteConfiguration(ctx, client, tlsConfigs),
		TCP:  p.loadIngressRouteTCPConfiguration(ctx, client, tlsConfigs),
		UDP:  p.loadIngressRouteUDPConfiguration(ctx, client),
		TLS: &dynamic.TLSConfiguration{
			Certificates: getTLSConfig(tlsConfigs),
			Options:      buildTLSOptions(ctx, client),
			Stores:       buildTLSStores(ctx, client),
		},
	}

	for _, middleware := range client.GetMiddlewares() {
		id := provider.Normalize(makeID(middleware.Namespace, middleware.Name))
		ctxMid := log.With(ctx, log.Str(log.MiddlewareName, id))

		basicAuth, err := createBasicAuthMiddleware(client, middleware.Namespace, middleware.Spec.BasicAuth)
		if err != nil {
			log.FromContext(ctxMid).Errorf("Error while reading basic auth middleware: %v", err)
			continue
		}

		digestAuth, err := createDigestAuthMiddleware(client, middleware.Namespace, middleware.Spec.DigestAuth)
		if err != nil {
			log.FromContext(ctxMid).Errorf("Error while reading digest auth middleware: %v", err)
			continue
		}

		forwardAuth, err := createForwardAuthMiddleware(client, middleware.Namespace, middleware.Spec.ForwardAuth)
		if err != nil {
			log.FromContext(ctxMid).Errorf("Error while reading forward auth middleware: %v", err)
			continue
		}

		errorPage, errorPageService, err := p.createErrorPageMiddleware(client, middleware.Namespace, middleware.Spec.Errors)
		if err != nil {
			log.FromContext(ctxMid).Errorf("Error while reading error page middleware: %v", err)
			continue
		}

		if errorPage != nil && errorPageService != nil {
			serviceName := id + "-errorpage-service"
			errorPage.Service = serviceName
			conf.HTTP.Services[serviceName] = errorPageService
		}

		conf.HTTP.Middlewares[id] = &dynamic.Middleware{
			AddPrefix:         middleware.Spec.AddPrefix,
			StripPrefix:       middleware.Spec.StripPrefix,
			StripPrefixRegex:  middleware.Spec.StripPrefixRegex,
			ReplacePath:       middleware.Spec.ReplacePath,
			ReplacePathRegex:  middleware.Spec.ReplacePathRegex,
			Chain:             createChainMiddleware(ctxMid, middleware.Namespace, middleware.Spec.Chain),
			IPWhiteList:       middleware.Spec.IPWhiteList,
			Headers:           middleware.Spec.Headers,
			Errors:            errorPage,
			RateLimit:         middleware.Spec.RateLimit,
			RedirectRegex:     middleware.Spec.RedirectRegex,
			RedirectScheme:    middleware.Spec.RedirectScheme,
			BasicAuth:         basicAuth,
			DigestAuth:        digestAuth,
			ForwardAuth:       forwardAuth,
			InFlightReq:       middleware.Spec.InFlightReq,
			Buffering:         middleware.Spec.Buffering,
			CircuitBreaker:    middleware.Spec.CircuitBreaker,
			Compress:          middleware.Spec.Compress,
			PassTLSClientCert: middleware.Spec.PassTLSClientCert,
			Retry:             middleware.Spec.Retry,
			ContentType:       middleware.Spec.ContentType,
			Plugin:            middleware.Spec.Plugin,
		}
	}

	cb := configBuilder{client, p.AllowCrossNamespace}
	for _, service := range client.GetTraefikServices() {
		err := cb.buildTraefikService(ctx, service, conf.HTTP.Services)
		if err != nil {
			log.FromContext(ctx).WithField(log.ServiceName, service.Name).
				Errorf("Error while building TraefikService: %v", err)
			continue
		}
	}

	return conf
}

func getServicePort(svc *corev1.Service, port int32) (*corev1.ServicePort, error) {
	if svc == nil {
		return nil, errors.New("service is not defined")
	}

	if port == 0 {
		return nil, errors.New("ingressRoute service port not defined")
	}

	hasValidPort := false
	for _, p := range svc.Spec.Ports {
		if p.Port == port {
			return &p, nil
		}

		if p.Port != 0 {
			hasValidPort = true
		}
	}

	if svc.Spec.Type != corev1.ServiceTypeExternalName {
		return nil, fmt.Errorf("service port not found: %d", port)
	}

	if hasValidPort {
		log.WithoutContext().
			Warning("The port %d from IngressRoute doesn't match with ports defined in the ExternalName service %s/%s.", port, svc.Namespace, svc.Name)
	}

	return &corev1.ServicePort{Port: port}, nil
}

func (p *Provider) createErrorPageMiddleware(client Client, namespace string, errorPage *v1alpha1.ErrorPage) (*dynamic.ErrorPage, *dynamic.Service, error) {
	if errorPage == nil {
		return nil, nil, nil
	}

	errorPageMiddleware := &dynamic.ErrorPage{
		Status: errorPage.Status,
		Query:  errorPage.Query,
	}

	balancerServerHTTP, err := configBuilder{client, p.AllowCrossNamespace}.buildServersLB(namespace, errorPage.Service.LoadBalancerSpec)
	if err != nil {
		return nil, nil, err
	}

	return errorPageMiddleware, balancerServerHTTP, nil
}

func createForwardAuthMiddleware(k8sClient Client, namespace string, auth *v1alpha1.ForwardAuth) (*dynamic.ForwardAuth, error) {
	if auth == nil {
		return nil, nil
	}
	if len(auth.Address) == 0 {
		return nil, fmt.Errorf("forward authentication requires an address")
	}

	forwardAuth := &dynamic.ForwardAuth{
		Address:             auth.Address,
		TrustForwardHeader:  auth.TrustForwardHeader,
		AuthResponseHeaders: auth.AuthResponseHeaders,
	}

	if auth.TLS == nil {
		return forwardAuth, nil
	}

	forwardAuth.TLS = &dynamic.ClientTLS{
		CAOptional:         auth.TLS.CAOptional,
		InsecureSkipVerify: auth.TLS.InsecureSkipVerify,
	}

	if len(auth.TLS.CASecret) > 0 {
		caSecret, err := loadCASecret(namespace, auth.TLS.CASecret, k8sClient)
		if err != nil {
			return nil, fmt.Errorf("failed to load auth ca secret: %w", err)
		}
		forwardAuth.TLS.CA = caSecret
	}

	if len(auth.TLS.CertSecret) > 0 {
		authSecretCert, authSecretKey, err := loadAuthTLSSecret(namespace, auth.TLS.CertSecret, k8sClient)
		if err != nil {
			return nil, fmt.Errorf("failed to load auth secret: %w", err)
		}
		forwardAuth.TLS.Cert = authSecretCert
		forwardAuth.TLS.Key = authSecretKey
	}

	return forwardAuth, nil
}

func loadCASecret(namespace, secretName string, k8sClient Client) (string, error) {
	secret, ok, err := k8sClient.GetSecret(namespace, secretName)
	if err != nil {
		return "", fmt.Errorf("failed to fetch secret '%s/%s': %w", namespace, secretName, err)
	}
	if !ok {
		return "", fmt.Errorf("secret '%s/%s' not found", namespace, secretName)
	}
	if secret == nil {
		return "", fmt.Errorf("data for secret '%s/%s' must not be nil", namespace, secretName)
	}
	if len(secret.Data) != 1 {
		return "", fmt.Errorf("found %d elements for secret '%s/%s', must be single element exactly", len(secret.Data), namespace, secretName)
	}

	for _, v := range secret.Data {
		return string(v), nil
	}
	return "", nil
}

func loadAuthTLSSecret(namespace, secretName string, k8sClient Client) (string, string, error) {
	secret, exists, err := k8sClient.GetSecret(namespace, secretName)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch secret '%s/%s': %w", namespace, secretName, err)
	}
	if !exists {
		return "", "", fmt.Errorf("secret '%s/%s' does not exist", namespace, secretName)
	}
	if secret == nil {
		return "", "", fmt.Errorf("data for secret '%s/%s' must not be nil", namespace, secretName)
	}
	if len(secret.Data) != 2 {
		return "", "", fmt.Errorf("found %d elements for secret '%s/%s', must be two elements exactly", len(secret.Data), namespace, secretName)
	}

	return getCertificateBlocks(secret, namespace, secretName)
}

func createBasicAuthMiddleware(client Client, namespace string, basicAuth *v1alpha1.BasicAuth) (*dynamic.BasicAuth, error) {
	if basicAuth == nil {
		return nil, nil
	}

	credentials, err := getAuthCredentials(client, basicAuth.Secret, namespace)
	if err != nil {
		return nil, err
	}

	return &dynamic.BasicAuth{
		Users:        credentials,
		Realm:        basicAuth.Realm,
		RemoveHeader: basicAuth.RemoveHeader,
		HeaderField:  basicAuth.HeaderField,
	}, nil
}

func createDigestAuthMiddleware(client Client, namespace string, digestAuth *v1alpha1.DigestAuth) (*dynamic.DigestAuth, error) {
	if digestAuth == nil {
		return nil, nil
	}

	credentials, err := getAuthCredentials(client, digestAuth.Secret, namespace)
	if err != nil {
		return nil, err
	}

	return &dynamic.DigestAuth{
		Users:        credentials,
		Realm:        digestAuth.Realm,
		RemoveHeader: digestAuth.RemoveHeader,
		HeaderField:  digestAuth.HeaderField,
	}, nil
}

func getAuthCredentials(k8sClient Client, authSecret, namespace string) ([]string, error) {
	if authSecret == "" {
		return nil, fmt.Errorf("auth secret must be set")
	}

	auth, err := loadAuthCredentials(namespace, authSecret, k8sClient)
	if err != nil {
		return nil, fmt.Errorf("failed to load auth credentials: %w", err)
	}

	return auth, nil
}

func loadAuthCredentials(namespace, secretName string, k8sClient Client) ([]string, error) {
	secret, ok, err := k8sClient.GetSecret(namespace, secretName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch secret '%s/%s': %w", namespace, secretName, err)
	}
	if !ok {
		return nil, fmt.Errorf("secret '%s/%s' not found", namespace, secretName)
	}
	if secret == nil {
		return nil, fmt.Errorf("data for secret '%s/%s' must not be nil", namespace, secretName)
	}
	if len(secret.Data) != 1 {
		return nil, fmt.Errorf("found %d elements for secret '%s/%s', must be single element exactly", len(secret.Data), namespace, secretName)
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
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading secret for %s/%s: %w", namespace, secretName, err)
	}
	if len(credentials) == 0 {
		return nil, fmt.Errorf("secret '%s/%s' does not contain any credentials", namespace, secretName)
	}

	return credentials, nil
}

func createChainMiddleware(ctx context.Context, namespace string, chain *v1alpha1.Chain) *dynamic.Chain {
	if chain == nil {
		return nil
	}

	var mds []string
	for _, mi := range chain.Middlewares {
		if strings.Contains(mi.Name, providerNamespaceSeparator) {
			if len(mi.Namespace) > 0 {
				log.FromContext(ctx).
					Warnf("namespace %q is ignored in cross-provider context", mi.Namespace)
			}
			mds = append(mds, mi.Name)
			continue
		}

		ns := mi.Namespace
		if len(ns) == 0 {
			ns = namespace
		}
		mds = append(mds, makeID(ns, mi.Name))
	}
	return &dynamic.Chain{Middlewares: mds}
}

func buildTLSOptions(ctx context.Context, client Client) map[string]tls.Options {
	tlsOptionsCRD := client.GetTLSOptions()
	var tlsOptions map[string]tls.Options

	if len(tlsOptionsCRD) == 0 {
		return tlsOptions
	}
	tlsOptions = make(map[string]tls.Options)
	var nsDefault []string

	for _, tlsOption := range tlsOptionsCRD {
		logger := log.FromContext(log.With(ctx, log.Str("tlsOption", tlsOption.Name), log.Str("namespace", tlsOption.Namespace)))
		var clientCAs []tls.FileOrContent

		for _, secretName := range tlsOption.Spec.ClientAuth.SecretNames {
			secret, exists, err := client.GetSecret(tlsOption.Namespace, secretName)
			if err != nil {
				logger.Errorf("Failed to fetch secret %s/%s: %v", tlsOption.Namespace, secretName, err)
				continue
			}

			if !exists {
				logger.Warnf("Secret %s/%s does not exist", tlsOption.Namespace, secretName)
				continue
			}

			cert, err := getCABlocks(secret, tlsOption.Namespace, secretName)
			if err != nil {
				logger.Errorf("Failed to extract CA from secret %s/%s: %v", tlsOption.Namespace, secretName, err)
				continue
			}

			clientCAs = append(clientCAs, tls.FileOrContent(cert))
		}

		id := makeID(tlsOption.Namespace, tlsOption.Name)
		// If the name is default, we override the default config.
		if tlsOption.Name == "default" {
			id = tlsOption.Name
			nsDefault = append(nsDefault, tlsOption.Namespace)
		}
		tlsOptions[id] = tls.Options{
			MinVersion:       tlsOption.Spec.MinVersion,
			MaxVersion:       tlsOption.Spec.MaxVersion,
			CipherSuites:     tlsOption.Spec.CipherSuites,
			CurvePreferences: tlsOption.Spec.CurvePreferences,
			ClientAuth: tls.ClientAuth{
				CAFiles:        clientCAs,
				ClientAuthType: tlsOption.Spec.ClientAuth.ClientAuthType,
			},
			SniStrict:                tlsOption.Spec.SniStrict,
			PreferServerCipherSuites: tlsOption.Spec.PreferServerCipherSuites,
		}
	}

	if len(nsDefault) > 1 {
		delete(tlsOptions, "default")
		log.FromContext(ctx).Errorf("Default TLS Options defined in multiple namespaces: %v", nsDefault)
	}

	return tlsOptions
}

func buildTLSStores(ctx context.Context, client Client) map[string]tls.Store {
	tlsStoreCRD := client.GetTLSStores()
	var tlsStores map[string]tls.Store

	if len(tlsStoreCRD) == 0 {
		return tlsStores
	}
	tlsStores = make(map[string]tls.Store)
	var nsDefault []string

	for _, tlsStore := range tlsStoreCRD {
		namespace := tlsStore.Namespace
		secretName := tlsStore.Spec.DefaultCertificate.SecretName
		logger := log.FromContext(log.With(ctx, log.Str("tlsStore", tlsStore.Name), log.Str("namespace", namespace), log.Str("secretName", secretName)))

		secret, exists, err := client.GetSecret(namespace, secretName)
		if err != nil {
			logger.Errorf("Failed to fetch secret %s/%s: %v", namespace, secretName, err)
			continue
		}
		if !exists {
			logger.Errorf("Secret %s/%s does not exist", namespace, secretName)
			continue
		}

		cert, key, err := getCertificateBlocks(secret, namespace, secretName)
		if err != nil {
			logger.Errorf("Could not get certificate blocks: %v", err)
			continue
		}

		id := makeID(tlsStore.Namespace, tlsStore.Name)
		// If the name is default, we override the default config.
		if tlsStore.Name == "default" {
			id = tlsStore.Name
			nsDefault = append(nsDefault, tlsStore.Namespace)
		}
		tlsStores[id] = tls.Store{
			DefaultCertificate: &tls.Certificate{
				CertFile: tls.FileOrContent(cert),
				KeyFile:  tls.FileOrContent(key),
			},
		}
	}

	if len(nsDefault) > 1 {
		delete(tlsStores, "default")
		log.FromContext(ctx).Errorf("Default TLS Stores defined in multiple namespaces: %v", nsDefault)
	}

	return tlsStores
}

func makeServiceKey(rule, ingressName string) (string, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(rule)); err != nil {
		return "", err
	}

	key := fmt.Sprintf("%s-%.10x", ingressName, h.Sum(nil))

	return key, nil
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

func getCABlocks(secret *corev1.Secret, namespace, secretName string) (string, error) {
	tlsCrtData, tlsCrtExists := secret.Data["tls.ca"]
	if !tlsCrtExists {
		return "", fmt.Errorf("the tls.ca entry is missing from secret %s/%s", namespace, secretName)
	}

	cert := string(tlsCrtData)
	if cert == "" {
		return "", fmt.Errorf("the tls.ca entry in secret %s/%s is empty", namespace, secretName)
	}

	return cert, nil
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

func isNamespaceAllowed(allowCrossNamespace *bool, parentNamespace, namespace string) bool {
	// If allowCrossNamespace option is not defined the default behavior is to allow cross namespace references.
	return allowCrossNamespace == nil || *allowCrossNamespace || parentNamespace == namespace
}
