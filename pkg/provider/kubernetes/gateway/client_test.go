package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/gateway-api/apis/v1alpha1"
)

func TestStatusEquals(t *testing.T) {
	testCases := []struct {
		desc     string
		statusA  v1alpha1.GatewayStatus
		statusB  v1alpha1.GatewayStatus
		expected bool
	}{
		{
			desc:     "Empty",
			statusA:  v1alpha1.GatewayStatus{},
			statusB:  v1alpha1.GatewayStatus{},
			expected: true,
		},
		{
			desc: "Same status",
			statusA: v1alpha1.GatewayStatus{
				Conditions: []metav1.Condition{
					{
						Type:   "foobar",
						Reason: "foobar",
					},
				},
				Listeners: []v1alpha1.ListenerStatus{
					{
						Port: 80,
						Conditions: []metav1.Condition{
							{
								Type:   "foobar",
								Reason: "foobar",
							},
						},
					},
				},
			},
			statusB: v1alpha1.GatewayStatus{
				Conditions: []metav1.Condition{
					{
						Type:   "foobar",
						Reason: "foobar",
					},
				},
				Listeners: []v1alpha1.ListenerStatus{
					{
						Port: 80,
						Conditions: []metav1.Condition{
							{
								Type:   "foobar",
								Reason: "foobar",
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			desc: "Listeners length not equal",
			statusA: v1alpha1.GatewayStatus{
				Listeners: []v1alpha1.ListenerStatus{},
			},
			statusB: v1alpha1.GatewayStatus{
				Listeners: []v1alpha1.ListenerStatus{
					{},
				},
			},
			expected: false,
		},
		{
			desc: "Gateway conditions length not equal",
			statusA: v1alpha1.GatewayStatus{
				Conditions: []metav1.Condition{},
			},
			statusB: v1alpha1.GatewayStatus{
				Conditions: []metav1.Condition{
					{},
				},
			},
			expected: false,
		},
		{
			desc: "Gateway conditions different types",
			statusA: v1alpha1.GatewayStatus{
				Conditions: []metav1.Condition{
					{
						Type: "foobar",
					},
				},
			},
			statusB: v1alpha1.GatewayStatus{
				Conditions: []metav1.Condition{
					{
						Type: "foobir",
					},
				},
			},
			expected: false,
		},
		{
			desc: "Gateway conditions same types but different reason",
			statusA: v1alpha1.GatewayStatus{
				Conditions: []metav1.Condition{
					{
						Type: "foobar",
					},
				},
			},
			statusB: v1alpha1.GatewayStatus{
				Conditions: []metav1.Condition{
					{
						Type:   "foobar",
						Reason: "Another reason",
					},
				},
			},
			expected: false,
		},
		{
			desc: "Gateway listeners conditions length",
			statusA: v1alpha1.GatewayStatus{
				Listeners: []v1alpha1.ListenerStatus{
					{
						Port:       80,
						Conditions: []metav1.Condition{},
					},
				},
			},
			statusB: v1alpha1.GatewayStatus{
				Listeners: []v1alpha1.ListenerStatus{
					{
						Port: 80,
						Conditions: []metav1.Condition{
							{},
						},
					},
				},
			},
			expected: false,
		},
		{
			desc: "Gateway listeners conditions same types but different status",
			statusA: v1alpha1.GatewayStatus{
				Listeners: []v1alpha1.ListenerStatus{
					{
						Conditions: []metav1.Condition{
							{
								Type: "foobar",
							},
						},
					},
				},
			},
			statusB: v1alpha1.GatewayStatus{
				Listeners: []v1alpha1.ListenerStatus{
					{
						Conditions: []metav1.Condition{
							{
								Type:   "foobar",
								Status: "Another status",
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			desc: "Gateway listeners conditions same types but different message",
			statusA: v1alpha1.GatewayStatus{
				Listeners: []v1alpha1.ListenerStatus{
					{
						Conditions: []metav1.Condition{
							{
								Type: "foobar",
							},
						},
					},
				},
			},
			statusB: v1alpha1.GatewayStatus{
				Listeners: []v1alpha1.ListenerStatus{
					{
						Conditions: []metav1.Condition{
							{
								Type:    "foobar",
								Message: "Another status",
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			desc: "Gateway listeners conditions same types/reason but different ports",
			statusA: v1alpha1.GatewayStatus{
				Listeners: []v1alpha1.ListenerStatus{
					{
						Port: 80,
						Conditions: []metav1.Condition{
							{
								Type:   "foobar",
								Reason: "foobar",
							},
						},
					},
				},
			},
			statusB: v1alpha1.GatewayStatus{
				Listeners: []v1alpha1.ListenerStatus{
					{
						Port: 443,
						Conditions: []metav1.Condition{
							{
								Type:   "foobar",
								Reason: "foobar",
							},
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, test := range testCases {
		test := test

		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			result := statusEquals(test.statusA, test.statusB)

			assert.Equal(t, test.expected, result)
		})
	}
}
