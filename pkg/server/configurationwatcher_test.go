package server

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/safe"
	th "github.com/traefik/traefik/v2/pkg/testhelpers"
	"github.com/traefik/traefik/v2/pkg/tls"
)

type mockProvider struct {
	messages []dynamic.Message
	wait     time.Duration
}

func (p *mockProvider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	for _, message := range p.messages {
		configurationChan <- message

		wait := p.wait
		if wait == 0 {
			wait = 20 * time.Millisecond
		}

		fmt.Println("wait", wait, time.Now().Nanosecond())
		time.Sleep(wait)
	}

	return nil
}

func (p *mockProvider) Init() error {
	panic("implement me")
}

func TestNewConfigurationWatcher(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())
	pvd := &mockProvider{
		messages: []dynamic.Message{{
			ProviderName: "mock",
			Configuration: &dynamic.Configuration{
				HTTP: th.BuildConfiguration(
					th.WithRouters(
						th.WithRouter("test",
							th.WithEntryPoints("e"),
							th.WithServiceName("scv"))),
				),
			},
		}},
	}

	watcher := NewConfigurationWatcher(routinesPool, pvd, time.Second, []string{}, "")

	run := make(chan struct{})

	watcher.AddListener(func(conf dynamic.Configuration) {
		expected := dynamic.Configuration{
			HTTP: th.BuildConfiguration(
				th.WithRouters(
					th.WithRouter("test@mock",
						th.WithEntryPoints("e"),
						th.WithServiceName("scv"))),
				th.WithMiddlewares(),
				th.WithLoadBalancerServices(),
			),
			TCP: &dynamic.TCPConfiguration{
				Routers:  map[string]*dynamic.TCPRouter{},
				Services: map[string]*dynamic.TCPService{},
			},
			TLS: &dynamic.TLSConfiguration{
				Options: map[string]tls.Options{
					"default": {},
				},
				Stores: map[string]tls.Store{},
			},
			UDP: &dynamic.UDPConfiguration{
				Routers:  map[string]*dynamic.UDPRouter{},
				Services: map[string]*dynamic.UDPService{},
			},
		}

		assert.Equal(t, expected, conf)
		close(run)
	})

	watcher.Start()
	<-run
}

func TestListenProvidersThrottleProviderConfigReload(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())

	pvd := &mockProvider{
		wait: 10 * time.Millisecond,
	}

	for i := 0; i < 5; i++ {
		pvd.messages = append(pvd.messages, dynamic.Message{
			ProviderName: "mock",
			Configuration: &dynamic.Configuration{
				HTTP: th.BuildConfiguration(
					th.WithRouters(th.WithRouter("foo"+strconv.Itoa(i))),
					th.WithLoadBalancerServices(th.WithService("bar")),
				),
			},
		})
	}

	watcher := NewConfigurationWatcher(routinesPool, pvd, 30*time.Millisecond, []string{}, "")

	publishedConfigCount := 0
	watcher.AddListener(func(_ dynamic.Configuration) {
		publishedConfigCount++
	})

	watcher.Start()
	defer watcher.Stop()

	// give some time so that the configuration can be processed
	time.Sleep(100 * time.Millisecond)

	// after 50 milliseconds 5 new configs were published
	// with a throttle duration of 30 milliseconds this means, we should have received 3 new configs
	assert.Equal(t, 3, publishedConfigCount, "times configs were published")
}

func TestListenProvidersSkipsEmptyConfigs(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())
	pvd := &mockProvider{
		messages: []dynamic.Message{{ProviderName: "mock"}},
	}

	watcher := NewConfigurationWatcher(routinesPool, pvd, time.Second, []string{}, "")
	watcher.AddListener(func(_ dynamic.Configuration) {
		t.Error("An empty configuration was published but it should not")
	})
	watcher.Start()
	defer watcher.Stop()

	// give some time so that the configuration can be processed
	time.Sleep(100 * time.Millisecond)
}

func TestListenProvidersSkipsSameConfigurationForProvider(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())
	message := dynamic.Message{
		ProviderName: "mock",
		Configuration: &dynamic.Configuration{
			HTTP: th.BuildConfiguration(
				th.WithRouters(th.WithRouter("foo")),
				th.WithLoadBalancerServices(th.WithService("bar")),
			),
		},
	}
	pvd := &mockProvider{
		messages: []dynamic.Message{message, message},
	}

	watcher := NewConfigurationWatcher(routinesPool, pvd, 0, []string{}, "")

	alreadyCalled := false
	watcher.AddListener(func(_ dynamic.Configuration) {
		if alreadyCalled {
			t.Error("Same configuration should not be published multiple times")
		}
		alreadyCalled = true
	})

	watcher.Start()
	defer watcher.Stop()

	// give some time so that the configuration can be processed
	time.Sleep(100 * time.Millisecond)
}

