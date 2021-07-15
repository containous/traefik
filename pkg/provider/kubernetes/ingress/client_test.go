package ingress

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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

	fakeDiscovery, ok := kubeClient.Discovery().(*fakediscovery.FakeDiscovery)
	if !ok {
		t.Fatalf("couldn't convert Discovery() to *FakeDiscovery")
	}
	fakeDiscovery.FakedServerVersion = &version.Info{
		GitVersion: "v1.17",
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
