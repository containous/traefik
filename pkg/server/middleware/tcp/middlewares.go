package tcpmiddleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/traefik/traefik/v2/pkg/config/runtime"
	ipwhitelist "github.com/traefik/traefik/v2/pkg/middlewares/tcp/ipwhitelist"
	"github.com/traefik/traefik/v2/pkg/server/provider"
	"github.com/traefik/traefik/v2/pkg/tcp"
)

type middlewareStackType int

const (
	middlewareStackKey middlewareStackType = iota
)

// Builder the middleware builder.
type Builder struct {
	configs map[string]*runtime.TCPMiddlewareInfo
}

// NewBuilder creates a new Builder.
func NewBuilder(configs map[string]*runtime.TCPMiddlewareInfo) *Builder {
	return &Builder{configs: configs}
}

// BuildChain creates a middleware chain.
func (b *Builder) BuildChain(ctx context.Context, middlewares []string) *tcp.Chain {
	chain := tcp.NewChain()

	for _, name := range middlewares {
		middlewareName := provider.GetQualifiedName(ctx, name)

		chain = chain.Append(func(next tcp.Handler) (tcp.Handler, error) {
			constructorContext := provider.AddInContext(ctx, middlewareName)
			if midInf, ok := b.configs[middlewareName]; !ok || midInf.TCPMiddleware == nil {
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

func (b *Builder) buildConstructor(ctx context.Context, middlewareName string) (tcp.Constructor, error) {
	config := b.configs[middlewareName]
	if config == nil || config.TCPMiddleware == nil {
		return nil, fmt.Errorf("invalid middleware %q configuration", middlewareName)
	}

	var middleware tcp.Constructor

	// IPWhiteList
	if config.IPWhiteList != nil {
		middleware = func(next tcp.Handler) (tcp.Handler, error) {
			return ipwhitelist.New(ctx, next, *config.IPWhiteList, middlewareName)
		}
	}

	if middleware == nil {
		return nil, fmt.Errorf("invalid middleware %q configuration: invalid middleware type or middleware does not exist", middlewareName)
	}

	return middleware, nil
}

func inSlice(element string, stack []string) bool {
	for _, value := range stack {
		if value == element {
			return true
		}
	}
	return false
}
