package middleware

import (
	"errors"
	"fmt"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/plugins"
)

// PluginsBuilder the plugin's builder interface.
type PluginsBuilder interface {
	Build(pName string, config map[string]interface{}, middlewareName string) (plugins.Constructor, error)
}

func findPluginConfig(rawConfig map[string]dynamic.PluginConf) (string, map[string]interface{}, error) {
	if len(rawConfig) != 1 {
		return "", nil, errors.New("invalid configuration: no configuration or too many plugin definition")
	}

	var pluginType string
	var rawPluginConfig map[string]interface{}

	for pType, pConfig := range rawConfig {
		pluginType = pType
		rawPluginConfig = pConfig
	}

	if pluginType == "" {
		return "", nil, errors.New("missing plugin type")
	}

	if len(rawPluginConfig) == 0 {
		return "", nil, fmt.Errorf("missing plugin configuration: %s", pluginType)
	}

	return pluginType, rawPluginConfig, nil
}
