package ingress

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/api/networking/v1beta1"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestTranslateNotFoundError(t *testing.T) {
	testCases := []struct {
		desc           string
		err            error
		expectedExists bool
		expectedError  error
	}{
		{
			desc:           "kubernetes not found error",
			err:            kubeerror.NewNotFound(schema.GroupResource{}, "foo"),
			expectedExists: false,
			expectedError:  nil,
		},
		{
			desc:           "nil error",
			err:            nil,
			expectedExists: true,
			expectedError:  nil,
		},
		{
			desc:           "not a kubernetes not found error",
			err:            fmt.Errorf("bar error"),
			expectedExists: false,
			expectedError:  fmt.Errorf("bar error"),
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			exists, err := translateNotFoundError(test.err)
			assert.Equal(t, test.expectedExists, exists)
			assert.Equal(t, test.expectedError, err)
		})
	}
}

func TestIsLoadBalancerIngressEquals(t *testing.T) {
	testCases := []struct {
		desc          string
		aSlice        []corev1.LoadBalancerIngress
		bSlice        []corev1.LoadBalancerIngress
		expectedEqual bool
	}{
		{
			desc:          "both slices are empty",
			expectedEqual: true,
		},
		{
			desc: "not the same length",
			bSlice: []corev1.LoadBalancerIngress{
				{IP: "192.168.1.1", Hostname: "traefik"},
			},
			expectedEqual: false,
		},
		{
			desc: "same ordered content",
			aSlice: []corev1.LoadBalancerIngress{
				{IP: "192.168.1.1", Hostname: "traefik"},
			},
			bSlice: []corev1.LoadBalancerIngress{
				{IP: "192.168.1.1", Hostname: "traefik"},
			},
			expectedEqual: true,
		},
		{
			desc: "same unordered content",
			aSlice: []corev1.LoadBalancerIngress{
				{IP: "192.168.1.1", Hostname: "traefik"},
				{IP: "192.168.1.2", Hostname: "traefik2"},
			},
			bSlice: []corev1.LoadBalancerIngress{
				{IP: "192.168.1.2", Hostname: "traefik2"},
				{IP: "192.168.1.1", Hostname: "traefik"},
			},
			expectedEqual: true,
		},
		{
			desc: "different ordered content",
			aSlice: []corev1.LoadBalancerIngress{
				{IP: "192.168.1.1", Hostname: "traefik"},
				{IP: "192.168.1.2", Hostname: "traefik2"},
			},
			bSlice: []corev1.LoadBalancerIngress{
				{IP: "192.168.1.1", Hostname: "traefik"},
				{IP: "192.168.1.2", Hostname: "traefik"},
			},
			expectedEqual: false,
		},
		{
			desc: "different unordered content",
			aSlice: []corev1.LoadBalancerIngress{
				{IP: "192.168.1.1", Hostname: "traefik"},
				{IP: "192.168.1.2", Hostname: "traefik2"},
			},
			bSlice: []corev1.LoadBalancerIngress{
				{IP: "192.168.1.2", Hostname: "traefik3"},
				{IP: "192.168.1.1", Hostname: "traefik"},
			},
			expectedEqual: false,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			gotEqual := isLoadBalancerIngressEquals(test.aSlice, test.bSlice)
			assert.Equal(t, test.expectedEqual, gotEqual)
		})
	}
}

