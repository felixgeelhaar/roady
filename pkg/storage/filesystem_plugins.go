package storage

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/pkg/domain/plugin"
	"gopkg.in/yaml.v3"
)

const PluginsFile = "plugins.yaml"

// SavePluginConfigs saves the plugin configurations to plugins.yaml
func (r *FilesystemRepository) SavePluginConfigs(configs *plugin.PluginConfigs) error {
	path, err := r.ResolvePath(PluginsFile)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(configs)
	if err != nil {
		return fmt.Errorf("failed to marshal plugin configs: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// LoadPluginConfigs loads plugin configurations from plugins.yaml
func (r *FilesystemRepository) LoadPluginConfigs() (*plugin.PluginConfigs, error) {
	path, err := r.ResolvePath(PluginsFile)
	if err != nil {
		return nil, err
	}

	// Return empty config if file doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return plugin.NewPluginConfigs(), nil
	}

	// #nosec G304 -- Path is resolved and validated via ResolvePath
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugins file: %w", err)
	}

	var configs plugin.PluginConfigs
	if err := yaml.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plugin configs: %w", err)
	}

	// Initialize nil map
	if configs.Plugins == nil {
		configs.Plugins = make(map[string]plugin.PluginConfig)
	}

	return &configs, nil
}

// GetPluginConfig returns a specific plugin configuration by name
func (r *FilesystemRepository) GetPluginConfig(name string) (*plugin.PluginConfig, error) {
	configs, err := r.LoadPluginConfigs()
	if err != nil {
		return nil, err
	}

	cfg := configs.Get(name)
	if cfg == nil {
		return nil, fmt.Errorf("plugin config not found: %s", name)
	}

	return cfg, nil
}

// SetPluginConfig adds or updates a plugin configuration
func (r *FilesystemRepository) SetPluginConfig(name string, cfg plugin.PluginConfig) error {
	configs, err := r.LoadPluginConfigs()
	if err != nil {
		return err
	}

	configs.Set(name, cfg)
	return r.SavePluginConfigs(configs)
}

// RemovePluginConfig removes a plugin configuration
func (r *FilesystemRepository) RemovePluginConfig(name string) error {
	configs, err := r.LoadPluginConfigs()
	if err != nil {
		return err
	}

	configs.Remove(name)
	return r.SavePluginConfigs(configs)
}
