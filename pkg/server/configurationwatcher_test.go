package server

import (
	"context"
	"fmt"
	"strconv"
	"sync"
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
	first    chan struct{}
}

func (p *mockProvider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	wait := p.wait
	if wait == 0 {
		wait = 20 * time.Millisecond
	}

	if len(p.messages) == 0 {
		return fmt.Errorf("no messages available")
	}

	configurationChan <- p.messages[0]

	if p.first != nil {
		<-p.first
	}

	for _, message := range p.messages[1:] {
		time.Sleep(wait)
		configurationChan <- message
	}

	return nil
}

func (p *mockProvider) Init() error {
	panic("implement me")
}

func TestNewConfigurationWatcher(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())
	defer routinesPool.Stop()
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
				Routers:     map[string]*dynamic.TCPRouter{},
				Middlewares: map[string]*dynamic.TCPMiddleware{},
				Services:    map[string]*dynamic.TCPService{},
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

	watcher.start()
	<-run
}

func TestWaitForRequiredProvider(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())
	defer routinesPool.Stop()

	pvdAggregator := &mockProvider{
		wait: 5 * time.Millisecond,
	}

	config := &dynamic.Configuration{
		HTTP: th.BuildConfiguration(
			th.WithRouters(th.WithRouter("foo")),
			th.WithLoadBalancerServices(th.WithService("bar")),
		),
	}

	pvdAggregator.messages = append(pvdAggregator.messages, dynamic.Message{
		ProviderName:  "mock",
		Configuration: config,
	})

	pvdAggregator.messages = append(pvdAggregator.messages, dynamic.Message{
		ProviderName:  "required",
		Configuration: config,
	})

	pvdAggregator.messages = append(pvdAggregator.messages, dynamic.Message{
		ProviderName:  "mock2",
		Configuration: config,
	})

	watcher := NewConfigurationWatcher(routinesPool, pvdAggregator, 1*time.Millisecond, []string{}, "required")

	publishedConfigCount := 0
	watcher.AddListener(func(_ dynamic.Configuration) {
		publishedConfigCount++
	})

	watcher.start()

	// give some time so that the configuration can be processed
	time.Sleep(20 * time.Millisecond)

	// after 20 milliseconds we should have 2 configs published
	assert.Equal(t, 2, publishedConfigCount, "times configs were published")
}

func TestListenProvidersThrottleProviderConfigReload(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())
	defer routinesPool.Stop()

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

	watcher.start()

	// give some time so that the configuration can be processed
	time.Sleep(100 * time.Millisecond)

	// after 50 milliseconds 5 new configs were published
	// with a throttle duration of 30 milliseconds this means, we should have received 3 new configs
	assert.Equal(t, 3, publishedConfigCount, "times configs were published")
}

func TestListenProvidersSkipsEmptyConfigs(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())
	defer routinesPool.Stop()
	pvd := &mockProvider{
		messages: []dynamic.Message{{ProviderName: "mock"}},
	}

	watcher := NewConfigurationWatcher(routinesPool, pvd, time.Second, []string{}, "")
	watcher.AddListener(func(_ dynamic.Configuration) {
		t.Error("An empty configuration was published but it should not")
	})
	watcher.start()

	// give some time so that the configuration can be processed
	time.Sleep(100 * time.Millisecond)
}

func TestListenProvidersSkipsSameConfigurationForProvider(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())
	defer routinesPool.Stop()
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

	watcher.start()

	// give some time so that the configuration can be processed
	time.Sleep(100 * time.Millisecond)
}

