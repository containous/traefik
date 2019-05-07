package tcp

import (
	"context"
	"errors"
	"testing"

	"github.com/containous/traefik/pkg/config"
	"github.com/containous/traefik/pkg/server/internal"
	"github.com/stretchr/testify/assert"
)

func TestManager_Build(t *testing.T) {
	testCases := []struct {
		desc         string
		serviceName  string
		configs      map[string]*config.TCPServiceInfo
		providerName string
		expected     error
	}{
		{
			desc:        "Simple service name",
			serviceName: "serviceName",
			configs: map[string]*config.TCPServiceInfo{
				"serviceName": {
					TCPService: &config.TCPService{
						LoadBalancer: &config.TCPLoadBalancerService{Method: "wrr"},
					},
				},
			},
		},
		{
			desc:        "Service name with provider",
			serviceName: "provider-1.serviceName",
			configs: map[string]*config.TCPServiceInfo{
				"provider-1.serviceName": {
					TCPService: &config.TCPService{
						LoadBalancer: &config.TCPLoadBalancerService{Method: "wrr"},
					},
				},
			},
		},
		{
			desc:        "Service name with provider in context",
			serviceName: "serviceName",
			configs: map[string]*config.TCPServiceInfo{
				"provider-1.serviceName": {
					TCPService: &config.TCPService{
						LoadBalancer: &config.TCPLoadBalancerService{Method: "wrr"},
					},
				},
			},
			providerName: "provider-1",
		},
		{
			desc:        "Server with correct host:port as address",
			serviceName: "serviceName",
			configs: map[string]*config.TCPServiceInfo{
				"provider-1.serviceName": {
					TCPService: &config.TCPService{
						LoadBalancer: &config.TCPLoadBalancerService{
							Servers: []config.TCPServer{
								{
									Address: "foobar.com:80",
								},
							},
							Method: "wrr",
						},
					},
				},
			},
			providerName: "provider-1",
		},
		{
			desc:        "Server with correct ip:port as address",
			serviceName: "serviceName",
			configs: map[string]*config.TCPServiceInfo{
				"provider-1.serviceName": {
					TCPService: &config.TCPService{
						LoadBalancer: &config.TCPLoadBalancerService{
							Servers: []config.TCPServer{
								{
									Address: "192.168.0.12:80",
								},
							},
							Method: "wrr",
						},
					},
				},
			},
			providerName: "provider-1",
		},
		{
			desc:        "Server address, hostname but missing port",
			serviceName: "serviceName",
			configs: map[string]*config.TCPServiceInfo{
				"provider-1.serviceName": {
					TCPService: &config.TCPService{
						LoadBalancer: &config.TCPLoadBalancerService{
							Servers: []config.TCPServer{
								{
									Address: "foobar.com",
								},
							},
							Method: "wrr",
						},
					},
				},
			},
			providerName: "provider-1",
			expected:     errors.New(`in service provider-1.serviceName: address foobar.com: missing port in address`),
		},
		{
			desc:        "Server address, ip but missing port",
			serviceName: "serviceName",
			configs: map[string]*config.TCPServiceInfo{
				"provider-1.serviceName": {
					TCPService: &config.TCPService{
						LoadBalancer: &config.TCPLoadBalancerService{
							Servers: []config.TCPServer{
								{
									Address: "192.168.0.12",
								},
							},
							Method: "wrr",
						},
					},
				},
			},
			providerName: "provider-1",
			expected:     errors.New("in service provider-1.serviceName: address 192.168.0.12: missing port in address"),
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			manager := NewManager(&config.RuntimeConfiguration{
				TCPServices: test.configs,
			})

			ctx := context.Background()
			if len(test.providerName) > 0 {
				ctx = internal.AddProviderInContext(ctx, test.providerName+".foobar")
			}

			_, err := manager.BuildTCP(ctx, test.serviceName)
			assert.Equal(t, test.expected, err)
		})
	}
}
