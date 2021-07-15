package ingress

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/traefik/traefik/v2/pkg/log"
	traefikversion "github.com/traefik/traefik/v2/pkg/version"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	resyncPeriod   = 10 * time.Minute
	defaultTimeout = 5 * time.Second
)

type marshaler interface {
	Marshal() ([]byte, error)
}

type resourceEventHandler struct {
	ev chan<- interface{}
}

func (reh *resourceEventHandler) OnAdd(obj interface{}) {
	eventHandlerFunc(reh.ev, obj)
}

func (reh *resourceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	eventHandlerFunc(reh.ev, newObj)
}

func (reh *resourceEventHandler) OnDelete(obj interface{}) {
	eventHandlerFunc(reh.ev, obj)
}

// Client is a client for the Provider master.
// WatchAll starts the watch of the Provider resources and updates the stores.
// The stores can then be accessed via the Get* functions.
type Client interface {
	WatchAll(namespaces []string, stopCh <-chan struct{}) (<-chan interface{}, error)
	GetIngresses() []*networkingv1beta1.Ingress
	GetIngressClasses() ([]*networkingv1beta1.IngressClass, error)
	GetService(namespace, name string) (*corev1.Service, bool, error)
	GetSecret(namespace, name string) (*corev1.Secret, bool, error)
	GetEndpoints(namespace, name string) (*corev1.Endpoints, bool, error)
	UpdateIngressStatus(ing *networkingv1beta1.Ingress, ingStatus []corev1.LoadBalancerIngress) error
	GetServerVersion() *version.Version
}

type clientWrapper struct {
	clientset            kubernetes.Interface
	factoriesKube        map[string]informers.SharedInformerFactory
	factoriesSecret      map[string]informers.SharedInformerFactory
	factoriesIngress     map[string]informers.SharedInformerFactory
	clusterFactory       informers.SharedInformerFactory
	ingressLabelSelector string
	isNamespaceAll       bool
	watchedNamespaces    []string
	serverVersion        *version.Version
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

func createClientFromConfig(c *rest.Config) (*clientWrapper, error) {
	c.UserAgent = fmt.Sprintf(
		"%s/%s (%s/%s) kubernetes/ingress",
		filepath.Base(os.Args[0]),
		traefikversion.Version,
		runtime.GOOS,
		runtime.GOARCH,
	)

	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	return newClientImpl(clientset), nil
}

func newClientImpl(clientset kubernetes.Interface) *clientWrapper {
	return &clientWrapper{
		clientset:        clientset,
		factoriesSecret:  make(map[string]informers.SharedInformerFactory),
		factoriesIngress: make(map[string]informers.SharedInformerFactory),
		factoriesKube:    make(map[string]informers.SharedInformerFactory),
	}
}

// WatchAll starts namespace-specific controllers for all relevant kinds.
func (c *clientWrapper) WatchAll(namespaces []string, stopCh <-chan struct{}) (<-chan interface{}, error) {
	// Get and store the serverVersion for future use.
	serverVersionInfo, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve server version: %w", err)
	}

	serverVersion, err := version.NewVersion(serverVersionInfo.GitVersion)
	if err != nil {
		return nil, fmt.Errorf("could not parse server version: %w", err)
	}

	c.serverVersion = serverVersion

	eventCh := make(chan interface{}, 1)
	eventHandler := &resourceEventHandler{eventCh}

	if len(namespaces) == 0 {
		namespaces = []string{metav1.NamespaceAll}
		c.isNamespaceAll = true
	}

	c.watchedNamespaces = namespaces

	notOwnedByHelm := func(opts *metav1.ListOptions) {
		opts.LabelSelector = "owner!=helm"
	}

	matchesLabelSelector := func(opts *metav1.ListOptions) {
		opts.LabelSelector = c.ingressLabelSelector
	}

	for _, ns := range namespaces {
		factoryIngress := informers.NewSharedInformerFactoryWithOptions(c.clientset, resyncPeriod, informers.WithNamespace(ns), informers.WithTweakListOptions(matchesLabelSelector))
		factoryIngress.Extensions().V1beta1().Ingresses().Informer().AddEventHandler(eventHandler)
		c.factoriesIngress[ns] = factoryIngress

		factoryKube := informers.NewSharedInformerFactoryWithOptions(c.clientset, resyncPeriod, informers.WithNamespace(ns))
		factoryKube.Core().V1().Services().Informer().AddEventHandler(eventHandler)
		factoryKube.Core().V1().Endpoints().Informer().AddEventHandler(eventHandler)
		c.factoriesKube[ns] = factoryKube

		factorySecret := informers.NewSharedInformerFactoryWithOptions(c.clientset, resyncPeriod, informers.WithNamespace(ns), informers.WithTweakListOptions(notOwnedByHelm))
		factorySecret.Core().V1().Secrets().Informer().AddEventHandler(eventHandler)
		c.factoriesSecret[ns] = factorySecret
	}

	for _, ns := range namespaces {
		c.factoriesIngress[ns].Start(stopCh)
		c.factoriesKube[ns].Start(stopCh)
		c.factoriesSecret[ns].Start(stopCh)
	}

	for _, ns := range namespaces {
		for typ, ok := range c.factoriesIngress[ns].WaitForCacheSync(stopCh) {
			if !ok {
				return nil, fmt.Errorf("timed out waiting for controller caches to sync %s in namespace %q", typ, ns)
			}
		}

		for typ, ok := range c.factoriesKube[ns].WaitForCacheSync(stopCh) {
			if !ok {
				return nil, fmt.Errorf("timed out waiting for controller caches to sync %s in namespace %q", typ, ns)
			}
		}

		for typ, ok := range c.factoriesSecret[ns].WaitForCacheSync(stopCh) {
			if !ok {
				return nil, fmt.Errorf("timed out waiting for controller caches to sync %s in namespace %q", typ, ns)
			}
		}
	}

	if supportsIngressClass(serverVersion) {
		c.clusterFactory = informers.NewSharedInformerFactoryWithOptions(c.clientset, resyncPeriod)
		c.clusterFactory.Networking().V1beta1().IngressClasses().Informer().AddEventHandler(eventHandler)
		c.clusterFactory.Start(stopCh)

		for typ, ok := range c.clusterFactory.WaitForCacheSync(stopCh) {
			if !ok {
				return nil, fmt.Errorf("timed out waiting for controller caches to sync %s", typ)
			}
		}
	}

	return eventCh, nil
}

