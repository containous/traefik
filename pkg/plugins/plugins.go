package plugins

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/traefik/traefik/v2/pkg/log"
)

const localGoPath = "./plugins-local/"

// SetupRemotePlugins setup remote plugins environment.
func SetupRemotePlugins(client *Client, plugins map[string]Descriptor) error {
	err := checkRemotePluginsConfiguration(plugins)
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

	return nil
}

func checkRemotePluginsConfiguration(plugins map[string]Descriptor) error {
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

// SetupLocalPlugins setup local plugins environment.
func SetupLocalPlugins(plugins map[string]LocalDescriptor) error {
	if plugins == nil {
		return nil
	}

	uniq := make(map[string]struct{})

	var errs *multierror.Error
	for pAlias, descriptor := range plugins {
		if descriptor.ModuleName == "" {
			errs = multierror.Append(errs, fmt.Errorf("%s: plugin name is missing", pAlias))
		}

		if strings.HasPrefix(descriptor.ModuleName, "/") || strings.HasSuffix(descriptor.ModuleName, "/") {
			errs = multierror.Append(errs, fmt.Errorf("%s: plugin name should not start or end with a /", pAlias))
			continue
		}

		if _, ok := uniq[descriptor.ModuleName]; ok {
			errs = multierror.Append(errs, fmt.Errorf("only one version of a plugin is allowed, there is a duplicate of %s", descriptor.ModuleName))
			continue
		}

		uniq[descriptor.ModuleName] = struct{}{}

		err := checkLocalPluginManifest(descriptor)
		errs = multierror.Append(errs, err)
	}

	return errs.ErrorOrNil()
}

func checkLocalPluginManifest(descriptor LocalDescriptor) error {
	m, err := ReadManifest(localGoPath, descriptor.ModuleName)
	if err != nil {
		return err
	}

	var errs *multierror.Error

	switch m.Type {
	case "middleware", "provider":
		// noop
	default:
		errs = multierror.Append(errs, fmt.Errorf("%s: unsupported type %q", descriptor.ModuleName, m.Type))
	}

	if m.Import == "" {
		errs = multierror.Append(errs, fmt.Errorf("%s: missing import", descriptor.ModuleName))
	}

	if !strings.HasPrefix(m.Import, descriptor.ModuleName) {
		errs = multierror.Append(errs, fmt.Errorf("the import %q must be related to the module name %q", m.Import, descriptor.ModuleName))
	}

	if m.DisplayName == "" {
		errs = multierror.Append(errs, fmt.Errorf("%s: missing DisplayName", descriptor.ModuleName))
	}

	if m.Summary == "" {
		errs = multierror.Append(errs, fmt.Errorf("%s: missing Summary", descriptor.ModuleName))
	}

	if m.TestData == nil {
		errs = multierror.Append(errs, fmt.Errorf("%s: missing TestData", descriptor.ModuleName))
	}

	return errs.ErrorOrNil()
}
