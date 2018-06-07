package server

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/containous/traefik/configuration"
	"github.com/containous/traefik/healthcheck"
	"github.com/containous/traefik/log"
	"github.com/containous/traefik/middlewares"
	"github.com/containous/traefik/middlewares/accesslog"
	traefiktls "github.com/containous/traefik/tls"
	"github.com/containous/traefik/types"
	"github.com/vulcand/oxy/buffer"
	"github.com/vulcand/oxy/ratelimit"
	"github.com/vulcand/oxy/roundrobin"
	"github.com/vulcand/oxy/utils"
)

func (s *Server) configureLBServers(lb healthcheck.BalancerHandler, config *types.Configuration, frontend *types.Frontend) error {
	for name, srv := range config.Backends[frontend.Backend].Servers {
		u, err := url.Parse(srv.URL)
		if err != nil {
			log.Errorf("Error parsing server URL %s: %v", srv.URL, err)
			return err
		}
		log.Debugf("Creating server %s at %s with weight %d", name, u, srv.Weight)
		if err := lb.UpsertServer(u, roundrobin.Weight(srv.Weight)); err != nil {
			log.Errorf("Error adding server %s to load balancer: %v", srv.URL, err)
			return err
		}
		s.metricsRegistry.BackendServerUpGauge().With("backend", frontend.Backend, "url", srv.URL).Set(1)
	}
	return nil
}

// getRoundTripper will either use server.defaultForwardingRoundTripper or create a new one
// given a custom TLS configuration is passed and the passTLSCert option is set to true.
func (s *Server) getRoundTripper(entryPointName string, globalConfiguration configuration.GlobalConfiguration, passTLSCert bool, tls *traefiktls.TLS) (http.RoundTripper, error) {
	if passTLSCert {
		tlsConfig, err := createClientTLSConfig(entryPointName, tls)
		if err != nil {
			log.Errorf("Failed to create TLSClientConfig: %s", err)
			return nil, err
		}

		transport := createHTTPTransport(globalConfiguration)
		transport.TLSClientConfig = tlsConfig
		return transport, nil
	}

	return s.defaultForwardingRoundTripper, nil
}

func createClientTLSConfig(entryPointName string, tlsOption *traefiktls.TLS) (*tls.Config, error) {
	if tlsOption == nil {
		return nil, errors.New("no TLS provided")
	}

	config, err := tlsOption.Certificates.CreateTLSConfig(entryPointName)
	if err != nil {
		return nil, err
	}

	if len(tlsOption.ClientCAFiles) > 0 {
		log.Warnf("Deprecated configuration found during client TLS configuration creation: %s. Please use %s (which allows to make the CA Files optional).", "tls.ClientCAFiles", "tls.ClientCA.files")
		tlsOption.ClientCA.Files = tlsOption.ClientCAFiles
		tlsOption.ClientCA.Optional = false
	}
	if len(tlsOption.ClientCA.Files) > 0 {
		pool := x509.NewCertPool()
		for _, caFile := range tlsOption.ClientCA.Files {
			data, err := ioutil.ReadFile(caFile)
			if err != nil {
				return nil, err
			}
			if !pool.AppendCertsFromPEM(data) {
				return nil, errors.New("invalid certificate(s) in " + caFile)
			}
		}
		config.RootCAs = pool
	}
	config.BuildNameToCertificate()
	return config, nil
}

func (s *Server) buildRetryMiddleware(handler http.Handler, globalConfig configuration.GlobalConfiguration, countServers int, backendName string) http.Handler {
	retryListeners := middlewares.RetryListeners{}
	if s.metricsRegistry.IsEnabled() {
		retryListeners = append(retryListeners, middlewares.NewMetricsRetryListener(s.metricsRegistry, backendName))
	}
	if s.accessLoggerMiddleware != nil {
		retryListeners = append(retryListeners, &accesslog.SaveRetries{})
	}

	retryAttempts := countServers
	if globalConfig.Retry.Attempts > 0 {
		retryAttempts = globalConfig.Retry.Attempts
	}

	log.Debugf("Creating retries max attempts %d", retryAttempts)

	return s.tracingMiddleware.NewHTTPHandlerWrapper("Retry", middlewares.NewRetry(retryAttempts, handler, retryListeners), false)
}

func (s *Server) buildRateLimiter(handler http.Handler, rlConfig *types.RateLimit) (http.Handler, error) {
	extractFunc, err := utils.NewExtractor(rlConfig.ExtractorFunc)
	if err != nil {
		return nil, err
	}
	log.Debugf("Creating load-balancer rate limiter")
	rateSet := ratelimit.NewRateSet()
	for _, rate := range rlConfig.RateSet {
		if err := rateSet.Add(time.Duration(rate.Period), rate.Average, rate.Burst); err != nil {
			return nil, err
		}
	}
	rateLimiter, err := ratelimit.New(handler, extractFunc, rateSet)
	return s.tracingMiddleware.NewHTTPHandlerWrapper("Rate limit", rateLimiter, false), err

}

func (s *Server) buildBufferingMiddleware(handler http.Handler, config *types.Buffering) (http.Handler, error) {
	log.Debugf("Setting up buffering: request limits: %d (mem), %d (max), response limits: %d (mem), %d (max) with retry: '%s'",
		config.MemRequestBodyBytes, config.MaxRequestBodyBytes, config.MemResponseBodyBytes,
		config.MaxResponseBodyBytes, config.RetryExpression)

	return buffer.New(
		handler,
		buffer.MemRequestBodyBytes(config.MemRequestBodyBytes),
		buffer.MaxRequestBodyBytes(config.MaxRequestBodyBytes),
		buffer.MemResponseBodyBytes(config.MemResponseBodyBytes),
		buffer.MaxResponseBodyBytes(config.MaxResponseBodyBytes),
		buffer.CondSetter(len(config.RetryExpression) > 0, buffer.Retry(config.RetryExpression)),
	)
}