func TestListenProvidersDoesNotSkipFlappingConfiguration(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())
	defer routinesPool.Stop()

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

	watcher.start()

	// give some time so that the configuration can be processed
	time.Sleep(100 * time.Millisecond)

	expected := dynamic.Configuration{
		HTTP: th.BuildConfiguration(
			th.WithRouters(th.WithRouter("foo@mock", th.WithEntryPoints("defaultEP"))),
			th.WithLoadBalancerServices(th.WithService("bar@mock")),
			th.WithMiddlewares(),
		),
		TCP: &dynamic.TCPConfiguration{
			Routers:     map[string]*dynamic.TCPRouter{},
			Middlewares: map[string]*dynamic.TCPMiddleware{},
			Services:    map[string]*dynamic.TCPService{},
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

func TestListenProvidersIgnoreSameConfig(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())
	defer routinesPool.Stop()

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

	// The transient configuration is sent alternatively with the configuration we want to be applied.
	// It is intended to show that even if the configurations are different,
	// those transient configurations will be ignored if they are sent in a time frame
	// lower than the provider throttle duration.
	pvd := &mockProvider{
		wait:  1 * time.Microsecond, // Enqueue them fast
		first: make(chan struct{}),
		messages: []dynamic.Message{
			{ProviderName: "mock", Configuration: configuration},
			{ProviderName: "mock", Configuration: transientConfiguration},
			{ProviderName: "mock", Configuration: configuration},
			{ProviderName: "mock", Configuration: transientConfiguration},
			{ProviderName: "mock", Configuration: configuration},
		},
	}

	watcher := NewConfigurationWatcher(routinesPool, pvd, 1*time.Millisecond, []string{"defaultEP"}, "")

	var configurationReloads int
	var lastConfig dynamic.Configuration
	var once sync.Once
	watcher.AddListener(func(conf dynamic.Configuration) {
		configurationReloads++
		lastConfig = conf

		// Allows next configurations to be sent by the mock provider
		// as soon as the first configuration message is applied.
		once.Do(func() {
			pvd.first <- struct{}{}
			// Wait for all configuration messages to pile in
			time.Sleep(5 * time.Millisecond)
		})
	})

	watcher.start()

	// Wait long enough
	time.Sleep(50 * time.Millisecond)

	expected := dynamic.Configuration{
		HTTP: th.BuildConfiguration(
			th.WithRouters(th.WithRouter("foo@mock", th.WithEntryPoints("defaultEP"))),
			th.WithLoadBalancerServices(th.WithService("bar@mock")),
			th.WithMiddlewares(),
		),
		TCP: &dynamic.TCPConfiguration{
			Routers:     map[string]*dynamic.TCPRouter{},
			Middlewares: map[string]*dynamic.TCPMiddleware{},
			Services:    map[string]*dynamic.TCPService{},
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

	assert.Equal(t, 1, configurationReloads)
}

func TestApplyConfigUnderStress(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())
	watcher := NewConfigurationWatcher(routinesPool, &mockProvider{}, 1*time.Nanosecond, []string{"defaultEP"}, "")

	routinesPool.GoCtx(func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				watcher.throttledConfigs <- dynamic.Message{ProviderName: "mock", Configuration: &dynamic.Configuration{}}
			}
		}
	})

	var configurationReloads int
	watcher.AddListener(func(conf dynamic.Configuration) {
		configurationReloads++
	})

	watcher.start()
	time.Sleep(100 * time.Millisecond)
	routinesPool.Stop()

	// Ensure that at least one configuration has been applied
	// while the configuration validated channel is constantly fed.
	t.Log(configurationReloads)
	assert.GreaterOrEqual(t, configurationReloads, 1)
}

func TestListenProvidersIgnoreIntermediateConfigs(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())
	defer routinesPool.Stop()

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

	transientConfiguration2 := &dynamic.Configuration{
		HTTP: th.BuildConfiguration(
			th.WithRouters(th.WithRouter("bad2")),
			th.WithLoadBalancerServices(th.WithService("bad2")),
		),
	}

	finalConfiguration := &dynamic.Configuration{
		HTTP: th.BuildConfiguration(
			th.WithRouters(th.WithRouter("final")),
			th.WithLoadBalancerServices(th.WithService("final")),
		),
	}

	pvd := &mockProvider{
		wait: 10 * time.Microsecond, // Enqueue them fast
		messages: []dynamic.Message{
			{ProviderName: "mock", Configuration: configuration},
			{ProviderName: "mock", Configuration: transientConfiguration},
			{ProviderName: "mock", Configuration: transientConfiguration2},
			{ProviderName: "mock", Configuration: finalConfiguration},
		},
	}

	watcher := NewConfigurationWatcher(routinesPool, pvd, 10*time.Millisecond, []string{"defaultEP"}, "")

	var configurationReloads int
	var lastConfig dynamic.Configuration
	watcher.AddListener(func(conf dynamic.Configuration) {
		configurationReloads++
		lastConfig = conf
	})

	watcher.start()

	// Wait long enough
	time.Sleep(50 * time.Millisecond)

	expected := dynamic.Configuration{
		HTTP: th.BuildConfiguration(
			th.WithRouters(th.WithRouter("final@mock", th.WithEntryPoints("defaultEP"))),
			th.WithLoadBalancerServices(th.WithService("final@mock")),
			th.WithMiddlewares(),
		),
		TCP: &dynamic.TCPConfiguration{
			Routers:     map[string]*dynamic.TCPRouter{},
			Middlewares: map[string]*dynamic.TCPMiddleware{},
			Services:    map[string]*dynamic.TCPService{},
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

	assert.Equal(t, 2, configurationReloads)
}

func TestListenProvidersPublishesConfigForEachProvider(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())
	defer routinesPool.Stop()

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

	watcher.start()

	// give some time so that the configuration can be processed
	time.Sleep(100 * time.Millisecond)

	expected := dynamic.Configuration{
		HTTP: th.BuildConfiguration(
			th.WithRouters(
				th.WithRouter("foo@mock", th.WithEntryPoints("defaultEP")),
				th.WithRouter("foo@mock2", th.WithEntryPoints("defaultEP")),
			),
			th.WithLoadBalancerServices(
				th.WithService("bar@mock"),
				th.WithService("bar@mock2"),
			),
			th.WithMiddlewares(),
		),
		TCP: &dynamic.TCPConfiguration{
			Routers:     map[string]*dynamic.TCPRouter{},
			Middlewares: map[string]*dynamic.TCPMiddleware{},
			Services:    map[string]*dynamic.TCPService{},
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
	defer routinesPool.Stop()

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

	watcher.start()

	// give some time so that the configuration can be processed.
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 2, publishedConfigCount)
}

func TestPublishConfigUpdatedByConfigWatcherListener(t *testing.T) {
	routinesPool := safe.NewPool(context.Background())
	defer routinesPool.Stop()

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

	watcher.start()

	// give some time so that the configuration can be processed.
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 1, publishedConfigCount)
}
