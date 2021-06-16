package plugins

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/traefik/traefik/v2/pkg/log"
)

// Setup setup plugins environment.
func Setup(client *Client, plugins map[string]Descriptor, devPlugin *DevPlugin) error {
	err := checkPluginsConfiguration(plugins)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	err = client.CleanArchives(plugins)
	if err != nil {
		return fmt.Errorf("failed to clean archives: %w", err)
	}

	ctx := context.Background()

	for pAlias, desc := range plugins {
		log.FromContext(ctx).Debugf("loading of plugin: %s: %s@%s", pAlias, desc.ModuleName, desc.Version)

		hash, err := client.Download(ctx, desc.ModuleName, desc.Version)
		if err != nil {
			_ = client.ResetAll()
			return fmt.Errorf("failed to download plugin %s: %w", desc.ModuleName, err)
		}

		err = client.Check(ctx, desc.ModuleName, desc.Version, hash)
		if err != nil {
			_ = client.ResetAll()
			return fmt.Errorf("failed to check archive integrity of the plugin %s: %w", desc.ModuleName, err)
		}
	}

	err = client.WriteState(plugins)
	if err != nil {
		_ = client.ResetAll()
		return fmt.Errorf("failed to write plugins state: %w", err)
	}

	for _, desc := range plugins {
		err = client.Unzip(desc.ModuleName, desc.Version)
		if err != nil {
			_ = client.ResetAll()
			return fmt.Errorf("failed to unzip archive: %w", err)
		}
	}

	if devPlugin != nil {
		err := checkDevPluginConfiguration(devPlugin)
		if err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}
	}

	return nil
}

func checkDevPluginConfiguration(plugin *DevPlugin) error {
	if plugin == nil {
		return nil
	}

	if plugin.GoPath == "" {
		return errors.New("missing Go Path (prefer a dedicated Go Path)")
	}

	if plugin.ModuleName == "" {
		return errors.New("missing module name")
	}

	m, err := ReadManifest(plugin.GoPath, plugin.ModuleName)
	if err != nil {
		return err
	}

	switch m.Type {
	case "middleware", "provider":
		// noop
	default:
		return errors.New("unsupported type")
	}

	if m.Import == "" {
		return errors.New("missing import")
	}

	if !strings.HasPrefix(m.Import, plugin.ModuleName) {
		return fmt.Errorf("the import %q must be related to the module name %q", m.Import, plugin.ModuleName)
	}

	if m.DisplayName == "" {
		return errors.New("missing DisplayName")
	}

	if m.Summary == "" {
		return errors.New("missing Summary")
	}

	if m.TestData == nil {
		return errors.New("missing TestData")
	}

	return nil
}

func checkPluginsConfiguration(plugins map[string]Descriptor) error {
	if plugins == nil {
		return nil
	}

	uniq := make(map[string]struct{})

	var errs []string
	for pAlias, descriptor := range plugins {
		if descriptor.ModuleName == "" {
			errs = append(errs, fmt.Sprintf("%s: plugin name is missing", pAlias))
		}

		if descriptor.Version == "" {
			errs = append(errs, fmt.Sprintf("%s: plugin version is missing", pAlias))
		}

		if strings.HasPrefix(descriptor.ModuleName, "/") || strings.HasSuffix(descriptor.ModuleName, "/") {
			errs = append(errs, fmt.Sprintf("%s: plugin name should not start or end with a /", pAlias))
			continue
		}

		if _, ok := uniq[descriptor.ModuleName]; ok {
			errs = append(errs, fmt.Sprintf("only one version of a plugin is allowed, there is a duplicate of %s", descriptor.ModuleName))
			continue
		}

		uniq[descriptor.ModuleName] = struct{}{}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, ": "))
	}

	return nil
}
