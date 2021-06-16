package plugins

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// Build builds a middleware plugin.
func (b Builder) Build(pName string, config map[string]interface{}, middlewareName string) (Constructor, error) {
	if b.middlewareDescriptors == nil {
		return nil, fmt.Errorf("no plugin definition in the static configuration: %s", pName)
	}

	descriptor, ok := b.middlewareDescriptors[pName]
	if !ok {
		return nil, fmt.Errorf("unknown plugin type: %s", pName)
	}

	m, err := newMiddleware(descriptor, config, middlewareName)
	if err != nil {
		return nil, err
	}

	return m.NewHandler, err
}

// Middleware is a HTTP handler plugin wrapper.
type Middleware struct {
	middlewareName string
	fnNew          reflect.Value
	config         reflect.Value
}

func newMiddleware(descriptor pluginContext, config map[string]interface{}, middlewareName string) (*Middleware, error) {
	basePkg := descriptor.BasePkg
	if basePkg == "" {
		basePkg = strings.ReplaceAll(path.Base(descriptor.Import), "-", "_")
	}

	vConfig, err := descriptor.interpreter.Eval(basePkg + `.CreateConfig()`)
	if err != nil {
		return nil, fmt.Errorf("failed to eval CreateConfig: %w", err)
	}

	cfg := &mapstructure.DecoderConfig{
		DecodeHook:       mapstructure.StringToSliceHookFunc(","),
		WeaklyTypedInput: true,
		Result:           vConfig.Interface(),
	}

	decoder, err := mapstructure.NewDecoder(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create configuration decoder: %w", err)
	}

	err = decoder.Decode(config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	fnNew, err := descriptor.interpreter.Eval(basePkg + `.New`)
	if err != nil {
		return nil, fmt.Errorf("failed to eval New: %w", err)
	}

	return &Middleware{
		middlewareName: middlewareName,
		fnNew:          fnNew,
		config:         vConfig,
	}, nil
}

// NewHandler creates a new HTTP handler.
func (m *Middleware) NewHandler(ctx context.Context, next http.Handler) (http.Handler, error) {
	args := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(next), m.config, reflect.ValueOf(m.middlewareName)}
	results := m.fnNew.Call(args)

	if len(results) > 1 && results[1].Interface() != nil {
		return nil, results[1].Interface().(error)
	}

	handler, ok := results[0].Interface().(http.Handler)
	if !ok {
		return nil, fmt.Errorf("invalid handler type: %T", results[0].Interface())
	}

	return handler, nil
}
