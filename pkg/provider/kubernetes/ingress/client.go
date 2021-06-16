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
	"github.com/traefik/traefik/v2/pkg/provider/kubernetes/k8s"
	traefikversion "github.com/traefik/traefik/v2/pkg/version"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
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

// Client is a client for the Provider master.
// WatchAll starts the watch of the Provider resources and updates the stores.
// The stores can then be accessed via the Get* functions.
type Client interface {
	WatchAll(namespaces []string, stopCh <-chan struct{}) (<-chan interface{}, error)
	GetIngresses() []*networkingv1.Ingress
	GetIngressClasses() ([]*networkingv1.IngressClass, error)
	GetService(namespace, name string) (*corev1.Service, bool, error)
	GetSecret(namespace, name string) (*corev1.Secret, bool, error)
	GetEndpoints(namespace, name string) (*corev1.Endpoints, bool, error)
	UpdateIngressStatus(ing *networkingv1.Ingress, ingStatus []corev1.LoadBalancerIngress) error
	GetServerVersion() (*version.Version, error)
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
		opts.LabelSelector = c.ingressLabelSelector
	}

	serverVersion, err := c.GetServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get server version: %w", err)
	}

	for _, ns := range namespaces {
		factoryIngress := informers.NewSharedInformerFactoryWithOptions(c.clientset, resyncPeriod, informers.WithNamespace(ns), informers.WithTweakListOptions(matchesLabelSelector))

		if supportsNetworkingV1Ingress(serverVersion) {
			factoryIngress.Networking().V1().Ingresses().Informer().AddEventHandler(eventHandler)
		} else {
			factoryIngress.Networking().V1beta1().Ingresses().Informer().AddEventHandler(eventHandler)
		}

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

		if supportsNetworkingV1Ingress(serverVersion) {
			c.clusterFactory.Networking().V1().IngressClasses().Informer().AddEventHandler(eventHandler)
		} else {
			c.clusterFactory.Networking().V1beta1().IngressClasses().Informer().AddEventHandler(eventHandler)
		}

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
func (c *clientWrapper) GetIngresses() []*networkingv1.Ingress {
	var results []*networkingv1.Ingress

	serverVersion, err := c.GetServerVersion()
	if err != nil {
		log.Errorf("Failed to get server version: %v", err)
		return results
	}

	isNetworkingV1Supported := supportsNetworkingV1Ingress(serverVersion)

	for ns, factory := range c.factoriesIngress {
		if isNetworkingV1Supported {
			// networking
			listNew, err := factory.Networking().V1().Ingresses().Lister().List(labels.Everything())
			if err != nil {
				log.WithoutContext().Errorf("Failed to list ingresses in namespace %s: %v", ns, err)
				continue
			}

			results = append(results, listNew...)
			continue
		}

		// networking beta
		list, err := factory.Networking().V1beta1().Ingresses().Lister().List(labels.Everything())
		if err != nil {
			log.WithoutContext().Errorf("Failed to list ingresses in namespace %s: %v", ns, err)
			continue
		}

		for _, ing := range list {
			n, err := toNetworkingV1(ing)
			if err != nil {
				log.WithoutContext().Errorf("Failed to convert ingress %s from networking/v1beta1 to networking/v1: %v", ns, err)
				continue
			}

			addServiceFromV1Beta1(n, *ing)

			results = append(results, n)
		}
	}
	return results
}

func toNetworkingV1(ing marshaler) (*networkingv1.Ingress, error) {
	data, err := ing.Marshal()
	if err != nil {
		return nil, err
	}

	ni := &networkingv1.Ingress{}
	err = ni.Unmarshal(data)
	if err != nil {
		return nil, err
	}

	return ni, nil
}

func toNetworkingV1IngressClass(ing marshaler) (*networkingv1.IngressClass, error) {
	data, err := ing.Marshal()
	if err != nil {
		return nil, err
	}

	ni := &networkingv1.IngressClass{}
	err = ni.Unmarshal(data)
	if err != nil {
		return nil, err
	}

	return ni, nil
}

func addServiceFromV1Beta1(ing *networkingv1.Ingress, old networkingv1beta1.Ingress) {
	if old.Spec.Backend != nil {
		port := networkingv1.ServiceBackendPort{}
		if old.Spec.Backend.ServicePort.Type == intstr.Int {
			port.Number = old.Spec.Backend.ServicePort.IntVal
		} else {
			port.Name = old.Spec.Backend.ServicePort.StrVal
		}

		if old.Spec.Backend.ServiceName != "" {
			ing.Spec.DefaultBackend = &networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: old.Spec.Backend.ServiceName,
					Port: port,
				},
			}
		}
	}

	for rc, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for pc, path := range rule.HTTP.Paths {
			if path.Backend.Service == nil {
				oldBackend := old.Spec.Rules[rc].HTTP.Paths[pc].Backend

				port := networkingv1.ServiceBackendPort{}
				if oldBackend.ServicePort.Type == intstr.Int {
					port.Number = oldBackend.ServicePort.IntVal
				} else {
					port.Name = oldBackend.ServicePort.StrVal
				}

				svc := networkingv1.IngressServiceBackend{
					Name: oldBackend.ServiceName,
					Port: port,
				}

				ing.Spec.Rules[rc].HTTP.Paths[pc].Backend.Service = &svc
			}
		}
	}
}

