package crd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/generated/clientset/versioned"
	"github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/generated/informers/externalversions"
	"github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	"github.com/traefik/traefik/v2/pkg/provider/kubernetes/k8s"
	"github.com/traefik/traefik/v2/pkg/version"
	corev1 "k8s.io/api/core/v1"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const resyncPeriod = 10 * time.Minute

// Client is a client for the Provider master.
// WatchAll starts the watch of the Provider resources and updates the stores.
// The stores can then be accessed via the Get* functions.
type Client interface {
	WatchAll(namespaces []string, stopCh <-chan struct{}) (<-chan interface{}, error)

	GetIngressRoutes() []*v1alpha1.IngressRoute
	GetIngressRouteTCPs() []*v1alpha1.IngressRouteTCP
	GetIngressRouteUDPs() []*v1alpha1.IngressRouteUDP
	GetMiddlewares() []*v1alpha1.Middleware
	GetMiddlewareTCPs() []*v1alpha1.MiddlewareTCP
	GetTraefikService(namespace, name string) (*v1alpha1.TraefikService, bool, error)
	GetTraefikServices() []*v1alpha1.TraefikService
	GetTLSOptions() []*v1alpha1.TLSOption
	GetServersTransports() []*v1alpha1.ServersTransport
	GetTLSStores() []*v1alpha1.TLSStore

	GetService(namespace, name string) (*corev1.Service, bool, error)
	GetSecret(namespace, name string) (*corev1.Secret, bool, error)
	GetEndpoints(namespace, name string) (*corev1.Endpoints, bool, error)
}

// TODO: add tests for the clientWrapper (and its methods) itself.
type clientWrapper struct {
	csCrd  versioned.Interface
	csKube kubernetes.Interface

	factoriesCrd    map[string]externalversions.SharedInformerFactory
	factoriesKube   map[string]informers.SharedInformerFactory
	factoriesSecret map[string]informers.SharedInformerFactory

	labelSelector string

	isNamespaceAll    bool
	watchedNamespaces []string
}

func createClientFromConfig(c *rest.Config) (*clientWrapper, error) {
	c.UserAgent = fmt.Sprintf(
		"%s/%s (%s/%s) kubernetes/crd",
		filepath.Base(os.Args[0]),
		version.Version,
		runtime.GOOS,
		runtime.GOARCH,
	)

	csCrd, err := versioned.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	csKube, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	return newClientImpl(csKube, csCrd), nil
}

func newClientImpl(csKube kubernetes.Interface, csCrd versioned.Interface) *clientWrapper {
	return &clientWrapper{
		csCrd:           csCrd,
		csKube:          csKube,
		factoriesCrd:    make(map[string]externalversions.SharedInformerFactory),
		factoriesKube:   make(map[string]informers.SharedInformerFactory),
		factoriesSecret: make(map[string]informers.SharedInformerFactory),
	}
}

// newInClusterClient returns a new Provider client that is expected to run
// inside the cluster.
func newInClusterClient(endpoint string) (*clientWrapper, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster configuration: %w", err)
	}

	if endpoint != "" {
		config.Host = endpoint
	}

	return createClientFromConfig(config)
}

func newExternalClusterClientFromFile(file string) (*clientWrapper, error) {
	configFromFlags, err := clientcmd.BuildConfigFromFlags("", file)
	if err != nil {
		return nil, err
	}
	return createClientFromConfig(configFromFlags)
}

// newExternalClusterClient returns a new Provider client that may run outside
// of the cluster.
// The endpoint parameter must not be empty.
func newExternalClusterClient(endpoint, token, caFilePath string) (*clientWrapper, error) {
	if endpoint == "" {
		return nil, errors.New("endpoint missing for external cluster client")
	}

	config := &rest.Config{
		Host:        endpoint,
		BearerToken: token,
	}

	if caFilePath != "" {
		caData, err := os.ReadFile(caFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file %s: %w", caFilePath, err)
		}

		config.TLSClientConfig = rest.TLSClientConfig{CAData: caData}
	}

	return createClientFromConfig(config)
}

