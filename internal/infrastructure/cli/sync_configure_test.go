package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
)

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
