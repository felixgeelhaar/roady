package plugin

// PluginConfig represents a named plugin configuration
type PluginConfig struct {
	// Binary is the path to the plugin binary
	Binary string `yaml:"binary" json:"binary"`
	// Config holds the plugin-specific configuration key-value pairs
	Config map[string]string `yaml:"config" json:"config"`
}

// PluginConfigs holds all configured plugins by name
type PluginConfigs struct {
	Plugins map[string]PluginConfig `yaml:"plugins" json:"plugins"`
}

// NewPluginConfigs creates an empty plugin configuration
func NewPluginConfigs() *PluginConfigs {
	return &PluginConfigs{
		Plugins: make(map[string]PluginConfig),
	}
}

// Get returns the plugin configuration for the given name, or nil if not found
func (c *PluginConfigs) Get(name string) *PluginConfig {
	if c.Plugins == nil {
		return nil
	}
	cfg, ok := c.Plugins[name]
	if !ok {
		return nil
	}
	return &cfg
}

// Set adds or updates a plugin configuration
func (c *PluginConfigs) Set(name string, cfg PluginConfig) {
	if c.Plugins == nil {
		c.Plugins = make(map[string]PluginConfig)
	}
	c.Plugins[name] = cfg
}

// Remove deletes a plugin configuration
func (c *PluginConfigs) Remove(name string) {
	if c.Plugins != nil {
		delete(c.Plugins, name)
	}
}

// Names returns all configured plugin names
func (c *PluginConfigs) Names() []string {
	if c.Plugins == nil {
		return nil
	}
	names := make([]string, 0, len(c.Plugins))
	for name := range c.Plugins {
		names = append(names, name)
	}
	return names
}