func TestListenProvidersDoesNotSkipFlappingConfiguration(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())

	configuration := &dynamic.Configuration{
		HTTP: th.BuildConfiguration(
			th.WithRouters(th.WithRouter("foo")),
			th.WithLoadBalancerServices(th.WithService("bar")),
		),
	}

	transientConfiguration := &dynamic.Configuration{
		HTTP: th.BuildConfiguration(
			th.WithRouters(th.WithRouter("bad")),
			th.WithLoadBalancerServices(th.WithService("bad")),
		),
	}

	pvd := &mockProvider{
		wait: 5 * time.Millisecond, // The last message needs to be received before the second has been fully processed
		messages: []dynamic.Message{
			{ProviderName: "mock", Configuration: configuration},
			{ProviderName: "mock", Configuration: transientConfiguration},
			{ProviderName: "mock", Configuration: configuration},
		},
	}

	watcher := NewConfigurationWatcher(routinesPool, pvd, 15*time.Millisecond, []string{"defaultEP"}, "")

	var lastConfig dynamic.Configuration
	watcher.AddListener(func(conf dynamic.Configuration) {
		lastConfig = conf
	})

	watcher.Start()
	defer watcher.Stop()

	// give some time so that the configuration can be processed
	time.Sleep(100 * time.Millisecond)

	expected := dynamic.Configuration{
		HTTP: th.BuildConfiguration(
			th.WithRouters(th.WithRouter("foo@mock", th.WithEntryPoints("defaultEP"))),
			th.WithLoadBalancerServices(th.WithService("bar@mock")),
			th.WithMiddlewares(),
		),
		TCP: &dynamic.TCPConfiguration{
			Routers:  map[string]*dynamic.TCPRouter{},
			Services: map[string]*dynamic.TCPService{},
		},
		UDP: &dynamic.UDPConfiguration{
			Routers:  map[string]*dynamic.UDPRouter{},
			Services: map[string]*dynamic.UDPService{},
		},
		TLS: &dynamic.TLSConfiguration{
			Options: map[string]tls.Options{
				"default": {},
			},
			Stores: map[string]tls.Store{},
		},
	}

	assert.Equal(t, expected, lastConfig)
}

func TestListenProvidersPublishesConfigForEachProvider(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())

	configuration := &dynamic.Configuration{
		HTTP: th.BuildConfiguration(
			th.WithRouters(th.WithRouter("foo")),
			th.WithLoadBalancerServices(th.WithService("bar")),
		),
	}

	pvd := &mockProvider{
		messages: []dynamic.Message{
			{ProviderName: "mock", Configuration: configuration},
			{ProviderName: "mock2", Configuration: configuration},
		},
	}

	watcher := NewConfigurationWatcher(routinesPool, pvd, 0, []string{"defaultEP"}, "")

	var publishedProviderConfig dynamic.Configuration

	watcher.AddListener(func(conf dynamic.Configuration) {
		publishedProviderConfig = conf
	})

	watcher.Start()
	defer watcher.Stop()

	// give some time so that the configuration can be processed
	time.Sleep(100 * time.Millisecond)

	expected := dynamic.Configuration{
		HTTP: th.BuildConfiguration(
			th.WithRouters(
				th.WithRouter("foo@mock", th.WithEntryPoints("defaultEP")),
				th.WithRouter("foo@mock2", th.WithEntryPoints("defaultEP")),
			),
			th.WithLoadBalancerServices(th.WithService("bar@mock"), th.WithService("bar@mock2")),
			th.WithMiddlewares(),
		),
		TCP: &dynamic.TCPConfiguration{
			Routers:  map[string]*dynamic.TCPRouter{},
			Services: map[string]*dynamic.TCPService{},
		},
		TLS: &dynamic.TLSConfiguration{
			Options: map[string]tls.Options{
				"default": {},
			},
			Stores: map[string]tls.Store{},
		},
		UDP: &dynamic.UDPConfiguration{
			Routers:  map[string]*dynamic.UDPRouter{},
			Services: map[string]*dynamic.UDPService{},
		},
	}

	assert.Equal(t, expected, publishedProviderConfig)
}

func TestPublishConfigUpdatedByProvider(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())

	pvdConfiguration := dynamic.Configuration{
		TCP: &dynamic.TCPConfiguration{
			Routers: map[string]*dynamic.TCPRouter{
				"foo": {},
			},
		},
	}

	pvd := &mockProvider{
		wait: 10 * time.Millisecond,
		messages: []dynamic.Message{
			{
				ProviderName:  "mock",
				Configuration: &pvdConfiguration,
			},
			{
				ProviderName:  "mock",
				Configuration: &pvdConfiguration,
			},
		},
	}

	watcher := NewConfigurationWatcher(routinesPool, pvd, 30*time.Millisecond, []string{}, "")

	publishedConfigCount := 0
	watcher.AddListener(func(configuration dynamic.Configuration) {
		publishedConfigCount++

		// Update the provider configuration published in next dynamic Message which should trigger a new publish.
		pvdConfiguration.TCP.Routers["bar"] = &dynamic.TCPRouter{}
	})

	watcher.Start()
	defer watcher.Stop()

	// give some time so that the configuration can be processed.
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 2, publishedConfigCount)
}

func TestPublishConfigUpdatedByConfigWatcherListener(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())

	pvd := &mockProvider{
		wait: 10 * time.Millisecond,
		messages: []dynamic.Message{
			{
				ProviderName: "mock",
				Configuration: &dynamic.Configuration{
					TCP: &dynamic.TCPConfiguration{
						Routers: map[string]*dynamic.TCPRouter{
							"foo": {},
						},
					},
				},
			},
			{
				ProviderName: "mock",
				Configuration: &dynamic.Configuration{
					TCP: &dynamic.TCPConfiguration{
						Routers: map[string]*dynamic.TCPRouter{
							"foo": {},
						},
					},
				},
			},
		},
	}

	watcher := NewConfigurationWatcher(routinesPool, pvd, 30*time.Millisecond, []string{}, "")

	publishedConfigCount := 0
	watcher.AddListener(func(configuration dynamic.Configuration) {
		publishedConfigCount++

		// Modify the provided configuration. This should not modify the configuration stored in the configuration
		// watcher and cause a new publish.
		configuration.TCP.Routers["foo@mock"].Rule = "bar"
	})

	watcher.Start()
	defer watcher.Stop()

	// give some time so that the configuration can be processed.
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 1, publishedConfigCount)
}
