package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_detectChanges(t *testing.T) {
	tests := []struct {
		name   string
		oldObj interface{}
		newObj interface{}
		want   bool
	}{
		{
			name: "With nil values",
			want: true,
		},
		{
			name:   "With empty endpoints",
			oldObj: &corev1.Endpoints{},
			newObj: &corev1.Endpoints{},
		},
		{
			name:   "With old nil",
			newObj: &corev1.Endpoints{},
			want:   true,
		},
		{
			name:   "With new nil",
			oldObj: &corev1.Endpoints{},
			want:   true,
		},
		{
			name: "With same version",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
			},
		},
		{
			name: "With different version",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
			},
		},
		{
			name: "With same annotations",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
					Annotations: map[string]string{
						"test-annotation": "_",
					},
				},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
					Annotations: map[string]string{
						"test-annotation": "_",
					},
				},
			},
		},
		{
			name: "With different annotations",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
					Annotations: map[string]string{
						"test-annotation": "V",
					},
				},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
					Annotations: map[string]string{
						"test-annotation": "X",
					},
				},
			},
		},
		{
			name: "With same subsets",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{},
			},
		},
		{
			name: "With different len of subsets",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
				Subsets: []corev1.EndpointSubset{{}},
			},
			want: true,
		},
		{
			name: "With same subsets with same len of addresses",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{},
				}},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{},
				}},
			},
		},
		{
			name: "With same subsets with different len of addresses",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{},
				}},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{{}},
				}},
			},
			want: true,
		},
		{
			name: "With same subsets with same len of ports",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{{
					Ports: []corev1.EndpointPort{},
				}},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
				Subsets: []corev1.EndpointSubset{{
					Ports: []corev1.EndpointPort{},
				}},
			},
		},
		{
			name: "With same subsets with different len of ports",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{{
					Ports: []corev1.EndpointPort{},
				}},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
				Subsets: []corev1.EndpointSubset{{
					Ports: []corev1.EndpointPort{{}},
				}},
			},
			want: true,
		},
		{
			name: "With same subsets with same len of addresses with same ip",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{{
						IP: "10.10.10.10",
					}},
				}},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{{
						IP: "10.10.10.10",
					}},
				}},
			},
		},
		{
			name: "With same subsets with same len of addresses with different ip",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{{
						IP: "10.10.10.10",
					}},
				}},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{{
						IP: "10.10.10.42",
					}},
				}},
			},
			want: true,
		},
		{
			name: "With same subsets with same len of addresses with same hostname",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{{
						Hostname: "foo",
					}},
				}},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{{
						Hostname: "foo",
					}},
				}},
			},
		},
		{
			name: "With same subsets with same len of addresses with same hostname",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{{
						Hostname: "foo",
					}},
				}},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{{
						Hostname: "bar",
					}},
				}},
			},
			want: true,
		},
		{
			name: "With same subsets with same len of port with same name",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{{
					Ports: []corev1.EndpointPort{{
						Name: "foo",
					}},
				}},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
				Subsets: []corev1.EndpointSubset{{
					Ports: []corev1.EndpointPort{{
						Name: "foo",
					}},
				}},
			},
		},
		{
			name: "With same subsets with same len of port with different name",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{{
					Ports: []corev1.EndpointPort{{
						Name: "foo",
					}},
				}},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
				Subsets: []corev1.EndpointSubset{{
					Ports: []corev1.EndpointPort{{
						Name: "bar",
					}},
				}},
			},
			want: true,
		},
		{
			name: "With same subsets with same len of port with same port",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{{
					Ports: []corev1.EndpointPort{{
						Port: 4242,
					}},
				}},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
				Subsets: []corev1.EndpointSubset{{
					Ports: []corev1.EndpointPort{{
						Port: 4242,
					}},
				}},
			},
		},
		{
			name: "With same subsets with same len of port with different port",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{{
					Ports: []corev1.EndpointPort{{
						Port: 4242,
					}},
				}},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
				Subsets: []corev1.EndpointSubset{{
					Ports: []corev1.EndpointPort{{
						Port: 6969,
					}},
				}},
			},
			want: true,
		},
		{
			name: "With same subsets with same len of port with same protocol",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{{
					Ports: []corev1.EndpointPort{{
						Protocol: "HTTP",
					}},
				}},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
				Subsets: []corev1.EndpointSubset{{
					Ports: []corev1.EndpointPort{{
						Protocol: "HTTP",
					}},
				}},
			},
		},
		{
			name: "With same subsets with same len of port with different protocol",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{{
					Ports: []corev1.EndpointPort{{
						Protocol: "HTTP",
					}},
				}},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
				Subsets: []corev1.EndpointSubset{{
					Ports: []corev1.EndpointPort{{
						Protocol: "TCP",
					}},
				}},
			},
			want: true,
		},
		{
			name: "With same subsets with same subset",
			oldObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
				},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{{
						IP:       "10.10.10.10",
						Hostname: "foo",
					}},
					Ports: []corev1.EndpointPort{{
						Name:     "bar",
						Port:     4242,
						Protocol: "HTTP",
					}},
				}},
			},
			newObj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
				},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{{
						IP:       "10.10.10.10",
						Hostname: "foo",
					}},
					Ports: []corev1.EndpointPort{{
						Name:     "bar",
						Port:     4242,
						Protocol: "HTTP",
					}},
				}},
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.want, objChanged(test.oldObj, test.newObj))
		})
	}
}