// GetIngresses returns all Ingresses for observed namespaces in the cluster.
func (c *clientWrapper) GetIngresses() []*networkingv1beta1.Ingress {
	var results []*networkingv1beta1.Ingress

	for ns, factory := range c.factoriesIngress {
		// extensions
		ings, err := factory.Extensions().V1beta1().Ingresses().Lister().List(labels.Everything())
		if err != nil {
			log.Errorf("Failed to list ingresses in namespace %s: %v", ns, err)
		}

		for _, ing := range ings {
			n, err := extensionsToNetworking(ing)
			if err != nil {
				log.Errorf("Failed to convert ingress %s from extensions/v1beta1 to networking/v1beta1: %v", ns, err)
				continue
			}
			results = append(results, n)
		}

		// networking
		list, err := factory.Networking().V1beta1().Ingresses().Lister().List(labels.Everything())
		if err != nil {
			log.Errorf("Failed to list ingresses in namespace %s: %v", ns, err)
		}
		results = append(results, list...)
	}
	return results
}

func extensionsToNetworking(ing marshaler) (*networkingv1beta1.Ingress, error) {
	data, err := ing.Marshal()
	if err != nil {
		return nil, err
	}

	ni := &networkingv1beta1.Ingress{}
	err = ni.Unmarshal(data)
	if err != nil {
		return nil, err
	}

	return ni, nil
}