// WatchAll starts namespace-specific controllers for all relevant kinds.
func (c *clientWrapper) WatchAll(namespaces []string, stopCh <-chan struct{}) (<-chan interface{}, error) {
	eventCh := make(chan interface{}, 1)
	eventHandler := &k8s.ResourceEventHandler{Ev: eventCh}

	if len(namespaces) == 0 {
		namespaces = []string{metav1.NamespaceAll}
		c.isNamespaceAll = true
	}

	c.watchedNamespaces = namespaces

	notOwnedByHelm := func(opts *metav1.ListOptions) {
		opts.LabelSelector = "owner!=helm"
	}

	matchesLabelSelector := func(opts *metav1.ListOptions) {
		opts.LabelSelector = c.labelSelector
	}

	for _, ns := range namespaces {
		factoryCrd := externalversions.NewSharedInformerFactoryWithOptions(c.csCrd, resyncPeriod, externalversions.WithNamespace(ns), externalversions.WithTweakListOptions(matchesLabelSelector))
		factoryCrd.Traefik().V1alpha1().IngressRoutes().Informer().AddEventHandler(eventHandler)
		factoryCrd.Traefik().V1alpha1().Middlewares().Informer().AddEventHandler(eventHandler)
		factoryCrd.Traefik().V1alpha1().MiddlewareTCPs().Informer().AddEventHandler(eventHandler)
		factoryCrd.Traefik().V1alpha1().IngressRouteTCPs().Informer().AddEventHandler(eventHandler)
		factoryCrd.Traefik().V1alpha1().IngressRouteUDPs().Informer().AddEventHandler(eventHandler)
		factoryCrd.Traefik().V1alpha1().TLSOptions().Informer().AddEventHandler(eventHandler)
		factoryCrd.Traefik().V1alpha1().ServersTransports().Informer().AddEventHandler(eventHandler)
		factoryCrd.Traefik().V1alpha1().TLSStores().Informer().AddEventHandler(eventHandler)
		factoryCrd.Traefik().V1alpha1().TraefikServices().Informer().AddEventHandler(eventHandler)

		factoryKube := informers.NewSharedInformerFactoryWithOptions(c.csKube, resyncPeriod, informers.WithNamespace(ns))
		factoryKube.Core().V1().Services().Informer().AddEventHandler(eventHandler)
		factoryKube.Core().V1().Endpoints().Informer().AddEventHandler(eventHandler)

		factorySecret := informers.NewSharedInformerFactoryWithOptions(c.csKube, resyncPeriod, informers.WithNamespace(ns), informers.WithTweakListOptions(notOwnedByHelm))
		factorySecret.Core().V1().Secrets().Informer().AddEventHandler(eventHandler)

		c.factoriesCrd[ns] = factoryCrd
		c.factoriesKube[ns] = factoryKube
		c.factoriesSecret[ns] = factorySecret
	}

	for _, ns := range namespaces {
		c.factoriesCrd[ns].Start(stopCh)
		c.factoriesKube[ns].Start(stopCh)
		c.factoriesSecret[ns].Start(stopCh)
	}

	for _, ns := range namespaces {
		for t, ok := range c.factoriesCrd[ns].WaitForCacheSync(stopCh) {
			if !ok {
				return nil, fmt.Errorf("timed out waiting for controller caches to sync %s in namespace %q", t.String(), ns)
			}
		}

		for t, ok := range c.factoriesKube[ns].WaitForCacheSync(stopCh) {
			if !ok {
				return nil, fmt.Errorf("timed out waiting for controller caches to sync %s in namespace %q", t.String(), ns)
			}
		}

		for t, ok := range c.factoriesSecret[ns].WaitForCacheSync(stopCh) {
			if !ok {
				return nil, fmt.Errorf("timed out waiting for controller caches to sync %s in namespace %q", t.String(), ns)
			}
		}
	}

	return eventCh, nil
}

