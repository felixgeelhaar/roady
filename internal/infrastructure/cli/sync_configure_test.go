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

func TestNewConfigureModelEmpty(t *testing.T) {
	m := newConfigureModel("", nil)

	// Should have at least name and binary inputs
	if len(m.inputs) < 2 {
		t.Errorf("expected at least 2 inputs, got %d", len(m.inputs))
	}

	// Should not be in editing mode
	if m.editing {
		t.Error("expected editing to be false for new config")
	}

	// Config fields should start after name and binary
	if m.configStart != 2 {
		t.Errorf("expected configStart to be 2, got %d", m.configStart)
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

	// Should have name preset
	if m.inputs[0].Value() != "test-jira" {
		t.Errorf("expected name to be 'test-jira', got %q", m.inputs[0].Value())
	}

	// Should have binary preset
	if m.inputs[1].Value() != "./roady-plugin-jira" {
		t.Errorf("expected binary to be './roady-plugin-jira', got %q", m.inputs[1].Value())
	}

	// Should detect jira plugin type
	if m.pluginType != "jira" {
		t.Errorf("expected pluginType to be 'jira', got %q", m.pluginType)
	}
}

func TestConfigureModelValidate(t *testing.T) {
	// Test empty name
	m := newConfigureModel("", nil)
	m.inputs[0].SetValue("")
	m.inputs[1].SetValue("./plugin")

	err := m.validate()
	if err == nil {
		t.Error("expected error for empty name")
	}

	// Test empty binary
	m.inputs[0].SetValue("test")
	m.inputs[1].SetValue("")

	err = m.validate()
	if err == nil {
		t.Error("expected error for empty binary")
	}

	// Test valid config
	m.inputs[0].SetValue("test")
	m.inputs[1].SetValue("./plugin")

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

func TestPluginTemplates(t *testing.T) {
	// Verify templates exist for known plugins
	knownPlugins := []string{"jira", "github", "linear", "custom"}

	for _, p := range knownPlugins {
		if _, ok := pluginTemplates[p]; !ok {
			t.Errorf("missing template for plugin %q", p)
		}
	}

	// Verify jira template has expected fields
	jiraFields := pluginTemplates["jira"]
	expectedFields := []string{"domain", "project_key", "email", "api_token"}

	if len(jiraFields) != len(expectedFields) {
		t.Errorf("jira template has %d fields, expected %d", len(jiraFields), len(expectedFields))
	}
}
