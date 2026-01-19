package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
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

	if m.phase != phaseSelectPlugin {
		t.Errorf("expected phase to be phaseSelectPlugin, got %d", m.phase)
	}

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

	if !m.editing {
		t.Error("expected editing to be true for existing config")
	}

	if m.phase != phaseConfigurePlugin {
		t.Errorf("expected phase to be phaseConfigurePlugin, got %d", m.phase)
	}

	if m.inputs[0].Value() != "test-jira" {
		t.Errorf("expected name to be 'test-jira', got %q", m.inputs[0].Value())
	}

	if m.pluginType != "jira" {
		t.Errorf("expected pluginType to be 'jira', got %q", m.pluginType)
	}
}

func TestConfigureModelValidate(t *testing.T) {
	existing := &plugin.PluginConfig{
		Binary: "./roady-plugin-jira",
		Config: map[string]string{},
	}

	m := newConfigureModel("", existing)
	m.inputs[0].SetValue("")

	err := m.validate()
	if err == nil {
		t.Error("expected error for empty name")
	}

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
	knownPlugins := []string{"jira", "github", "linear", "asana", "notion", "trello"}

	for _, name := range knownPlugins {
		found := false
		for _, p := range availablePlugins {
			if p.Name == name {
				found = true
				if len(p.ConfigKeys) == 0 {
					t.Errorf("plugin %q has no config keys", name)
				}
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
	path := getPluginBinaryPath("jira")
	if path == "" {
		t.Error("expected non-empty path")
	}
	if !strings.Contains(path, "roady-plugin-jira") {
		t.Errorf("expected path to contain 'roady-plugin-jira', got %q", path)
	}
}

func TestConfigureModelView_PluginSelection(t *testing.T) {
	model := newConfigureModel("", nil)
	view := model.View()

	if !strings.Contains(view, "Add Plugin Configuration") {
		t.Fatalf("expected selection view header, got:\n%s", view)
	}
	if !strings.Contains(view, "Select Plugin Type") {
		t.Fatalf("expected list title in view, got:\n%s", view)
	}
}

func TestConfigureModelView_InstallPlugin(t *testing.T) {
	model := configureModel{
		phase:      phaseInstallPlugin,
		pluginType: "github",
		spinner:    spinner.New(),
	}

	view := model.View()
	if !strings.Contains(view, "Installing Github Plugin") {
		t.Fatalf("expected install view header, got:\n%s", view)
	}

	model.installErr = errors.New("boom")
	view = model.View()
	if !strings.Contains(view, "Installation failed") {
		t.Fatalf("expected install error message, got:\n%s", view)
	}
}

func TestConfigureModelView_ConfigInputs(t *testing.T) {
	nameInput := textinput.New()
	nameInput.SetValue("config-name")

	tokenInput := textinput.New()
	tokenInput.SetValue("token-value")

	model := configureModel{
		phase:       phaseConfigurePlugin,
		pluginType:  "github",
		binaryPath:  "/tmp/roady-plugin-github",
		inputs:      []textinput.Model{nameInput, tokenInput},
		labels:      []string{"Configuration Name", "Token"},
		focusIndex:  1,
		editing:     false,
		configStart: 1,
	}

	view := model.View()
	if !strings.Contains(view, "Configure Github Plugin") {
		t.Fatalf("expected config view header, got:\n%s", view)
	}
	if !strings.Contains(view, "Token") {
		t.Fatalf("expected config field label, got:\n%s", view)
	}
}

func TestConfigureModelGetConfig_NewField(t *testing.T) {
	nameInput := textinput.New()
	nameInput.SetValue("my-config")

	fieldInput := textinput.New()
	fieldInput.SetValue("custom_key=custom_value")

	model := configureModel{
		binaryPath:  "/tmp/roady-plugin-github",
		inputs:      []textinput.Model{nameInput, fieldInput},
		labels:      []string{"Configuration Name", "New Field"},
		configStart: 1,
	}

	name, cfg := model.getConfig()
	if name != "my-config" {
		t.Fatalf("expected name my-config, got %s", name)
	}

	if cfg.Config["custom_key"] != "custom_value" {
		t.Fatalf("expected custom field in config, got: %#v", cfg.Config)
	}
}
