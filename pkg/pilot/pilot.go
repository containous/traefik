package pilot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/traefik/traefik/v2/pkg/anonymize"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/metrics"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/traefik/traefik/v2/pkg/version"
)

const (
	baseInstanceInfoURL = "https://instance-info.pilot.traefik.io/public"
	baseGatewayURL      = "https://gateway.pilot.traefik.io"
)

const (
	tokenHeader     = "X-Token"
	tokenHashHeader = "X-Token-Hash"
)

const (
	pilotInstanceInfoTimer = 5 * time.Minute
	pilotDynConfTimer      = 12 * time.Hour
	maxElapsedTime         = 4 * time.Minute
	initialInterval        = 5 * time.Second
	multiplier             = 3
)

type instanceInfo struct {
	ID      string                `json:"id,omitempty"`
	Metrics []metrics.PilotMetric `json:"metrics,omitempty"`
}

// New creates a new Pilot.
func New(token string, metricsRegistry *metrics.PilotRegistry, pool *safe.Pool) *Pilot {
	tokenHash := fnv.New64a()

	// the `sum64a` implementation of the `Write` method never returns an error.
	_, _ = tokenHash.Write([]byte(token))

	return &Pilot{
		dynamicConfigCh: make(chan dynamic.Configuration),
		client: &client{
			token:               token,
			tokenHash:           fmt.Sprintf("%x", tokenHash.Sum64()),
			httpClient:          &http.Client{Timeout: 5 * time.Second},
			baseInstanceInfoURL: baseInstanceInfoURL,
			baseGatewayURL:      baseGatewayURL,
		},
		routinesPool:    pool,
		metricsRegistry: metricsRegistry,
	}
}

// Pilot connector with Pilot.
type Pilot struct {
	routinesPool *safe.Pool
	client       *client

	dynamicConfig   dynamic.Configuration
	dynamicConfigCh chan dynamic.Configuration
	metricsRegistry *metrics.PilotRegistry
}

// SetDynamicConfiguration stores the dynamic configuration.
func (p *Pilot) SetDynamicConfiguration(dynamicConfig dynamic.Configuration) {
	p.dynamicConfigCh <- dynamicConfig
}

func (p *Pilot) sendAnonDynConf(ctx context.Context, config dynamic.Configuration) {
	err := p.client.SendAnonDynConf(ctx, config)
	if err != nil {
		log.WithoutContext().Error(err)
	}
}

func (p *Pilot) sendInstanceInfo(ctx context.Context, pilotMetrics []metrics.PilotMetric) {
	err := p.client.SendInstanceInfo(ctx, pilotMetrics)
	if err != nil {
		log.WithoutContext().Error(err)
	}
}

// Tick sends data periodically.
func (p *Pilot) Tick(ctx context.Context) {
	pilotMetrics := p.metricsRegistry.Data()

	p.routinesPool.GoCtx(func(ctxRt context.Context) {
		p.sendInstanceInfo(ctxRt, pilotMetrics)
	})

	instanceInfoTicker := time.NewTicker(pilotInstanceInfoTimer)
	dynConfTicker := time.NewTicker(pilotDynConfTimer)

	for {
		select {
		case tick := <-instanceInfoTicker.C:
			log.WithoutContext().Debugf("Send instance info to pilot: %s", tick)

			pilotMetrics := p.metricsRegistry.Data()

			p.routinesPool.GoCtx(func(ctxRt context.Context) {
				p.sendInstanceInfo(ctxRt, pilotMetrics)
			})
		case tick := <-dynConfTicker.C:
			log.WithoutContext().Debugf("Send anonymized dynamic configuration to pilot: %s", tick)

			p.routinesPool.GoCtx(func(ctxRt context.Context) {
				p.sendAnonDynConf(ctxRt, p.dynamicConfig)
			})
		case dynamicConfig := <-p.dynamicConfigCh:
			p.dynamicConfig = dynamicConfig
		case <-ctx.Done():
			return
		}
	}
}

type client struct {
	httpClient          *http.Client
	baseInstanceInfoURL string
	baseGatewayURL      string
	token               string
	tokenHash           string
	uuid                string
}

func (c *client) createUUID() (string, error) {
	data := []byte(`{"version":"` + version.Version + `","codeName":"` + version.Codename + `"}`)
	req, err := http.NewRequest(http.MethodPost, c.baseInstanceInfoURL+"/", bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(tokenHeader, c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed call Pilot: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed read response body: %w", err)
	}

	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("wrong status code while sending configuration: %d: %s", resp.StatusCode, body)
	}

	created := instanceInfo{}
	err = json.Unmarshal(body, &created)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return created.ID, nil
}

// SendAnonDynConf sends anonymized dynamic configuration to Pilot.
func (c *client) SendAnonDynConf(ctx context.Context, config dynamic.Configuration) error {
	anonConfig, err := anonymize.Do(&config, false)
	if err != nil {
		return fmt.Errorf("unable to anonymize dynamic configuration: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseGatewayURL+"/collect", bytes.NewReader([]byte(anonConfig)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	return c.sendDataRetryable(ctx, req)
}

// SendInstanceInfo sends instance information to Pilot.
func (c *client) SendInstanceInfo(ctx context.Context, pilotMetrics []metrics.PilotMetric) error {
	if len(c.uuid) == 0 {
		var err error
		c.uuid, err = c.createUUID()
		if err != nil {
			return fmt.Errorf("failed to create UUID: %w", err)
		}

		version.UUID = c.uuid
	}

	info := instanceInfo{
		ID:      c.uuid,
		Metrics: pilotMetrics,
	}

	b, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshall request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseInstanceInfoURL+"/command", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("failed to create instance info request: %w", err)
	}

	req.Header.Set(tokenHeader, c.token)

	return c.sendDataRetryable(ctx, req)
}

func (c *client) sendDataRetryable(ctx context.Context, req *http.Request) error {
	exponentialBackOff := backoff.NewExponentialBackOff()
	exponentialBackOff.MaxElapsedTime = maxElapsedTime
	exponentialBackOff.InitialInterval = initialInterval
	exponentialBackOff.Multiplier = multiplier

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(tokenHashHeader, c.tokenHash)

	return backoff.RetryNotify(
		func() error {
			resp, err := c.httpClient.Do(req)
			if err != nil {
				return fmt.Errorf("failed to call Pilot: %w", err)
			}

			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response body: %w", err)
			}

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("wrong status code while sending configuration: %d: %s", resp.StatusCode, body)
			}

			return nil
		},
		backoff.WithContext(exponentialBackOff, ctx),
		func(err error, duration time.Duration) {
			log.WithoutContext().Errorf("retry in %s due to: %v ", duration, err)
		})
}
