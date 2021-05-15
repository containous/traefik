package plugins

import (
	"context"
	"fmt"
	"net/http"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

const devPluginName = "dev"

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
func NewBuilder(client *Client, plugins map[string]Descriptor, devPlugin *DevPlugin) (*Builder, error) {
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
		i.Use(stdlib.Symbols)
		i.Use(ppSymbols())

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

	if devPlugin != nil {
		manifest, err := ReadManifest(devPlugin.GoPath, devPlugin.ModuleName)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to read manifest: %w", devPlugin.ModuleName, err)
		}

		i := interp.New(interp.Options{GoPath: devPlugin.GoPath})
		i.Use(stdlib.Symbols)
		i.Use(ppSymbols())

		_, err = i.Eval(fmt.Sprintf(`import "%s"`, manifest.Import))
		if err != nil {
			return nil, fmt.Errorf("%s: failed to import plugin code %q: %w", devPlugin.ModuleName, manifest.Import, err)
		}

		switch manifest.Type {
		case "middleware":
			pb.middlewareDescriptors[devPluginName] = pluginContext{
				interpreter: i,
				GoPath:      devPlugin.GoPath,
				Import:      manifest.Import,
				BasePkg:     manifest.BasePkg,
			}
		case "provider":
			pb.providerDescriptors[devPluginName] = pluginContext{
				interpreter: i,
				GoPath:      devPlugin.GoPath,
				Import:      manifest.Import,
				BasePkg:     manifest.BasePkg,
			}
		default:
			return nil, fmt.Errorf("unknow plugin type: %s", manifest.Type)
		}
	}

	return pb, nil
}
