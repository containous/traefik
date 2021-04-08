package gateway

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/traefik/traefik/v2/pkg/provider/kubernetes/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/gateway-api/apis/v1alpha1"
)

var _ Client = (*clientMock)(nil)

func init() {
	// required by k8s.MustParseYaml
	err := v1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
}

type clientMock struct {
	services  []*corev1.Service
	secrets   []*corev1.Secret
	endpoints []*corev1.Endpoints

	apiServiceError   error
	apiSecretError    error
	apiEndpointsError error

	gatewayClasses []*v1alpha1.GatewayClass
	gateways       []*v1alpha1.Gateway
	httpRoutes     []*v1alpha1.HTTPRoute

	watchChan chan interface{}
}

func newClientMock(paths ...string) clientMock {
	var c clientMock

	for _, path := range paths {
		yamlContent, err := os.ReadFile(filepath.FromSlash("./fixtures/" + path))
		if err != nil {
			panic(err)
		}

		k8sObjects := k8s.MustParseYaml(yamlContent)
		for _, obj := range k8sObjects {
			switch o := obj.(type) {
			case *corev1.Service:
				c.services = append(c.services, o)
			case *corev1.Secret:
				c.secrets = append(c.secrets, o)
			case *corev1.Endpoints:
				c.endpoints = append(c.endpoints, o)
			case *v1alpha1.GatewayClass:
				c.gatewayClasses = append(c.gatewayClasses, o)
			case *v1alpha1.Gateway:
				c.gateways = append(c.gateways, o)
			case *v1alpha1.HTTPRoute:
				c.httpRoutes = append(c.httpRoutes, o)
			default:
				panic(fmt.Sprintf("Unknown runtime object %+v %T", o, o))
			}
		}
	}

	return c
}

func (c clientMock) UpdateGatewayStatus(gateway *v1alpha1.Gateway, gatewayStatus v1alpha1.GatewayStatus) error {
	for _, g := range c.gateways {
		if g.Name == gateway.Name {
			if !statusEquals(g.Status, gatewayStatus) {
				g.Status = gatewayStatus
				return nil
			}
			return fmt.Errorf("cannot update gateway %v", gateway.Name)
		}
	}
	return nil
}

func (c clientMock) UpdateGatewayClassStatus(gatewayClass *v1alpha1.GatewayClass, condition metav1.Condition) error {
	for _, gc := range c.gatewayClasses {
		if gc.Name == gatewayClass.Name {
			for _, c := range gc.Status.Conditions {
				if c.Type == condition.Type && c.Status != condition.Status {
					c.Status = condition.Status
					c.LastTransitionTime = condition.LastTransitionTime
					c.Message = condition.Message
					c.Reason = condition.Reason
				}
			}
		}
	}
	return nil
}

func (c clientMock) UpdateGatewayStatusConditions(gateway *v1alpha1.Gateway, condition metav1.Condition) error {
	for _, g := range c.gatewayClasses {
		if g.Name == gateway.Name {
			for _, c := range g.Status.Conditions {
				if c.Type == condition.Type && (c.Status != condition.Status || c.Reason != condition.Reason) {
					c.Status = condition.Status
					c.LastTransitionTime = condition.LastTransitionTime
					c.Message = condition.Message
					c.Reason = condition.Reason
				}
			}
		}
	}
	return nil
}

func (c clientMock) GetGatewayClasses() ([]*v1alpha1.GatewayClass, error) {
	return c.gatewayClasses, nil
}

func (c clientMock) GetGateways() []*v1alpha1.Gateway {
	return c.gateways
}

func (c clientMock) GetHTTPRoutes(namespace string, selector labels.Selector) ([]*v1alpha1.HTTPRoute, error) {
	httpRoutes := make([]*v1alpha1.HTTPRoute, len(c.httpRoutes))

	for _, httpRoute := range c.httpRoutes {
		if httpRoute.Namespace == namespace && selector.Matches(labels.Set(httpRoute.Labels)) {
			httpRoutes = append(httpRoutes, httpRoute)
		}
	}
	return httpRoutes, nil
}

func (c clientMock) GetService(namespace, name string) (*corev1.Service, bool, error) {
	if c.apiServiceError != nil {
		return nil, false, c.apiServiceError
	}

	for _, service := range c.services {
		if service.Namespace == namespace && service.Name == name {
			return service, true, nil
		}
	}
	return nil, false, c.apiServiceError
}

func (c clientMock) GetEndpoints(namespace, name string) (*corev1.Endpoints, bool, error) {
	if c.apiEndpointsError != nil {
		return nil, false, c.apiEndpointsError
	}

	for _, endpoints := range c.endpoints {
		if endpoints.Namespace == namespace && endpoints.Name == name {
			return endpoints, true, nil
		}
	}

	return &corev1.Endpoints{}, false, nil
}

func (c clientMock) GetSecret(namespace, name string) (*corev1.Secret, bool, error) {
	if c.apiSecretError != nil {
		return nil, false, c.apiSecretError
	}

	for _, secret := range c.secrets {
		if secret.Namespace == namespace && secret.Name == name {
			return secret, true, nil
		}
	}
	return nil, false, nil
}

func (c clientMock) WatchAll(namespaces []string, stopCh <-chan struct{}) (<-chan interface{}, error) {
	return c.watchChan, nil
}
