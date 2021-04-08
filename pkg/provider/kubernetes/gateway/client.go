package gateway

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/traefik/traefik/v2/pkg/log"
	corev1 "k8s.io/api/core/v1"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/gateway-api/apis/v1alpha1"
	"sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	"sigs.k8s.io/gateway-api/pkg/client/informers/externalversions"
)

const resyncPeriod = 10 * time.Minute

type resourceEventHandler struct {
	ev chan<- interface{}
}

func (reh *resourceEventHandler) OnAdd(obj interface{}) {
	eventHandlerFunc(reh.ev, obj)
}

func (reh *resourceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	switch oldObj.(type) {
	case *v1alpha1.GatewayClass:
		// Skip update for gateway classes. We only manage addition or deletion for this cluster-wide resource.
		return
	default:
		eventHandlerFunc(reh.ev, newObj)
	}
}

func (reh *resourceEventHandler) OnDelete(obj interface{}) {
	eventHandlerFunc(reh.ev, obj)
}

// Client is a client for the Provider master.
// WatchAll starts the watch of the Provider resources and updates the stores.
// The stores can then be accessed via the Get* functions.
type Client interface {
	WatchAll(namespaces []string, stopCh <-chan struct{}) (<-chan interface{}, error)

	GetGatewayClasses() ([]*v1alpha1.GatewayClass, error)
	UpdateGatewayStatus(gateway *v1alpha1.Gateway, gatewayStatus v1alpha1.GatewayStatus) error
	UpdateGatewayClassStatus(gatewayClass *v1alpha1.GatewayClass, condition metav1.Condition) error
	GetGateways() []*v1alpha1.Gateway
	GetHTTPRoutes(namespace string, selector labels.Selector) ([]*v1alpha1.HTTPRoute, error)

	GetService(namespace, name string) (*corev1.Service, bool, error)
	GetSecret(namespace, name string) (*corev1.Secret, bool, error)
	GetEndpoints(namespace, name string) (*corev1.Endpoints, bool, error)
}

type clientWrapper struct {
	csGateway versioned.Interface
	csKube    kubernetes.Interface

	factoryGatewayClass externalversions.SharedInformerFactory
	factoriesGateway    map[string]externalversions.SharedInformerFactory
	factoriesKube       map[string]informers.SharedInformerFactory
	factoriesSecret     map[string]informers.SharedInformerFactory

	isNamespaceAll    bool
	watchedNamespaces []string

	labelSelector string
}

func createClientFromConfig(c *rest.Config) (*clientWrapper, error) {
	csGateway, err := versioned.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	csKube, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	return newClientImpl(csKube, csGateway), nil
}

