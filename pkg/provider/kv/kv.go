package kv

import (
	"context"
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/abronan/valkeyrie"
	"github.com/abronan/valkeyrie/store"
	"github.com/abronan/valkeyrie/store/consul"
	etcdv3 "github.com/abronan/valkeyrie/store/etcd/v3"
	"github.com/abronan/valkeyrie/store/redis"
	"github.com/abronan/valkeyrie/store/zookeeper"
	"github.com/cenkalti/backoff/v4"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/config/kv"
	"github.com/traefik/traefik/v2/pkg/job"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/traefik/traefik/v2/pkg/types"
)

// Provider holds configurations of the provider.
type Provider struct {
	RootKey string `description:"Root key used for KV store" export:"true" json:"rootKey,omitempty" toml:"rootKey,omitempty" yaml:"rootKey,omitempty"`

	Endpoints []string         `description:"KV store endpoints" json:"endpoints,omitempty" toml:"endpoints,omitempty" yaml:"endpoints,omitempty"`
	Username  string           `description:"KV Username" json:"username,omitempty" toml:"username,omitempty" yaml:"username,omitempty"`
	Password  string           `description:"KV Password" json:"password,omitempty" toml:"password,omitempty" yaml:"password,omitempty"`
	TLS       *types.ClientTLS `description:"Enable TLS support" export:"true" json:"tls,omitempty" toml:"tls,omitempty" yaml:"tls,omitempty"`

	storeType store.Backend
	kvClient  store.Store
	name      string
}

// SetDefaults sets the default values.
func (p *Provider) SetDefaults() {
	p.RootKey = "traefik"
}

// Init the provider.
func (p *Provider) Init(storeType store.Backend, name string) error {
	ctx := log.With(context.Background(), log.Str(log.ProviderName, name))

	p.storeType = storeType
	p.name = name

	kvClient, err := p.createKVClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to Connect to KV store: %w", err)
	}

	p.kvClient = kvClient

	return nil
}

// Provide allows the docker provider to provide configurations to traefik using the given configuration channel.
func (p *Provider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	ctx := log.With(context.Background(), log.Str(log.ProviderName, p.name))
	logger := log.FromContext(ctx)

	operation := func() error {
		if _, err := p.kvClient.Exists(path.Join(p.RootKey, "qmslkjdfmqlskdjfmqlksjazçueznbvbwzlkajzebvkwjdcqmlsfj"), nil); err != nil {
			return fmt.Errorf("KV store connection error: %w", err)
		}
		return nil
	}

	notify := func(err error, time time.Duration) {
		logger.Errorf("KV connection error: %+v, retrying in %s", err, time)
		p.sendConfigLoadErrorToChannel(configurationChan)
	}
	err := backoff.RetryNotify(safe.OperationWithRecover(operation), job.NewBackOff(backoff.NewExponentialBackOff()), notify)
	if err != nil {
		p.sendConfigLoadErrorToChannel(configurationChan)
		return fmt.Errorf("cannot connect to KV server: %w", err)
	}

	configuration, err := p.buildConfiguration()
	if err != nil {
		logger.Errorf("Cannot build the configuration: %v", err)
		p.sendConfigLoadErrorToChannel(configurationChan)
	} else {
		configurationChan <- dynamic.Message{
			ProviderName:  p.name,
			Configuration: configuration,
		}
	}

	pool.GoCtx(func(ctxPool context.Context) {
		ctxLog := log.With(ctxPool, log.Str(log.ProviderName, p.name))

		err := p.watchKv(ctxLog, configurationChan)
		if err != nil {
			logger.Errorf("Cannot watch KV store: %v", err)
			p.sendConfigLoadErrorToChannel(configurationChan)
		}
	})

	return nil
}

func (p *Provider) watchKv(ctx context.Context, configurationChan chan<- dynamic.Message) error {
	operation := func() error {
		events, err := p.kvClient.WatchTree(p.RootKey, ctx.Done(), nil)
		if err != nil {
			return fmt.Errorf("failed to watch KV: %w", err)
		}

		for {
			select {
			case <-ctx.Done():
				return nil
			case _, ok := <-events:
				if !ok {
					return errors.New("the WatchTree channel is closed")
				}

				configuration, errC := p.buildConfiguration()
				if errC != nil {
					return errC
				}

				if configuration != nil {
					configurationChan <- dynamic.Message{
						ProviderName:  p.name,
						Configuration: configuration,
					}
				}
			}
		}
	}

	notify := func(err error, time time.Duration) {
		log.FromContext(ctx).Errorf("KV connection error: %+v, retrying in %s", err, time)
		p.sendConfigLoadErrorToChannel(configurationChan)
	}

	err := backoff.RetryNotify(safe.OperationWithRecover(operation),
		backoff.WithContext(job.NewBackOff(backoff.NewExponentialBackOff()), ctx), notify)
	if err != nil {
		p.sendConfigLoadErrorToChannel(configurationChan)
		return fmt.Errorf("cannot connect to KV server: %w", err)
	}
	return nil
}

func (p *Provider) buildConfiguration() (*dynamic.Configuration, error) {
	pairs, err := p.kvClient.List(p.RootKey, nil)
	if err != nil {
		return nil, err
	}

	cfg := &dynamic.Configuration{}
	err = kv.Decode(pairs, cfg, p.RootKey)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (p *Provider) createKVClient(ctx context.Context) (store.Store, error) {
	storeConfig := &store.Config{
		ConnectionTimeout: 3 * time.Second,
		Bucket:            "traefik",
		Username:          p.Username,
		Password:          p.Password,
	}

	if p.TLS != nil {
		var err error
		storeConfig.TLS, err = p.TLS.CreateTLSConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	switch p.storeType {
	case store.CONSUL:
		consul.Register()
	case store.ETCDV3:
		etcdv3.Register()
	case store.ZK:
		zookeeper.Register()
	case store.REDIS:
		redis.Register()
	}

	kvStore, err := valkeyrie.NewStore(p.storeType, p.Endpoints, storeConfig)
	if err != nil {
		return nil, err
	}

	return &storeWrapper{Store: kvStore}, nil
}

func (p *Provider) sendConfigLoadErrorToChannel(configurationChan chan<- dynamic.Message) {
	configurationChan <- dynamic.Message{
		ProviderName:       p.name,
		ErrorLoadingConfig: true,
	}
}
