package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/containous/alice"
	"github.com/traefik/traefik/v2/pkg/config/runtime"
	"github.com/traefik/traefik/v2/pkg/middlewares/addprefix"
	"github.com/traefik/traefik/v2/pkg/middlewares/auth"
	"github.com/traefik/traefik/v2/pkg/middlewares/buffering"
	"github.com/traefik/traefik/v2/pkg/middlewares/chain"
	"github.com/traefik/traefik/v2/pkg/middlewares/circuitbreaker"
	"github.com/traefik/traefik/v2/pkg/middlewares/compress"
	"github.com/traefik/traefik/v2/pkg/middlewares/customerrors"
	"github.com/traefik/traefik/v2/pkg/middlewares/headers"
	"github.com/traefik/traefik/v2/pkg/middlewares/inflightreq"
	"github.com/traefik/traefik/v2/pkg/middlewares/ipwhitelist"
	"github.com/traefik/traefik/v2/pkg/middlewares/passtlsclientcert"
	"github.com/traefik/traefik/v2/pkg/middlewares/ratelimiter"
	"github.com/traefik/traefik/v2/pkg/middlewares/redirect"
	"github.com/traefik/traefik/v2/pkg/middlewares/replacepath"
	"github.com/traefik/traefik/v2/pkg/middlewares/replacepathregex"
	"github.com/traefik/traefik/v2/pkg/middlewares/retry"
	"github.com/traefik/traefik/v2/pkg/middlewares/stripprefix"
	"github.com/traefik/traefik/v2/pkg/middlewares/stripprefixregex"
	"github.com/traefik/traefik/v2/pkg/middlewares/tracing"
	"github.com/traefik/traefik/v2/pkg/server/provider"
)

type middlewareStackType int

const (
	middlewareStackKey middlewareStackType = iota
)

// Builder the middleware builder.
type Builder struct {
	configs        map[string]*runtime.MiddlewareInfo
	pluginBuilder  PluginsBuilder
	serviceBuilder serviceBuilder
}

type serviceBuilder interface {
	BuildHTTP(ctx context.Context, serviceName string) (http.Handler, error)
}

// NewBuilder creates a new Builder.
func NewBuilder(configs map[string]*runtime.MiddlewareInfo, serviceBuilder serviceBuilder, pluginBuilder PluginsBuilder) *Builder {
	return &Builder{configs: configs, serviceBuilder: serviceBuilder, pluginBuilder: pluginBuilder}
}

// BuildChain creates a middleware chain.
func (b *Builder) BuildChain(ctx context.Context, middlewares []string) *alice.Chain {
	chain := alice.New()
	for _, name := range middlewares {
		middlewareName := provider.GetQualifiedName(ctx, name)

		chain = chain.Append(func(next http.Handler) (http.Handler, error) {
			constructorContext := provider.AddInContext(ctx, middlewareName)
			if midInf, ok := b.configs[middlewareName]; !ok || midInf.Middleware == nil {
				return nil, fmt.Errorf("middleware %q does not exist", middlewareName)
			}

			var err error
			if constructorContext, err = checkRecursion(constructorContext, middlewareName); err != nil {
				b.configs[middlewareName].AddError(err, true)
				return nil, err
			}

			constructor, err := b.buildConstructor(constructorContext, middlewareName)
			if err != nil {
				b.configs[middlewareName].AddError(err, true)
				return nil, err
			}

			handler, err := constructor(next)
			if err != nil {
				b.configs[middlewareName].AddError(err, true)
				return nil, err
			}

			return handler, nil
		})
	}
	return &chain
}

func checkRecursion(ctx context.Context, middlewareName string) (context.Context, error) {
	currentStack, ok := ctx.Value(middlewareStackKey).([]string)
	if !ok {
		currentStack = []string{}
	}
	if inSlice(middlewareName, currentStack) {
		return ctx, fmt.Errorf("could not instantiate middleware %s: recursion detected in %s", middlewareName, strings.Join(append(currentStack, middlewareName), "->"))
	}
	return context.WithValue(ctx, middlewareStackKey, append(currentStack, middlewareName)), nil
}