// UpdateIngressStatus updates an Ingress with a provided status.
func (c *clientWrapper) UpdateIngressStatus(src *networkingv1beta1.Ingress, ingStatus []corev1.LoadBalancerIngress) error {
	if !c.isWatchedNamespace(src.Namespace) {
		return fmt.Errorf("failed to get ingress %s/%s: namespace is not within watched namespaces", src.Namespace, src.Name)
	}

	if src.GetObjectKind().GroupVersionKind().Group != "networking.k8s.io" {
		return c.updateIngressStatusOld(src, ingStatus)
	}

	ing, err := c.factoriesIngress[c.lookupNamespace(src.Namespace)].Networking().V1beta1().Ingresses().Lister().Ingresses(src.Namespace).Get(src.Name)
	if err != nil {
		return fmt.Errorf("failed to get ingress %s/%s: %w", src.Namespace, src.Name, err)
	}

	logger := log.WithoutContext().WithField("namespace", ing.Namespace).WithField("ingress", ing.Name)

	if isLoadBalancerIngressEquals(ing.Status.LoadBalancer.Ingress, ingStatus) {
		logger.Debug("Skipping ingress status update")
		return nil
	}

	ingCopy := ing.DeepCopy()
	ingCopy.Status = networkingv1beta1.IngressStatus{LoadBalancer: corev1.LoadBalancerStatus{Ingress: ingStatus}}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	_, err = c.clientset.NetworkingV1beta1().Ingresses(ingCopy.Namespace).UpdateStatus(ctx, ingCopy, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ingress status %s/%s: %w", src.Namespace, src.Name, err)
	}

	logger.Info("Updated ingress status")
	return nil
}

func (c *clientWrapper) updateIngressStatusOld(src *networkingv1beta1.Ingress, ingStatus []corev1.LoadBalancerIngress) error {
	ing, err := c.factoriesIngress[c.lookupNamespace(src.Namespace)].Extensions().V1beta1().Ingresses().Lister().Ingresses(src.Namespace).Get(src.Name)
	if err != nil {
		return fmt.Errorf("failed to get ingress %s/%s: %w", src.Namespace, src.Name, err)
	}

	logger := log.WithoutContext().WithField("namespace", ing.Namespace).WithField("ingress", ing.Name)

	if isLoadBalancerIngressEquals(ing.Status.LoadBalancer.Ingress, ingStatus) {
		logger.Debug("Skipping ingress status update")
		return nil
	}

	ingCopy := ing.DeepCopy()
	ingCopy.Status = extensionsv1beta1.IngressStatus{LoadBalancer: corev1.LoadBalancerStatus{Ingress: ingStatus}}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	_, err = c.clientset.ExtensionsV1beta1().Ingresses(ingCopy.Namespace).UpdateStatus(ctx, ingCopy, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ingress status %s/%s: %w", src.Namespace, src.Name, err)
	}

	logger.Info("Updated ingress status")
	return nil
}

// isLoadBalancerIngressEquals returns true if the given slices are equal, false otherwise.
func isLoadBalancerIngressEquals(aSlice, bSlice []corev1.LoadBalancerIngress) bool {
	if len(aSlice) != len(bSlice) {
		return false
	}

	aMap := make(map[string]struct{})
	for _, aIngress := range aSlice {
		aMap[aIngress.Hostname+aIngress.IP] = struct{}{}
	}

	for _, bIngress := range bSlice {
		if _, exists := aMap[bIngress.Hostname+bIngress.IP]; !exists {
			return false
		}
	}

	return true
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

func (c *clientWrapper) GetIngressClasses() ([]*networkingv1beta1.IngressClass, error) {
	if c.clusterFactory == nil {
		return nil, errors.New("cluster factory not loaded")
	}

	ingressClasses, err := c.clusterFactory.Networking().V1beta1().IngressClasses().Lister().List(labels.Everything())
	if err != nil {
		return nil, err
	}

	var ics []*networkingv1beta1.IngressClass
	for _, ic := range ingressClasses {
		if ic.Spec.Controller == traefikDefaultIngressClassController {
			ics = append(ics, ic)
		}
	}

	return ics, nil
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

// GetServerVersion returns the cluster server version, or an error.
func (c *clientWrapper) GetServerVersion() *version.Version {
	return c.serverVersion
}

// eventHandlerFunc will pass the obj on to the events channel or drop it.
// This is so passing the events along won't block in the case of high volume.
// The events are only used for signaling anyway so dropping a few is ok.
func eventHandlerFunc(events chan<- interface{}, obj interface{}) {
	select {
	case events <- obj:
	default:
	}
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

// IngressClass objects are supported since Kubernetes v1.18.
// See https://kubernetes.io/docs/concepts/services-networking/ingress/#ingress-class
func supportsIngressClass(serverVersion *version.Version) bool {
	ingressClassVersion := version.Must(version.NewVersion("1.18"))

	return ingressClassVersion.LessThanOrEqual(serverVersion)
}