func TestClientIgnoresHelmOwnedSecrets(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "secret",
		},
	}
	helmSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "helm-secret",
			Labels: map[string]string{
				"owner": "helm",
			},
		},
	}

	kubeClient := kubefake.NewSimpleClientset(helmSecret, secret)

	discovery, _ := kubeClient.Discovery().(*fakediscovery.FakeDiscovery)
	discovery.FakedServerVersion = &version.Info{
		GitVersion: "v1.19",
	}

	client := newClientImpl(kubeClient)

	stopCh := make(chan struct{})

	eventCh, err := client.WatchAll(nil, stopCh)
	require.NoError(t, err)

	select {
	case event := <-eventCh:
		secret, ok := event.(*corev1.Secret)
		require.True(t, ok)

		assert.NotEqual(t, "helm-secret", secret.Name)
	case <-time.After(50 * time.Millisecond):
		assert.Fail(t, "expected to receive event for secret")
	}

	select {
	case <-eventCh:
		assert.Fail(t, "received more than one event")
	case <-time.After(50 * time.Millisecond):
	}

	_, found, err := client.GetSecret("default", "secret")
	require.NoError(t, err)
	assert.True(t, found)

	_, found, err = client.GetSecret("default", "helm-secret")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestClientIgnoresEmptyEndpointUpdates(t *testing.T) {
	emptyEndpoint := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "empty-endpoint",
			Namespace:       "test",
			ResourceVersion: "1244",
			Annotations: map[string]string{
				"test-annotation": "_",
			},
		},
	}

	filledEndpoint := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "filled-endpoint",
			Namespace:       "test",
			ResourceVersion: "1234",
		},
		Subsets: []corev1.EndpointSubset{{
			Addresses: []corev1.EndpointAddress{{
				IP: "10.13.37.1",
			}},
			Ports: []corev1.EndpointPort{{
				Name:     "testing",
				Port:     1337,
				Protocol: "tcp",
			}},
		}},
	}

	kubeClient := kubefake.NewSimpleClientset(emptyEndpoint, filledEndpoint)

	discovery, _ := kubeClient.Discovery().(*fakediscovery.FakeDiscovery)
	discovery.FakedServerVersion = &version.Info{
		GitVersion: "v1.19",
	}

	client := newClientImpl(kubeClient)

	stopCh := make(chan struct{})

	eventCh, err := client.WatchAll(nil, stopCh)
	require.NoError(t, err)

	select {
	case event := <-eventCh:
		ep, ok := event.(*corev1.Endpoints)
		require.True(t, ok)

		assert.True(t, ep.Name == "empty-endpoint" || ep.Name == "filled-endpoint")
	case <-time.After(50 * time.Millisecond):
		assert.Fail(t, "expected to receive event for endpoints")
	}

	emptyEndpoint, err = kubeClient.CoreV1().Endpoints("test").Get(context.TODO(), "empty-endpoint", metav1.GetOptions{})
	assert.NoError(t, err)

	// Update endpoint annotation and resource version (apparently not done by fake client itself)
	// to show an update that should not trigger an update event on our eventCh.
	// This reflects the behavior of kubernetes controllers which use endpoint annotations for leader election.
	emptyEndpoint.Annotations["test-annotation"] = "___"
	emptyEndpoint.ResourceVersion = "1245"
	_, err = kubeClient.CoreV1().Endpoints("test").Update(context.TODO(), emptyEndpoint, metav1.UpdateOptions{})
	require.NoError(t, err)

	select {
	case event := <-eventCh:
		ep, ok := event.(*corev1.Endpoints)
		require.True(t, ok)

		assert.Fail(t, "didn't expect to receive event for empty endpoint update", ep.Name)
	case <-time.After(50 * time.Millisecond):
	}

	filledEndpoint, err = kubeClient.CoreV1().Endpoints("test").Get(context.TODO(), "filled-endpoint", metav1.GetOptions{})
	assert.NoError(t, err)

	filledEndpoint.Subsets[0].Addresses[0].IP = "10.13.37.2"
	filledEndpoint.ResourceVersion = "1235"
	_, err = kubeClient.CoreV1().Endpoints("test").Update(context.TODO(), filledEndpoint, metav1.UpdateOptions{})
	require.NoError(t, err)

	select {
	case event := <-eventCh:
		ep, ok := event.(*corev1.Endpoints)
		require.True(t, ok)

		assert.Equal(t, "filled-endpoint", ep.Name)
	case <-time.After(50 * time.Millisecond):
		assert.Fail(t, "expected to receive event for filled endpoint")
	}

	select {
	case <-eventCh:
		assert.Fail(t, "received more than one event")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestClientUsesCorrectServerVersion(t *testing.T) {
	ingressV1Beta := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ingress-v1beta",
		},
	}

	ingressV1 := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ingress-v1",
		},
	}

	kubeClient := kubefake.NewSimpleClientset(ingressV1Beta, ingressV1)

	discovery, _ := kubeClient.Discovery().(*fakediscovery.FakeDiscovery)
	discovery.FakedServerVersion = &version.Info{
		GitVersion: "v1.18.12+foobar",
	}

	stopCh := make(chan struct{})

	client := newClientImpl(kubeClient)

	eventCh, err := client.WatchAll(nil, stopCh)
	require.NoError(t, err)

	select {
	case event := <-eventCh:
		ingress, ok := event.(*v1beta1.Ingress)
		require.True(t, ok)

		assert.Equal(t, "ingress-v1beta", ingress.Name)
	case <-time.After(50 * time.Millisecond):
		assert.Fail(t, "expected to receive event for ingress")
	}

	select {
	case <-eventCh:
		assert.Fail(t, "received more than one event")
	case <-time.After(50 * time.Millisecond):
	}

	discovery.FakedServerVersion = &version.Info{
		GitVersion: "v1.19",
	}

	eventCh, err = client.WatchAll(nil, stopCh)
	require.NoError(t, err)

	select {
	case event := <-eventCh:
		ingress, ok := event.(*networkingv1.Ingress)
		require.True(t, ok)

		assert.Equal(t, "ingress-v1", ingress.Name)
	case <-time.After(50 * time.Millisecond):
		assert.Fail(t, "expected to receive event for ingress")
	}

	select {
	case <-eventCh:
		assert.Fail(t, "received more than one event")
	case <-time.After(50 * time.Millisecond):
	}
}
