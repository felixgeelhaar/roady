package cli

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/plugin"
)

func TestDetectPluginType(t *testing.T) {
	tests := []struct {
		binary   string
		expected string
	}{
		{"./roady-plugin-jira", "jira"},
		{"./roady-plugin-github", "github"},
		{"./roady-plugin-linear", "linear"},
		{"/usr/local/bin/jira-sync", "jira"},
		{"./my-custom-plugin", "custom"},
		{"", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.binary, func(t *testing.T) {
			result := detectPluginType(tt.binary)
			if result != tt.expected {
				t.Errorf("detectPluginType(%q) = %q, want %q", tt.binary, result, tt.expected)
			}
		})
	}
}

func TestFormatLabel(t *testing.T) {
	tests := []struct {
		field    string
		expected string
	}{
		{"api_token", "Api Token"},
		{"domain", "Domain"},
		{"project_key", "Project Key"},
		{"email", "Email"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			result := formatLabel(tt.field)
			if result != tt.expected {
				t.Errorf("formatLabel(%q) = %q, want %q", tt.field, result, tt.expected)
			}
		})
	}
}

func TestToTitleCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"jira", "Jira"},
		{"github", "Github"},
		{"linear", "Linear"},
		{"", ""},
		{"ALREADY", "ALREADY"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toTitleCase(tt.input)
			if result != tt.expected {
				t.Errorf("toTitleCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewConfigureModelEmpty(t *testing.T) {
	m := newConfigureModel("", nil)

	// Should start in plugin selection phase
	if m.phase != phaseSelectPlugin {
		t.Errorf("expected phase to be phaseSelectPlugin, got %d", m.phase)
	}

	// Should not be in editing mode
	if m.editing {
		t.Error("expected editing to be false for new config")
	}
}

func TestNewConfigureModelWithExisting(t *testing.T) {
	existing := &plugin.PluginConfig{
		Binary: "./roady-plugin-jira",
		Config: map[string]string{
			"domain":    "https://test.atlassian.net",
			"api_token": "secret123",
		},
	}

	m := newConfigureModel("test-jira", existing)

	// Should be in editing mode
	if !m.editing {
		t.Error("expected editing to be true for existing config")
	}

	// Should skip to configure phase
	if m.phase != phaseConfigurePlugin {
		t.Errorf("expected phase to be phaseConfigurePlugin, got %d", m.phase)
	}

	// Should have name preset
	if m.inputs[0].Value() != "test-jira" {
		t.Errorf("expected name to be 'test-jira', got %q", m.inputs[0].Value())
	}

	// Should detect jira plugin type
	if m.pluginType != "jira" {
		t.Errorf("expected pluginType to be 'jira', got %q", m.pluginType)
	}
}

func TestConfigureModelValidate(t *testing.T) {
	existing := &plugin.PluginConfig{
		Binary: "./roady-plugin-jira",
		Config: map[string]string{},
	}

	// Test empty name
	m := newConfigureModel("", existing)
	m.inputs[0].SetValue("")

	err := m.validate()
	if err == nil {
		t.Error("expected error for empty name")
	}

	// Test valid config
	m.inputs[0].SetValue("test")

	err = m.validate()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestConfigureModelGetConfig(t *testing.T) {
	existing := &plugin.PluginConfig{
		Binary: "./roady-plugin-jira",
		Config: map[string]string{
			"domain":    "https://test.atlassian.net",
			"api_token": "secret123",
		},
	}

	m := newConfigureModel("test-jira", existing)

	name, config := m.getConfig()

	if name != "test-jira" {
		t.Errorf("expected name 'test-jira', got %q", name)
	}

	if config.Binary != "./roady-plugin-jira" {
		t.Errorf("expected binary './roady-plugin-jira', got %q", config.Binary)
	}
}

func TestSensitiveFields(t *testing.T) {
	sensitiveKeys := []string{"token", "api_token", "api_key", "password", "secret"}

	for _, key := range sensitiveKeys {
		if !sensitiveFields[key] {
			t.Errorf("expected %q to be sensitive", key)
		}
	}

	if sensitiveFields["domain"] {
		t.Error("'domain' should not be sensitive")
	}
}

func TestAvailablePlugins(t *testing.T) {
	// Verify plugins exist
	knownPlugins := []string{"jira", "github", "linear", "asana", "notion", "trello"}

	for _, name := range knownPlugins {
		found := false
		for _, p := range availablePlugins {
			if p.Name == name {
				found = true
				// Verify has config keys
				if len(p.ConfigKeys) == 0 {
					t.Errorf("plugin %q has no config keys", name)
				}
				// Verify has go package
				if p.GoPackage == "" {
					t.Errorf("plugin %q has no go package", name)
				}
				break
			}
		}
		if !found {
			t.Errorf("missing plugin %q", name)
		}
	}
}

func TestGetPlaceholder(t *testing.T) {
	tests := []struct {
		field    string
		expected string
	}{
		{"domain", "https://company.atlassian.net"},
		{"api_token", "your-api-token"},
		{"unknown_field", "Enter unknown_field"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			result := getPlaceholder(tt.field)
			if result != tt.expected {
				t.Errorf("getPlaceholder(%q) = %q, want %q", tt.field, result, tt.expected)
			}
		})
	}
}

func TestGetPluginBinaryPath(t *testing.T) {
	// This test just verifies the function returns a non-empty path
	path := getPluginBinaryPath("jira")
	if path == "" {
		t.Error("expected non-empty path")
	}
	if !contains(path, "roady-plugin-jira") {
		t.Errorf("expected path to contain 'roady-plugin-jira', got %q", path)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