func (c *clientWrapper) GetIngressRoutes() []*v1alpha1.IngressRoute {
	var result []*v1alpha1.IngressRoute

	for ns, factory := range c.factoriesCrd {
		ings, err := factory.Traefik().V1alpha1().IngressRoutes().Lister().List(labels.Everything())
		if err != nil {
			log.Errorf("Failed to list ingress routes in namespace %s: %v", ns, err)
		}
		result = append(result, ings...)
	}

	return result
}

func (c *clientWrapper) GetIngressRouteTCPs() []*v1alpha1.IngressRouteTCP {
	var result []*v1alpha1.IngressRouteTCP

	for ns, factory := range c.factoriesCrd {
		ings, err := factory.Traefik().V1alpha1().IngressRouteTCPs().Lister().List(labels.Everything())
		if err != nil {
			log.Errorf("Failed to list tcp ingress routes in namespace %s: %v", ns, err)
		}
		result = append(result, ings...)
	}

	return result
}

func (c *clientWrapper) GetIngressRouteUDPs() []*v1alpha1.IngressRouteUDP {
	var result []*v1alpha1.IngressRouteUDP

	for ns, factory := range c.factoriesCrd {
		ings, err := factory.Traefik().V1alpha1().IngressRouteUDPs().Lister().List(labels.Everything())
		if err != nil {
			log.Errorf("Failed to list udp ingress routes in namespace %s: %v", ns, err)
		}
		result = append(result, ings...)
	}

	return result
}

func (c *clientWrapper) GetMiddlewares() []*v1alpha1.Middleware {
	var result []*v1alpha1.Middleware

	for ns, factory := range c.factoriesCrd {
		middlewares, err := factory.Traefik().V1alpha1().Middlewares().Lister().List(labels.Everything())
		if err != nil {
			log.Errorf("Failed to list middlewares in namespace %s: %v", ns, err)
		}
		result = append(result, middlewares...)
	}

	return result
}

func (c *clientWrapper) GetMiddlewareTCPs() []*v1alpha1.MiddlewareTCP {
	var result []*v1alpha1.MiddlewareTCP

	for ns, factory := range c.factoriesCrd {
		middlewares, err := factory.Traefik().V1alpha1().MiddlewareTCPs().Lister().List(labels.Everything())
		if err != nil {
			log.Errorf("Failed to list TCP middlewares in namespace %s: %v", ns, err)
		}
		result = append(result, middlewares...)
	}

	return result
}

// GetTraefikService returns the named service from the given namespace.
func (c *clientWrapper) GetTraefikService(namespace, name string) (*v1alpha1.TraefikService, bool, error) {
	if !c.isWatchedNamespace(namespace) {
		return nil, false, fmt.Errorf("failed to get service %s/%s: namespace is not within watched namespaces", namespace, name)
	}

	service, err := c.factoriesCrd[c.lookupNamespace(namespace)].Traefik().V1alpha1().TraefikServices().Lister().TraefikServices(namespace).Get(name)
	exist, err := translateNotFoundError(err)

	return service, exist, err
}

func (c *clientWrapper) GetTraefikServices() []*v1alpha1.TraefikService {
	var result []*v1alpha1.TraefikService

	for ns, factory := range c.factoriesCrd {
		ings, err := factory.Traefik().V1alpha1().TraefikServices().Lister().List(labels.Everything())
		if err != nil {
			log.Errorf("Failed to list Traefik services in namespace %s: %v", ns, err)
		}
		result = append(result, ings...)
	}

	return result
}

// GetServersTransport returns all ServersTransport.
func (c *clientWrapper) GetServersTransports() []*v1alpha1.ServersTransport {
	var result []*v1alpha1.ServersTransport

	for ns, factory := range c.factoriesCrd {
		serversTransports, err := factory.Traefik().V1alpha1().ServersTransports().Lister().List(labels.Everything())
		if err != nil {
			log.Errorf("Failed to list servers transport in namespace %s: %v", ns, err)
		}
		result = append(result, serversTransports...)
	}

	return result
}

