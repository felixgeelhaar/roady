package cli

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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

func TestListItemFilterValue(t *testing.T) {
	item := listItem{name: "jira", desc: ""}
	if item.FilterValue() != "jira" {
		t.Errorf("expected FilterValue to return name")
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

func TestConfigureModelUpdatePluginSelection_Installed(t *testing.T) {
	model := newConfigureModel("", nil)
	model.phase = phaseSelectPlugin
	items := []list.Item{
		listItem{name: "jira", desc: "", installed: true},
	}
	model.pluginList = list.New(items, list.NewDefaultDelegate(), 20, 10)
	model.pluginList.Select(0)

	updated, _ := model.updatePluginSelection(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(configureModel)
	if result.phase != phaseConfigurePlugin {
		t.Errorf("expected phaseConfigurePlugin, got %d", result.phase)
	}
	if result.binaryPath == "" {
		t.Error("expected binary path to be set")
	}
}

func TestConfigureModelUpdatePluginSelection_NeedsInstall(t *testing.T) {
	model := newConfigureModel("", nil)
	model.phase = phaseSelectPlugin
	items := []list.Item{
		listItem{name: "jira", desc: "", installed: false},
	}
	model.pluginList = list.New(items, list.NewDefaultDelegate(), 20, 10)
	model.pluginList.Select(0)

	updated, _ := model.updatePluginSelection(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(configureModel)
	if result.phase != phaseInstallPlugin {
		t.Errorf("expected phaseInstallPlugin, got %d", result.phase)
	}
	if !result.needsInstall {
		t.Error("expected needsInstall to be true")
	}
}

func TestConfigureModelUpdateInstallPlugin_Skip(t *testing.T) {
	model := configureModel{
		phase:      phaseInstallPlugin,
		pluginType: "jira",
		installErr: errors.New("boom"),
	}

	updated, _ := model.updateInstallPlugin(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	result := updated.(configureModel)
	if result.phase != phaseConfigurePlugin {
		t.Errorf("expected phaseConfigurePlugin, got %d", result.phase)
	}
	if result.binaryPath == "" {
		t.Error("expected binary path to be set")
	}
}

func TestConfigureModelUpdateConfigInputs_EnterDone(t *testing.T) {
	nameInput := textinput.New()
	nameInput.SetValue("config-name")
	valueInput := textinput.New()
	valueInput.SetValue("value")

	model := configureModel{
		phase:       phaseConfigurePlugin,
		inputs:      []textinput.Model{nameInput, valueInput},
		labels:      []string{"Configuration Name", "Value"},
		focusIndex:  1,
		configStart: 1,
	}

	updated, _ := model.updateConfigInputs(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(configureModel)
	if !result.done {
		t.Error("expected done to be true")
	}
}

func TestConfigureModelUpdateConfigInputs_AddDeleteField(t *testing.T) {
	nameInput := textinput.New()
	nameInput.SetValue("config-name")
	valueInput := textinput.New()
	valueInput.SetValue("value")

	model := configureModel{
		phase:       phaseConfigurePlugin,
		inputs:      []textinput.Model{nameInput, valueInput},
		labels:      []string{"Configuration Name", "Value"},
		focusIndex:  1,
		configStart: 1,
	}

	updated, _ := model.updateConfigInputs(tea.KeyMsg{Type: tea.KeyCtrlA})
	result := updated.(configureModel)
	if len(result.inputs) != 3 {
		t.Errorf("expected 3 inputs after add, got %d", len(result.inputs))
	}

	updated, _ = result.updateConfigInputs(tea.KeyMsg{Type: tea.KeyCtrlD})
	result = updated.(configureModel)
	if len(result.inputs) != 2 {
		t.Errorf("expected 2 inputs after delete, got %d", len(result.inputs))
	}
}

func TestConfigureModelInit(t *testing.T) {
	model := newConfigureModel("", nil)
	if cmd := model.Init(); cmd != nil {
		t.Error("expected nil cmd for select phase")
	}

	model.phase = phaseInstallPlugin
	model.pluginType = "jira"
	model.spinner = spinner.New()
	if cmd := model.Init(); cmd == nil {
		t.Error("expected cmd for install phase")
	}

	model.phase = phaseConfigurePlugin
	if cmd := model.Init(); cmd == nil {
		t.Error("expected cmd for configure phase")
	}
}

func TestConfigureModelUpdate_CtrlC(t *testing.T) {
	model := newConfigureModel("", nil)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result := updated.(configureModel)
	if !result.cancelled {
		t.Error("expected model to be cancelled")
	}
}

func TestConfigureModelUpdate_EscFromConfigure(t *testing.T) {
	model := newConfigureModel("", nil)
	model.phase = phaseConfigurePlugin
	model.editing = false
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := updated.(configureModel)
	if result.phase != phaseSelectPlugin {
		t.Errorf("expected phaseSelectPlugin, got %d", result.phase)
	}
}

func TestConfigureModelUpdate_EscFromInstall(t *testing.T) {
	model := newConfigureModel("", nil)
	model.phase = phaseInstallPlugin
	model.installErr = errors.New("boom")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := updated.(configureModel)
	if result.phase != phaseSelectPlugin {
		t.Errorf("expected phaseSelectPlugin, got %d", result.phase)
	}
}

func TestConfigureModelUpdate_EscInstallNoError(t *testing.T) {
	model := newConfigureModel("", nil)
	model.phase = phaseInstallPlugin
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := updated.(configureModel)
	if result.phase != phaseInstallPlugin {
		t.Errorf("expected phaseInstallPlugin, got %d", result.phase)
	}
}

func TestConfigureModelUpdate_InstallResult(t *testing.T) {
	model := newConfigureModel("", nil)
	model.phase = phaseInstallPlugin
	model.pluginType = "jira"
	updated, _ := model.Update(installResultMsg{})
	result := updated.(configureModel)
	if result.phase != phaseConfigurePlugin {
		t.Errorf("expected phaseConfigurePlugin, got %d", result.phase)
	}
}

func TestConfigureModelUpdateInputsMsg(t *testing.T) {
	nameInput := textinput.New()
	nameInput.SetValue("config-name")
	valueInput := textinput.New()
	valueInput.SetValue("value")

	model := configureModel{
		phase:       phaseConfigurePlugin,
		inputs:      []textinput.Model{nameInput, valueInput},
		labels:      []string{"Configuration Name", "Value"},
		focusIndex:  0,
		configStart: 1,
	}

	_ = model.updateInputsMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
}

func TestConfigureModelUpdate_WindowSize(t *testing.T) {
	model := newConfigureModel("", nil)
	model.phase = phaseSelectPlugin
	model.pluginList = list.New([]list.Item{}, list.NewDefaultDelegate(), 10, 10)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	result := updated.(configureModel)
	if result.width != 120 || result.height != 40 {
		t.Errorf("expected size 120x40, got %dx%d", result.width, result.height)
	}
}

func TestConfigureModelUpdateConfigInputs_CtrlS(t *testing.T) {
	nameInput := textinput.New()
	nameInput.SetValue("config-name")
	valueInput := textinput.New()
	valueInput.SetValue("value")

	model := configureModel{
		phase:       phaseConfigurePlugin,
		inputs:      []textinput.Model{nameInput, valueInput},
		labels:      []string{"Configuration Name", "Value"},
		focusIndex:  0,
		configStart: 1,
	}

	updated, _ := model.updateConfigInputs(tea.KeyMsg{Type: tea.KeyCtrlS})
	result := updated.(configureModel)
	if !result.done {
		t.Error("expected done to be true")
	}
}

func TestConfigureModelUpdateConfigInputs_ShiftTab(t *testing.T) {
	nameInput := textinput.New()
	nameInput.SetValue("config-name")
	valueInput := textinput.New()
	valueInput.SetValue("value")

	model := configureModel{
		phase:       phaseConfigurePlugin,
		inputs:      []textinput.Model{nameInput, valueInput},
		labels:      []string{"Configuration Name", "Value"},
		focusIndex:  0,
		configStart: 1,
	}

	updated, _ := model.updateConfigInputs(tea.KeyMsg{Type: tea.KeyShiftTab})
	result := updated.(configureModel)
	if result.focusIndex != 1 {
		t.Errorf("expected focusIndex 1, got %d", result.focusIndex)
	}
}

func TestInstallPluginCmdUnknown(t *testing.T) {
	cmd := installPluginCmd("unknown")
	msg := cmd()
	res, ok := msg.(installResultMsg)
	if !ok {
		t.Fatalf("expected installResultMsg, got %T", msg)
	}
	if res.err == nil {
		t.Error("expected error for unknown plugin")
	}
}

func TestInstallPluginSync_Invalid(t *testing.T) {
	err := installPluginSync(PluginInfo{GoPackage: "-h"})
	if err == nil {
		t.Error("expected error from go install -h")
	}
}

func TestConfigureModelUpdate_InstallResultError(t *testing.T) {
	model := newConfigureModel("", nil)
	model.phase = phaseInstallPlugin
	model.pluginType = "jira"
	updated, _ := model.Update(installResultMsg{err: errors.New("boom")})
	result := updated.(configureModel)
	if result.installErr == nil {
		t.Error("expected installErr to be set")
	}
}

func TestConfigureModelUpdate_SpinnerTick(t *testing.T) {
	model := newConfigureModel("", nil)
	model.phase = phaseInstallPlugin
	model.spinner = spinner.New()
	updated, _ := model.Update(spinner.TickMsg{})
	_ = updated.(configureModel)
}

func TestConfigureModelUpdateInstallPlugin_Retry(t *testing.T) {
	model := configureModel{
		phase:      phaseInstallPlugin,
		pluginType: "jira",
		installErr: errors.New("boom"),
	}

	_, cmd := model.updateInstallPlugin(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Error("expected retry cmd")
	}
}

func TestConfigureModelUpdateConfigInputs_Tab(t *testing.T) {
	nameInput := textinput.New()
	nameInput.SetValue("config-name")
	valueInput := textinput.New()
	valueInput.SetValue("value")

	model := configureModel{
		phase:       phaseConfigurePlugin,
		inputs:      []textinput.Model{nameInput, valueInput},
		labels:      []string{"Configuration Name", "Value"},
		focusIndex:  0,
		configStart: 1,
	}

	updated, _ := model.updateConfigInputs(tea.KeyMsg{Type: tea.KeyTab})
	result := updated.(configureModel)
	if result.focusIndex != 1 {
		t.Errorf("expected focusIndex 1, got %d", result.focusIndex)
	}
}

func TestConfigureModelUpdateConfigInputs_EnterAdvances(t *testing.T) {
	nameInput := textinput.New()
	nameInput.SetValue("config-name")
	valueInput := textinput.New()
	valueInput.SetValue("value")

	model := configureModel{
		phase:       phaseConfigurePlugin,
		inputs:      []textinput.Model{nameInput, valueInput},
		labels:      []string{"Configuration Name", "Value"},
		focusIndex:  0,
		configStart: 1,
	}

	updated, _ := model.updateConfigInputs(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(configureModel)
	if result.focusIndex != 1 {
		t.Errorf("expected focusIndex 1, got %d", result.focusIndex)
	}
}

func TestConfigureModelUpdateConfigInputs_DeleteIgnored(t *testing.T) {
	nameInput := textinput.New()
	nameInput.SetValue("config-name")
	valueInput := textinput.New()
	valueInput.SetValue("value")

	model := configureModel{
		phase:       phaseConfigurePlugin,
		inputs:      []textinput.Model{nameInput, valueInput},
		labels:      []string{"Configuration Name", "Value"},
		focusIndex:  0,
		configStart: 1,
	}

	updated, _ := model.updateConfigInputs(tea.KeyMsg{Type: tea.KeyCtrlD})
	result := updated.(configureModel)
	if len(result.inputs) != 2 {
		t.Errorf("expected inputs to remain, got %d", len(result.inputs))
	}
}

func TestIsPluginInstalled_Path(t *testing.T) {
	tempDir := t.TempDir()
	binary := tempDir + "/roady-plugin-jira"
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", tempDir+":"+oldPath)

	if !isPluginInstalled("jira") {
		t.Error("expected plugin to be installed via PATH")
	}

	path := getPluginBinaryPath("jira")
	if path != binary {
		t.Errorf("expected binary path %q, got %q", binary, path)
	}
}

func TestIsPluginInstalled_GOPATH(t *testing.T) {
	tempDir := t.TempDir()
	binDir := tempDir + "/bin"
	if err := os.MkdirAll(binDir, 0700); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	binary := binDir + "/roady-plugin-jira"
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", "")
	t.Setenv("GOPATH", tempDir)
	defer t.Setenv("PATH", oldPath)

	if !isPluginInstalled("jira") {
		t.Error("expected plugin to be installed via GOPATH")
	}

	path := getPluginBinaryPath("jira")
	if path != binary {
		t.Errorf("expected binary path %q, got %q", binary, path)
	}
}

func TestIsPluginInstalled_Local(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldDir)
	}()

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	binary := tempDir + "/roady-plugin-jira"
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	t.Setenv("PATH", "")

	if !isPluginInstalled("jira") {
		t.Error("expected plugin to be installed via local path")
	}

	path := getPluginBinaryPath("jira")
	if path != "./roady-plugin-jira" {
		t.Errorf("expected local binary path, got %q", path)
	}
}

func TestIsPluginInstalled_False(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("GOPATH", t.TempDir())
	if isPluginInstalled("nonexistent") {
		t.Error("expected plugin to be not installed")
	}
}

func TestGetPluginBinaryPath_Fallback(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", "")
	t.Setenv("GOPATH", root)

	path := getPluginBinaryPath("jira")
	expected := root + "/bin/roady-plugin-jira"
	if path != expected {
		t.Errorf("expected fallback path %q, got %q", expected, path)
	}
}