// it is the responsibility of the caller to make sure that b.configs[middlewareName].Middleware exists.
func (b *Builder) buildConstructor(ctx context.Context, middlewareName string) (alice.Constructor, error) {
	config := b.configs[middlewareName]
	if config == nil || config.Middleware == nil {
		return nil, fmt.Errorf("invalid middleware %q configuration", middlewareName)
	}

	var middleware alice.Constructor
	badConf := errors.New("cannot create middleware: multi-types middleware not supported, consider declaring two different pieces of middleware instead")

	// AddPrefix
	if config.AddPrefix != nil {
		middleware = func(next http.Handler) (http.Handler, error) {
			return addprefix.New(ctx, next, *config.AddPrefix, middlewareName)
		}
	}

	// BasicAuth
	if config.BasicAuth != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return auth.NewBasic(ctx, next, *config.BasicAuth, middlewareName)
		}
	}

	// Buffering
	if config.Buffering != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return buffering.New(ctx, next, *config.Buffering, middlewareName)
		}
	}

	// Chain
	if config.Chain != nil {
		if middleware != nil {
			return nil, badConf
		}

		var qualifiedNames []string
		for _, name := range config.Chain.Middlewares {
			qualifiedNames = append(qualifiedNames, provider.GetQualifiedName(ctx, name))
		}
		config.Chain.Middlewares = qualifiedNames
		middleware = func(next http.Handler) (http.Handler, error) {
			return chain.New(ctx, next, *config.Chain, b, middlewareName)
		}
	}

	// CircuitBreaker
	if config.CircuitBreaker != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return circuitbreaker.New(ctx, next, *config.CircuitBreaker, middlewareName)
		}
	}

	// Compress
	if config.Compress != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return compress.New(ctx, next, *config.Compress, middlewareName)
		}
	}

	// ContentType
	if config.ContentType != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if !config.ContentType.AutoDetect {
					rw.Header()["Content-Type"] = nil
				}
				next.ServeHTTP(rw, req)
			}), nil
		}
	}

	// CustomErrors
	if config.Errors != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return customerrors.New(ctx, next, *config.Errors, b.serviceBuilder, middlewareName)
		}
	}

	// DigestAuth
	if config.DigestAuth != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return auth.NewDigest(ctx, next, *config.DigestAuth, middlewareName)
		}
	}

	// ForwardAuth
	if config.ForwardAuth != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return auth.NewForward(ctx, next, *config.ForwardAuth, middlewareName)
		}
	}

	// Headers
	if config.Headers != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return headers.New(ctx, next, *config.Headers, middlewareName)
		}
	}

	// IPWhiteList
	if config.IPWhiteList != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return ipwhitelist.New(ctx, next, *config.IPWhiteList, middlewareName)
		}
	}

	// InFlightReq
	if config.InFlightReq != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return inflightreq.New(ctx, next, *config.InFlightReq, middlewareName)
		}
	}

	// PassTLSClientCert
	if config.PassTLSClientCert != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return passtlsclientcert.New(ctx, next, *config.PassTLSClientCert, middlewareName)
		}
	}

	// RateLimit
	if config.RateLimit != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return ratelimiter.New(ctx, next, *config.RateLimit, middlewareName)
		}
	}

	// RedirectRegex
	if config.RedirectRegex != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return redirect.NewRedirectRegex(ctx, next, *config.RedirectRegex, middlewareName)
		}
	}

	// RedirectScheme
	if config.RedirectScheme != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return redirect.NewRedirectScheme(ctx, next, *config.RedirectScheme, middlewareName)
		}
	}

	// ReplacePath
	if config.ReplacePath != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return replacepath.New(ctx, next, *config.ReplacePath, middlewareName)
		}
	}

	// ReplacePathRegex
	if config.ReplacePathRegex != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return replacepathregex.New(ctx, next, *config.ReplacePathRegex, middlewareName)
		}
	}

	// Retry
	if config.Retry != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			// FIXME missing metrics / accessLog
			return retry.New(ctx, next, *config.Retry, retry.Listeners{}, middlewareName)
		}
	}

	// StripPrefix
	if config.StripPrefix != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return stripprefix.New(ctx, next, *config.StripPrefix, middlewareName)
		}
	}

	// StripPrefixRegex
	if config.StripPrefixRegex != nil {
		if middleware != nil {
			return nil, badConf
		}
		middleware = func(next http.Handler) (http.Handler, error) {
			return stripprefixregex.New(ctx, next, *config.StripPrefixRegex, middlewareName)
		}
	}

	// Plugin
	if config.Plugin != nil {
		if middleware != nil {
			return nil, badConf
		}

		pluginType, rawPluginConfig, err := findPluginConfig(config.Plugin)
		if err != nil {
			return nil, fmt.Errorf("plugin: %w", err)
		}

		plug, err := b.pluginBuilder.Build(pluginType, rawPluginConfig, middlewareName)
		if err != nil {
			return nil, fmt.Errorf("plugin: %w", err)
		}

		middleware = func(next http.Handler) (http.Handler, error) {
			return plug(ctx, next)
		}
	}

	if middleware == nil {
		return nil, fmt.Errorf("invalid middleware %q configuration: invalid middleware type or middleware does not exist", middlewareName)
	}

	return tracing.Wrap(ctx, middleware), nil
}

func inSlice(element string, stack []string) bool {
	for _, value := range stack {
		if value == element {
			return true
		}
	}
	return false
}
