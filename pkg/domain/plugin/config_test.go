package plugin_test

import (
	"sort"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/plugin"
)

func TestNewPluginConfigs(t *testing.T) {
	configs := plugin.NewPluginConfigs()
	if configs == nil {
		t.Fatal("expected non-nil PluginConfigs")
	}
	if configs.Plugins == nil {
		t.Fatal("expected initialized Plugins map")
	}
	if len(configs.Plugins) != 0 {
		t.Errorf("expected empty Plugins map, got %d entries", len(configs.Plugins))
	}
}

func TestPluginConfigs_Get(t *testing.T) {
	t.Run("returns nil when plugins map is nil", func(t *testing.T) {
		configs := &plugin.PluginConfigs{Plugins: nil}
		got := configs.Get("anything")
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("returns nil for non-existent plugin", func(t *testing.T) {
		configs := plugin.NewPluginConfigs()
		got := configs.Get("nonexistent")
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("returns config for existing plugin", func(t *testing.T) {
		configs := plugin.NewPluginConfigs()
		configs.Plugins["github"] = plugin.PluginConfig{
			Binary: "/usr/local/bin/roady-plugin-github",
			Config: map[string]string{"token": "abc123"},
		}

		got := configs.Get("github")
		if got == nil {
			t.Fatal("expected non-nil config")
		}
		if got.Binary != "/usr/local/bin/roady-plugin-github" {
			t.Errorf("expected binary path, got %s", got.Binary)
		}
		if got.Config["token"] != "abc123" {
			t.Errorf("expected token abc123, got %s", got.Config["token"])
		}
	})
}

func TestPluginConfigs_Set(t *testing.T) {
	t.Run("adds new plugin to initialized map", func(t *testing.T) {
		configs := plugin.NewPluginConfigs()
		cfg := plugin.PluginConfig{
			Binary: "/usr/local/bin/roady-plugin-jira",
			Config: map[string]string{"url": "https://jira.example.com"},
		}

		configs.Set("jira", cfg)

		got := configs.Get("jira")
		if got == nil {
			t.Fatal("expected non-nil config after Set")
		}
		if got.Binary != "/usr/local/bin/roady-plugin-jira" {
			t.Errorf("expected jira binary, got %s", got.Binary)
		}
	})

	t.Run("initializes nil map before setting", func(t *testing.T) {
		configs := &plugin.PluginConfigs{Plugins: nil}
		cfg := plugin.PluginConfig{Binary: "/bin/test"}

		configs.Set("test", cfg)

		if configs.Plugins == nil {
			t.Fatal("expected Plugins map to be initialized")
		}
		got := configs.Get("test")
		if got == nil {
			t.Fatal("expected config after Set on nil map")
		}
	})

	t.Run("overwrites existing plugin", func(t *testing.T) {
		configs := plugin.NewPluginConfigs()
		configs.Set("github", plugin.PluginConfig{Binary: "/old/path"})
		configs.Set("github", plugin.PluginConfig{Binary: "/new/path"})

		got := configs.Get("github")
		if got == nil {
			t.Fatal("expected non-nil config")
		}
		if got.Binary != "/new/path" {
			t.Errorf("expected updated binary, got %s", got.Binary)
		}
	})
}

func TestPluginConfigs_Remove(t *testing.T) {
	t.Run("removes existing plugin", func(t *testing.T) {
		configs := plugin.NewPluginConfigs()
		configs.Set("github", plugin.PluginConfig{Binary: "/bin/github"})

		configs.Remove("github")

		got := configs.Get("github")
		if got != nil {
			t.Errorf("expected nil after remove, got %v", got)
		}
	})

	t.Run("no-op on nil map", func(t *testing.T) {
		configs := &plugin.PluginConfigs{Plugins: nil}
		// Should not panic
		configs.Remove("anything")
	})

	t.Run("no-op for non-existent plugin", func(t *testing.T) {
		configs := plugin.NewPluginConfigs()
		configs.Set("github", plugin.PluginConfig{Binary: "/bin/github"})

		configs.Remove("nonexistent")

		if len(configs.Plugins) != 1 {
			t.Errorf("expected 1 plugin remaining, got %d", len(configs.Plugins))
		}
	})
}

func TestPluginConfigs_Names(t *testing.T) {
	t.Run("returns nil for nil map", func(t *testing.T) {
		configs := &plugin.PluginConfigs{Plugins: nil}
		names := configs.Names()
		if names != nil {
			t.Errorf("expected nil, got %v", names)
		}
	})

	t.Run("returns empty for empty map", func(t *testing.T) {
		configs := plugin.NewPluginConfigs()
		names := configs.Names()
		if len(names) != 0 {
			t.Errorf("expected 0 names, got %d", len(names))
		}
	})

	t.Run("returns all plugin names", func(t *testing.T) {
		configs := plugin.NewPluginConfigs()
		configs.Set("github", plugin.PluginConfig{Binary: "/bin/github"})
		configs.Set("jira", plugin.PluginConfig{Binary: "/bin/jira"})
		configs.Set("linear", plugin.PluginConfig{Binary: "/bin/linear"})

		names := configs.Names()
		if len(names) != 3 {
			t.Fatalf("expected 3 names, got %d", len(names))
		}

		sort.Strings(names)
		expected := []string{"github", "jira", "linear"}
		for i, name := range names {
			if name != expected[i] {
				t.Errorf("expected name %s at index %d, got %s", expected[i], i, name)
			}
		}
	})
}