func newClientImpl(csKube kubernetes.Interface, csGateway versioned.Interface) *clientWrapper {
	return &clientWrapper{
		csGateway:        csGateway,
		csKube:           csKube,
		factoriesGateway: make(map[string]externalversions.SharedInformerFactory),
		factoriesKube:    make(map[string]informers.SharedInformerFactory),
		factoriesSecret:  make(map[string]informers.SharedInformerFactory),
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
	eventHandler := &resourceEventHandler{ev: eventCh}

	if len(namespaces) == 0 {
		namespaces = []string{metav1.NamespaceAll}
		c.isNamespaceAll = true
	}

	c.watchedNamespaces = namespaces

	notOwnedByHelm := func(opts *metav1.ListOptions) {
		opts.LabelSelector = "owner!=helm"
	}

	labelSelectorOptions := func(options *metav1.ListOptions) {
		options.LabelSelector = c.labelSelector
	}

	c.factoryGatewayClass = externalversions.NewSharedInformerFactoryWithOptions(c.csGateway, resyncPeriod, externalversions.WithTweakListOptions(labelSelectorOptions))
	c.factoryGatewayClass.Networking().V1alpha1().GatewayClasses().Informer().AddEventHandler(eventHandler)

	for _, ns := range namespaces {
		factoryGateway := externalversions.NewSharedInformerFactoryWithOptions(c.csGateway, resyncPeriod, externalversions.WithNamespace(ns))
		factoryGateway.Networking().V1alpha1().Gateways().Informer().AddEventHandler(eventHandler)
		factoryGateway.Networking().V1alpha1().HTTPRoutes().Informer().AddEventHandler(eventHandler)

		factoryKube := informers.NewSharedInformerFactoryWithOptions(c.csKube, resyncPeriod, informers.WithNamespace(ns))
		factoryKube.Core().V1().Services().Informer().AddEventHandler(eventHandler)
		factoryKube.Core().V1().Endpoints().Informer().AddEventHandler(eventHandler)

		factorySecret := informers.NewSharedInformerFactoryWithOptions(c.csKube, resyncPeriod, informers.WithNamespace(ns), informers.WithTweakListOptions(notOwnedByHelm))
		factorySecret.Core().V1().Secrets().Informer().AddEventHandler(eventHandler)

		c.factoriesGateway[ns] = factoryGateway
		c.factoriesKube[ns] = factoryKube
		c.factoriesSecret[ns] = factorySecret
	}

	c.factoryGatewayClass.Start(stopCh)

	for _, ns := range namespaces {
		c.factoriesGateway[ns].Start(stopCh)
		c.factoriesKube[ns].Start(stopCh)
		c.factoriesSecret[ns].Start(stopCh)
	}

	for t, ok := range c.factoryGatewayClass.WaitForCacheSync(stopCh) {
		if !ok {
			return nil, fmt.Errorf("timed out waiting for controller caches to sync %s", t.String())
		}
	}

	for _, ns := range namespaces {
		for t, ok := range c.factoriesGateway[ns].WaitForCacheSync(stopCh) {
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

func (c *clientWrapper) GetHTTPRoutes(namespace string, selector labels.Selector) ([]*v1alpha1.HTTPRoute, error) {
	if !c.isWatchedNamespace(namespace) {
		return nil, fmt.Errorf("failed to get HTTPRoute %s with labels selector %s: namespace is not within watched namespaces", namespace, selector)
	}
	httpRoutes, err := c.factoriesGateway[c.lookupNamespace(namespace)].Networking().V1alpha1().HTTPRoutes().Lister().HTTPRoutes(namespace).List(selector)
	if err != nil {
		return nil, err
	}

	if len(httpRoutes) == 0 {
		log.WithoutContext().Debugf("No HTTPRoute found in %q namespace with labels selector %s", namespace, selector)
	}

	return httpRoutes, nil
}

func (c *clientWrapper) GetGateways() []*v1alpha1.Gateway {
	var result []*v1alpha1.Gateway

	for ns, factory := range c.factoriesGateway {
		gateways, err := factory.Networking().V1alpha1().Gateways().Lister().List(labels.Everything())
		if err != nil {
			log.WithoutContext().Errorf("Failed to list Gateways in namespace %s: %v", ns, err)
			continue
		}
		result = append(result, gateways...)
	}

	return result
}

func (c *clientWrapper) GetGatewayClasses() ([]*v1alpha1.GatewayClass, error) {
	return c.factoryGatewayClass.Networking().V1alpha1().GatewayClasses().Lister().List(labels.Everything())
}

func (c *clientWrapper) UpdateGatewayClassStatus(gatewayClass *v1alpha1.GatewayClass, condition metav1.Condition) error {
	gc := gatewayClass.DeepCopy()

	var newConditions []metav1.Condition
	for _, cond := range gc.Status.Conditions {
		// No update for identical condition.
		if cond.Type == condition.Type && cond.Status == condition.Status {
			return nil
		}

		// Keep other condition types.
		if cond.Type != condition.Type {
			newConditions = append(newConditions, cond)
		}
	}

	// Append the condition to update.
	newConditions = append(newConditions, condition)
	gc.Status.Conditions = newConditions

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.csGateway.NetworkingV1alpha1().GatewayClasses().UpdateStatus(ctx, gc, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update GatewayClass %q status: %w", gatewayClass.Name, err)
	}

	return nil
}

func (c *clientWrapper) UpdateGatewayStatus(gateway *v1alpha1.Gateway, gatewayStatus v1alpha1.GatewayStatus) error {
	if !c.isWatchedNamespace(gateway.Namespace) {
		return fmt.Errorf("cannot update Gateway status %s/%s: namespace is not within watched namespaces", gateway.Namespace, gateway.Name)
	}

	if statusEquals(gateway.Status, gatewayStatus) {
		return nil
	}

	g := gateway.DeepCopy()
	g.Status = gatewayStatus

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.csGateway.NetworkingV1alpha1().Gateways(gateway.Namespace).UpdateStatus(ctx, g, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update Gateway %q status: %w", gateway.Name, err)
	}

	return nil
}

func statusEquals(oldStatus, newStatus v1alpha1.GatewayStatus) bool {
	if len(oldStatus.Listeners) != len(newStatus.Listeners) {
		return false
	}

	if !conditionsEquals(oldStatus.Conditions, newStatus.Conditions) {
		return false
	}

	listenerMatches := 0
	for _, newListener := range newStatus.Listeners {
		for _, oldListener := range oldStatus.Listeners {
			if newListener.Port == oldListener.Port {
				if !conditionsEquals(newListener.Conditions, oldListener.Conditions) {
					return false
				}

				listenerMatches++
			}
		}
	}

	return listenerMatches == len(oldStatus.Listeners)
}

func conditionsEquals(conditionsA, conditionsB []metav1.Condition) bool {
	if len(conditionsA) != len(conditionsB) {
		return false
	}

	conditionMatches := 0
	for _, conditionA := range conditionsA {
		for _, conditionB := range conditionsB {
			if conditionA.Type == conditionB.Type {
				if conditionA.Reason != conditionB.Reason || conditionA.Status != conditionB.Status || conditionA.Message != conditionB.Message {
					return false
				}
				conditionMatches++
			}
		}
	}

	return conditionMatches == len(conditionsA)
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
