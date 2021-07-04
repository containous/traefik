package plugins

import (
	"context"
	"fmt"
	"net/http"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// Constructor creates a plugin handler.
type Constructor func(context.Context, http.Handler) (http.Handler, error)

// pluginContext The static part of a plugin configuration.
type pluginContext struct {
	// GoPath plugin's GOPATH
	GoPath string `json:"goPath,omitempty" toml:"goPath,omitempty" yaml:"goPath,omitempty"`

	// Import plugin's import/package
	Import string `json:"import,omitempty" toml:"import,omitempty" yaml:"import,omitempty"`

	// BasePkg plugin's base package name (optional)
	BasePkg string `json:"basePkg,omitempty" toml:"basePkg,omitempty" yaml:"basePkg,omitempty"`

	interpreter *interp.Interpreter
}

// Builder is a plugin builder.
type Builder struct {
	middlewareDescriptors map[string]pluginContext
	providerDescriptors   map[string]pluginContext
}

// NewBuilder creates a new Builder.
func NewBuilder(client *Client, plugins map[string]Descriptor, localPlugins map[string]LocalDescriptor) (*Builder, error) {
	pb := &Builder{
		middlewareDescriptors: map[string]pluginContext{},
		providerDescriptors:   map[string]pluginContext{},
	}

	for pName, desc := range plugins {
		manifest, err := client.ReadManifest(desc.ModuleName)
		if err != nil {
			_ = client.ResetAll()
			return nil, fmt.Errorf("%s: failed to read manifest: %w", desc.ModuleName, err)
		}

		i := interp.New(interp.Options{GoPath: client.GoPath()})

		err = i.Use(stdlib.Symbols)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to load symbols: %w", desc.ModuleName, err)
		}

		err = i.Use(ppSymbols())
		if err != nil {
			return nil, fmt.Errorf("%s: failed to load provider symbols: %w", desc.ModuleName, err)
		}

		_, err = i.Eval(fmt.Sprintf(`import "%s"`, manifest.Import))
		if err != nil {
			return nil, fmt.Errorf("%s: failed to import plugin code %q: %w", desc.ModuleName, manifest.Import, err)
		}

		switch manifest.Type {
		case "middleware":
			pb.middlewareDescriptors[pName] = pluginContext{
				interpreter: i,
				GoPath:      client.GoPath(),
				Import:      manifest.Import,
				BasePkg:     manifest.BasePkg,
			}
		case "provider":
			pb.providerDescriptors[pName] = pluginContext{
				interpreter: i,
				GoPath:      client.GoPath(),
				Import:      manifest.Import,
				BasePkg:     manifest.BasePkg,
			}
		default:
			return nil, fmt.Errorf("unknow plugin type: %s", manifest.Type)
		}
	}

	for pName, desc := range localPlugins {
		manifest, err := ReadManifest(localGoPath, desc.ModuleName)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to read manifest: %w", desc.ModuleName, err)
		}

		i := interp.New(interp.Options{GoPath: localGoPath})

		err = i.Use(stdlib.Symbols)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to load symbols: %w", desc.ModuleName, err)
		}

		err = i.Use(ppSymbols())
		if err != nil {
			return nil, fmt.Errorf("%s: failed to load provider symbols: %w", desc.ModuleName, err)
		}

		_, err = i.Eval(fmt.Sprintf(`import "%s"`, manifest.Import))
		if err != nil {
			return nil, fmt.Errorf("%s: failed to import plugin code %q: %w", desc.ModuleName, manifest.Import, err)
		}

		switch manifest.Type {
		case "middleware":
			pb.middlewareDescriptors[pName] = pluginContext{
				interpreter: i,
				GoPath:      localGoPath,
				Import:      manifest.Import,
				BasePkg:     manifest.BasePkg,
			}
		case "provider":
			pb.providerDescriptors[pName] = pluginContext{
				interpreter: i,
				GoPath:      localGoPath,
				Import:      manifest.Import,
				BasePkg:     manifest.BasePkg,
			}
		default:
			return nil, fmt.Errorf("unknow plugin type: %s", manifest.Type)
		}
	}

	return pb, nil
}