// GetTLSOptions returns all TLS options.
func (c *clientWrapper) GetTLSOptions() []*v1alpha1.TLSOption {
	var result []*v1alpha1.TLSOption

	for ns, factory := range c.factoriesCrd {
		options, err := factory.Traefik().V1alpha1().TLSOptions().Lister().List(labels.Everything())
		if err != nil {
			log.Errorf("Failed to list tls options in namespace %s: %v", ns, err)
		}
		result = append(result, options...)
	}

	return result
}

// GetTLSStores returns all TLS stores.
func (c *clientWrapper) GetTLSStores() []*v1alpha1.TLSStore {
	var result []*v1alpha1.TLSStore

	for ns, factory := range c.factoriesCrd {
		stores, err := factory.Traefik().V1alpha1().TLSStores().Lister().List(labels.Everything())
		if err != nil {
			log.Errorf("Failed to list tls stores in namespace %s: %v", ns, err)
		}
		result = append(result, stores...)
	}

	return result
}

// GetService returns the named service from the given namespace.
func (c *clientWrapper) GetService(namespace, name string) (*corev1.Service, bool, error) {
	if !c.isWatchedNamespace(namespace) {
		return nil, false, fmt.Errorf("failed to get service %s/%s: namespace is not within watched namespaces", namespace, name)
	}

	service, err := c.factoriesKube[c.lookupNamespace(namespace)].Core().V1().Services().Lister().Services(namespace).Get(name)
	exist, err := translateNotFoundError(err)
	return service, exist, err
}

// GetEndpoints returns the named endpoints from the given namespace.
func (c *clientWrapper) GetEndpoints(namespace, name string) (*corev1.Endpoints, bool, error) {
	if !c.isWatchedNamespace(namespace) {
		return nil, false, fmt.Errorf("failed to get endpoints %s/%s: namespace is not within watched namespaces", namespace, name)
	}

	endpoint, err := c.factoriesKube[c.lookupNamespace(namespace)].Core().V1().Endpoints().Lister().Endpoints(namespace).Get(name)
	exist, err := translateNotFoundError(err)
	return endpoint, exist, err
}

// GetSecret returns the named secret from the given namespace.
func (c *clientWrapper) GetSecret(namespace, name string) (*corev1.Secret, bool, error) {
	if !c.isWatchedNamespace(namespace) {
		return nil, false, fmt.Errorf("failed to get secret %s/%s: namespace is not within watched namespaces", namespace, name)
	}

	secret, err := c.factoriesSecret[c.lookupNamespace(namespace)].Core().V1().Secrets().Lister().Secrets(namespace).Get(name)
	exist, err := translateNotFoundError(err)
	return secret, exist, err
}

// lookupNamespace returns the lookup namespace key for the given namespace.
// When listening on all namespaces, it returns the client-go identifier ("")
// for all-namespaces. Otherwise, it returns the given namespace.
// The distinction is necessary because we index all informers on the special
// identifier iff all-namespaces are requested but receive specific namespace
// identifiers from the Kubernetes API, so we have to bridge this gap.
func (c *clientWrapper) lookupNamespace(ns string) string {
	if c.isNamespaceAll {
		return metav1.NamespaceAll
	}
	return ns
}

// translateNotFoundError will translate a "not found" error to a boolean return
// value which indicates if the resource exists and a nil error.
func translateNotFoundError(err error) (bool, error) {
	if kubeerror.IsNotFound(err) {
		return false, nil
	}
	return err == nil, err
}

// isWatchedNamespace checks to ensure that the namespace is being watched before we request
// it to ensure we don't panic by requesting an out-of-watch object.
func (c *clientWrapper) isWatchedNamespace(ns string) bool {
	if c.isNamespaceAll {
		return true
	}
	for _, watchedNamespace := range c.watchedNamespaces {
		if watchedNamespace == ns {
			return true
		}
	}
	return false
}
