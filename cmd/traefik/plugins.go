package main

import (
	"fmt"

	"github.com/traefik/traefik/v2/pkg/config/static"
	"github.com/traefik/traefik/v2/pkg/plugins"
)

const outputDir = "./plugins-storage/"

func createPluginBuilder(staticConfiguration *static.Configuration) (*plugins.Builder, error) {
	client, plgs, localPlgs, err := initPlugins(staticConfiguration)
	if err != nil {
		return nil, err
	}

	return plugins.NewBuilder(client, plgs, localPlgs)
}

func initPlugins(staticCfg *static.Configuration) (*plugins.Client, map[string]plugins.Descriptor, map[string]plugins.LocalDescriptor, error) {
	err := checkUniquePluginNames(staticCfg.Experimental)
	if err != nil {
		return nil, nil, nil, err
	}

	var client *plugins.Client
	plgs := map[string]plugins.Descriptor{}

	if isPilotEnabled(staticCfg) && hasPlugins(staticCfg) {
		opts := plugins.ClientOptions{
			Output: outputDir,
			Token:  staticCfg.Pilot.Token,
		}

		var err error
		client, err = plugins.NewClient(opts)
		if err != nil {
			return nil, nil, nil, err
		}

		err = plugins.SetupRemotePlugins(client, staticCfg.Experimental.Plugins)
		if err != nil {
			return nil, nil, nil, err
		}

		plgs = staticCfg.Experimental.Plugins
	}

	localPlgs := map[string]plugins.LocalDescriptor{}

	if hasLocalPlugins(staticCfg) {
		err := plugins.SetupLocalPlugins(staticCfg.Experimental.LocalPlugins)
		if err != nil {
			return nil, nil, nil, err
		}

		localPlgs = staticCfg.Experimental.LocalPlugins
	}

	return client, plgs, localPlgs, nil
}

func checkUniquePluginNames(e *static.Experimental) error {
	if e == nil {
		return nil
	}

	for s := range e.LocalPlugins {
		if _, ok := e.Plugins[s]; ok {
			return fmt.Errorf("the plugin's name %q must be unique", s)
		}
	}

	return nil
}

func isPilotEnabled(staticCfg *static.Configuration) bool {
	return staticCfg.Pilot != nil && staticCfg.Pilot.Token != ""
}

func hasPlugins(staticCfg *static.Configuration) bool {
	return staticCfg.Experimental != nil && len(staticCfg.Experimental.Plugins) > 0
}

func hasLocalPlugins(staticCfg *static.Configuration) bool {
	return staticCfg.Experimental != nil && len(staticCfg.Experimental.LocalPlugins) > 0
}
