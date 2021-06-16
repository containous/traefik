// Package headers Middleware based on https://github.com/unrolled/secure.
package headers

import (
	"context"
	"errors"
	"net/http"

	"github.com/opentracing/opentracing-go/ext"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/middlewares"
	"github.com/traefik/traefik/v2/pkg/tracing"
)

const (
	typeName = "Headers"
)

func handleDeprecation(ctx context.Context, cfg *dynamic.Headers) {
	if cfg.SSLRedirect {
		log.FromContext(ctx).Warn("SSLRedirect is deprecated, please use entrypoint redirection instead.")
	}
	if cfg.SSLTemporaryRedirect {
		log.FromContext(ctx).Warn("SSLTemporaryRedirect is deprecated, please use entrypoint redirection instead.")
	}
	if cfg.SSLHost != "" {
		log.FromContext(ctx).Warn("SSLHost is deprecated, please use RedirectRegex middleware instead.")
	}
	if cfg.SSLForceHost {
		log.FromContext(ctx).Warn("SSLForceHost is deprecated, please use RedirectScheme middleware instead.")
	}
}

type headers struct {
	name    string
	handler http.Handler
}

// New creates a Headers middleware.
func New(ctx context.Context, next http.Handler, cfg dynamic.Headers, name string) (http.Handler, error) {
	// HeaderMiddleware -> SecureMiddleWare -> next
	mCtx := middlewares.GetLoggerCtx(ctx, name, typeName)
	logger := log.FromContext(mCtx)
	logger.Debug("Creating middleware")

	handleDeprecation(mCtx, &cfg)

	hasSecureHeaders := cfg.HasSecureHeadersDefined()
	hasCustomHeaders := cfg.HasCustomHeadersDefined()
	hasCorsHeaders := cfg.HasCorsHeadersDefined()

	if !hasSecureHeaders && !hasCustomHeaders && !hasCorsHeaders {
		return nil, errors.New("headers configuration not valid")
	}

	var handler http.Handler
	nextHandler := next

	if hasSecureHeaders {
		logger.Debugf("Setting up secureHeaders from %v", cfg)
		handler = newSecure(next, cfg, name)
		nextHandler = handler
	}

	if hasCustomHeaders || hasCorsHeaders {
		logger.Debugf("Setting up customHeaders/Cors from %v", cfg)
		var err error
		handler, err = NewHeader(nextHandler, cfg)
		if err != nil {
			return nil, err
		}
	}

	return &headers{
		handler: handler,
		name:    name,
	}, nil
}

func (h *headers) GetTracingInformation() (string, ext.SpanKindEnum) {
	return h.name, tracing.SpanKindNoneEnum
}

func (h *headers) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	h.handler.ServeHTTP(rw, req)
}