// UpdateIngressStatus updates an Ingress with a provided status.
func (c *clientWrapper) UpdateIngressStatus(src *networkingv1.Ingress, ingStatus []corev1.LoadBalancerIngress) error {
	if !c.isWatchedNamespace(src.Namespace) {
		return fmt.Errorf("failed to get ingress %s/%s: namespace is not within watched namespaces", src.Namespace, src.Name)
	}

	serverVersion, err := c.GetServerVersion()
	if err != nil {
		log.WithoutContext().Errorf("Failed to get server version: %v", err)
		return err
	}

	if !supportsNetworkingV1Ingress(serverVersion) {
		return c.updateIngressStatusOld(src, ingStatus)
	}

	ing, err := c.factoriesIngress[c.lookupNamespace(src.Namespace)].Networking().V1().Ingresses().Lister().Ingresses(src.Namespace).Get(src.Name)
	if err != nil {
		return fmt.Errorf("failed to get ingress %s/%s: %w", src.Namespace, src.Name, err)
	}

	logger := log.WithoutContext().WithField("namespace", ing.Namespace).WithField("ingress", ing.Name)

	if isLoadBalancerIngressEquals(ing.Status.LoadBalancer.Ingress, ingStatus) {
		logger.Debug("Skipping ingress status update")
		return nil
	}

	ingCopy := ing.DeepCopy()
	ingCopy.Status = networkingv1.IngressStatus{LoadBalancer: corev1.LoadBalancerStatus{Ingress: ingStatus}}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	_, err = c.clientset.NetworkingV1().Ingresses(ingCopy.Namespace).UpdateStatus(ctx, ingCopy, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ingress status %s/%s: %w", src.Namespace, src.Name, err)
	}

	logger.Info("Updated ingress status")
	return nil
}

func (c *clientWrapper) updateIngressStatusOld(src *networkingv1.Ingress, ingStatus []corev1.LoadBalancerIngress) error {
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

func (c *clientWrapper) GetIngressClasses() ([]*networkingv1.IngressClass, error) {
	serverVersion, err := c.GetServerVersion()
	if err != nil {
		log.WithoutContext().Errorf("Failed to get server version: %v", err)
		return nil, err
	}

	if c.clusterFactory == nil {
		return nil, errors.New("cluster factory not loaded")
	}

	var ics []*networkingv1.IngressClass
	if !supportsNetworkingV1Ingress(serverVersion) {
		ingressClasses, err := c.clusterFactory.Networking().V1beta1().IngressClasses().Lister().List(labels.Everything())
		if err != nil {
			return nil, err
		}

		for _, ic := range ingressClasses {
			if ic.Spec.Controller == traefikDefaultIngressClassController {
				icN, err := toNetworkingV1IngressClass(ic)
				if err != nil {
					log.WithoutContext().Errorf("Failed to convert ingress class %s from networking/v1beta1 to networking/v1: %v", ic.Name, err)
					continue
				}
				ics = append(ics, icN)
			}
		}

		return ics, nil
	}

	ingressClasses, err := c.clusterFactory.Networking().V1().IngressClasses().Lister().List(labels.Everything())
	if err != nil {
		return nil, err
	}

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
func (c *clientWrapper) GetServerVersion() (*version.Version, error) {
	serverVersion, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve server version: %w", err)
	}

	return version.NewVersion(serverVersion.GitVersion)
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

// filterIngressClassByName return a slice containing ingressclasses with the correct name.
func filterIngressClassByName(ingressClassName string, ics []*networkingv1.IngressClass) []*networkingv1.IngressClass {
	var ingressClasses []*networkingv1.IngressClass

	for _, ic := range ics {
		if ic.Name == ingressClassName {
			ingressClasses = append(ingressClasses, ic)
		}
	}

	return ingressClasses
}

//  Ingress in networking.k8s.io/v1 is supported starting 1.19.
// thus, we query it in K8s starting 1.19.
func supportsNetworkingV1Ingress(serverVersion *version.Version) bool {
	ingressNetworkingVersion := version.Must(version.NewVersion("1.19"))

	return serverVersion.GreaterThanOrEqual(ingressNetworkingVersion)
}
