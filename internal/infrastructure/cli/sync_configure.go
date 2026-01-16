package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/felixgeelhaar/roady/pkg/domain/plugin"
	"github.com/spf13/cobra"
)

// PluginInfo describes an available plugin
type PluginInfo struct {
	Name        string
	Description string
	ConfigKeys  []string
	GoPackage   string // for go install
}

// Available plugins
var availablePlugins = []PluginInfo{
	{
		Name:        "jira",
		Description: "Atlassian Jira - Issue tracking and project management",
		ConfigKeys:  []string{"domain", "project_key", "email", "api_token"},
		GoPackage:   "github.com/felixgeelhaar/roady/cmd/roady-plugin-jira@latest",
	},
	{
		Name:        "github",
		Description: "GitHub Issues - Track issues and pull requests",
		ConfigKeys:  []string{"owner", "repo", "token"},
		GoPackage:   "github.com/felixgeelhaar/roady/cmd/roady-plugin-github@latest",
	},
	{
		Name:        "linear",
		Description: "Linear - Modern issue tracking for software teams",
		ConfigKeys:  []string{"api_key", "team_id"},
		GoPackage:   "github.com/felixgeelhaar/roady/cmd/roady-plugin-linear@latest",
	},
	{
		Name:        "asana",
		Description: "Asana - Work management and team collaboration",
		ConfigKeys:  []string{"token", "workspace_id", "project_id"},
		GoPackage:   "github.com/felixgeelhaar/roady/cmd/roady-plugin-asana@latest",
	},
	{
		Name:        "notion",
		Description: "Notion - All-in-one workspace for notes and tasks",
		ConfigKeys:  []string{"token", "database_id"},
		GoPackage:   "github.com/felixgeelhaar/roady/cmd/roady-plugin-notion@latest",
	},
	{
		Name:        "trello",
		Description: "Trello - Visual project management with boards",
		ConfigKeys:  []string{"api_key", "token", "board_id"},
		GoPackage:   "github.com/felixgeelhaar/roady/cmd/roady-plugin-trello@latest",
	},
}

// Sensitive field names that should be masked
var sensitiveFields = map[string]bool{
	"token":     true,
	"api_token": true,
	"api_key":   true,
	"password":  true,
	"secret":    true,
}

// listItem implements list.Item for plugin selection
type listItem struct {
	name, desc string
	installed  bool
}

func (i listItem) Title() string {
	status := "○"
	if i.installed {
		status = "●"
	}
	return fmt.Sprintf("%s %s", status, i.name)
}
func (i listItem) Description() string { return i.desc }
func (i listItem) FilterValue() string { return i.name }

// Phase of the configuration wizard
type configPhase int

const (
	phaseSelectPlugin configPhase = iota
	phaseInstallPlugin
	phaseConfigurePlugin
)

// Messages for async operations
type installResultMsg struct {
	err error
}

type configureModel struct {
	phase        configPhase
	pluginList   list.Model
	spinner      spinner.Model
	inputs       []textinput.Model
	labels       []string
	focusIndex   int
	pluginType   string
	pluginName   string
	binaryPath   string
	editing      bool
	done         bool
	cancelled    bool
	err          error
	installErr   error
	width        int
	height       int
	configStart  int
	needsInstall bool
}

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	noStyle      = lipgloss.NewStyle()

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginBottom(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			MarginTop(1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	listTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginLeft(2)

	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
)

func newConfigureModel(existingName string, existingConfig *plugin.PluginConfig) configureModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	m := configureModel{
		editing:    existingConfig != nil,
		pluginName: existingName,
		width:      80,
		height:     20,
		spinner:    s,
	}

	if existingConfig != nil {
		// Skip to configure phase for editing
		m.phase = phaseConfigurePlugin
		m.pluginType = detectPluginType(existingConfig.Binary)
		m.binaryPath = existingConfig.Binary
		m.initConfigInputs(existingConfig)
	} else {
		// Start with plugin selection
		m.phase = phaseSelectPlugin
		m.initPluginList()
	}

	return m
}

func (m *configureModel) initPluginList() {
	items := make([]list.Item, len(availablePlugins))
	for i, p := range availablePlugins {
		installed := isPluginInstalled(p.Name)
		items[i] = listItem{
			name:      p.Name,
			desc:      p.Description,
			installed: installed,
		}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("205")).
		BorderLeftForeground(lipgloss.Color("205"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("241"))

	l := list.New(items, delegate, m.width, m.height-4)
	l.Title = "Select Plugin Type"
	l.Styles.Title = listTitleStyle
	l.SetShowStatusBar(false)
	l.SetShowHelp(true)
	l.SetFilteringEnabled(true)

	m.pluginList = l
}

func (m *configureModel) initConfigInputs(existingConfig *plugin.PluginConfig) {
	var inputs []textinput.Model
	var labels []string

	// Plugin name input
	nameInput := textinput.New()
	nameInput.Placeholder = fmt.Sprintf("my-%s-prod", m.pluginType)
	nameInput.CharLimit = 50
	nameInput.Width = 40
	if m.pluginName != "" {
		nameInput.SetValue(m.pluginName)
	}
	inputs = append(inputs, nameInput)
	labels = append(labels, "Configuration Name")

	m.configStart = len(inputs)

	// Get config fields from plugin info
	var configFields []string
	for _, p := range availablePlugins {
		if p.Name == m.pluginType {
			configFields = p.ConfigKeys
			break
		}
	}

	// If editing, merge with existing config fields
	if existingConfig != nil && len(existingConfig.Config) > 0 {
		existingFields := make(map[string]bool)
		for _, f := range configFields {
			existingFields[f] = true
		}
		for k := range existingConfig.Config {
			if !existingFields[k] {
				configFields = append(configFields, k)
			}
		}
	}

	for _, field := range configFields {
		input := textinput.New()
		input.Placeholder = getPlaceholder(field)
		input.CharLimit = 200
		input.Width = 40

		if sensitiveFields[field] {
			input.EchoMode = textinput.EchoPassword
		}

		if existingConfig != nil {
			if val, ok := existingConfig.Config[field]; ok {
				input.SetValue(val)
			}
		}

		inputs = append(inputs, input)
		labels = append(labels, formatLabel(field))
	}

	// Focus first input
	if len(inputs) > 0 {
		inputs[0].Focus()
		inputs[0].PromptStyle = focusedStyle
		inputs[0].TextStyle = focusedStyle
	}

	m.inputs = inputs
	m.labels = labels
}

func getPlaceholder(field string) string {
	placeholders := map[string]string{
		"domain":       "https://company.atlassian.net",
		"project_key":  "PROJ",
		"email":        "user@example.com",
		"api_token":    "your-api-token",
		"token":        "your-token",
		"api_key":      "your-api-key",
		"owner":        "organization-or-username",
		"repo":         "repository-name",
		"team_id":      "TEAM-123",
		"workspace_id": "workspace-id",
		"project_id":   "project-id",
		"database_id":  "notion-database-id",
		"board_id":     "trello-board-id",
	}
	if p, ok := placeholders[field]; ok {
		return p
	}
	return fmt.Sprintf("Enter %s", field)
}

func isPluginInstalled(pluginType string) bool {
	binaryName := fmt.Sprintf("roady-plugin-%s", pluginType)

	// Check in PATH
	if _, err := exec.LookPath(binaryName); err == nil {
		return true
	}

	// Check in GOPATH/bin
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	binaryPath := filepath.Join(gopath, "bin", binaryName)
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}
	if _, err := os.Stat(binaryPath); err == nil {
		return true
	}

	// Check in current directory
	if _, err := os.Stat("./" + binaryName); err == nil {
		return true
	}

	return false
}

func getPluginBinaryPath(pluginType string) string {
	binaryName := fmt.Sprintf("roady-plugin-%s", pluginType)

	// Check in PATH first
	if path, err := exec.LookPath(binaryName); err == nil {
		return path
	}

	// Check in GOPATH/bin
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	binaryPath := filepath.Join(gopath, "bin", binaryName)
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath
	}

	// Check in current directory
	localPath := "./" + binaryName
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}

	// Return expected path even if not installed
	return binaryPath
}

func detectPluginType(binary string) string {
	lower := strings.ToLower(binary)
	for _, p := range availablePlugins {
		if strings.Contains(lower, p.Name) {
			return p.Name
		}
	}
	return "custom"
}

func formatLabel(field string) string {
	parts := strings.Split(field, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

func toTitleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// installPluginCmd runs the go install command
func installPluginCmd(pluginType string) tea.Cmd {
	return func() tea.Msg {
		var pkg string
		for _, p := range availablePlugins {
			if p.Name == pluginType {
				pkg = p.GoPackage
				break
			}
		}
		if pkg == "" {
			return installResultMsg{err: fmt.Errorf("unknown plugin: %s", pluginType)}
		}

		// #nosec G204 -- Package path is from trusted internal list
		cmd := exec.Command("go", "install", pkg)
		err := cmd.Run()
		return installResultMsg{err: err}
	}
}

func (m configureModel) Init() tea.Cmd {
	if m.phase == phaseSelectPlugin {
		return nil
	}
	if m.phase == phaseInstallPlugin {
		return tea.Batch(m.spinner.Tick, installPluginCmd(m.pluginType))
	}
	return textinput.Blink
}

func (m configureModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancelled = true
			return m, tea.Quit

		case "esc":
			if m.phase == phaseConfigurePlugin && !m.editing {
				// Go back to plugin selection
				m.phase = phaseSelectPlugin
				m.initPluginList()
				return m, nil
			}
			if m.phase == phaseInstallPlugin {
				// Can't cancel during installation, but can go back after error
				if m.installErr != nil {
					m.phase = phaseSelectPlugin
					m.installErr = nil
					m.initPluginList()
					return m, nil
				}
				return m, nil
			}
			m.cancelled = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.phase == phaseSelectPlugin {
			m.pluginList.SetSize(msg.Width, msg.Height-4)
		}

	case installResultMsg:
		if msg.err != nil {
			m.installErr = msg.err
			return m, nil
		}
		// Installation succeeded, move to configure phase
		m.binaryPath = getPluginBinaryPath(m.pluginType)
		m.phase = phaseConfigurePlugin
		m.initConfigInputs(nil)
		return m, textinput.Blink

	case spinner.TickMsg:
		if m.phase == phaseInstallPlugin && m.installErr == nil {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	switch m.phase {
	case phaseSelectPlugin:
		return m.updatePluginSelection(msg)
	case phaseInstallPlugin:
		return m.updateInstallPlugin(msg)
	default:
		return m.updateConfigInputs(msg)
	}
}

func (m configureModel) updatePluginSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := m.pluginList.SelectedItem().(listItem); ok {
				m.pluginType = item.name
				m.needsInstall = !item.installed

				if m.needsInstall {
					// Go to install phase
					m.phase = phaseInstallPlugin
					return m, tea.Batch(m.spinner.Tick, installPluginCmd(m.pluginType))
				}

				// Plugin already installed, go directly to configure
				m.binaryPath = getPluginBinaryPath(item.name)
				m.phase = phaseConfigurePlugin
				m.initConfigInputs(nil)
				return m, textinput.Blink
			}
		}
	}

	var cmd tea.Cmd
	m.pluginList, cmd = m.pluginList.Update(msg)
	return m, cmd
}

func (m configureModel) updateInstallPlugin(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Installation is handled via messages, just update spinner
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.installErr != nil {
			switch msg.String() {
			case "enter", "r":
				// Retry installation
				m.installErr = nil
				return m, tea.Batch(m.spinner.Tick, installPluginCmd(m.pluginType))
			case "s":
				// Skip installation, continue anyway
				m.binaryPath = getPluginBinaryPath(m.pluginType)
				m.phase = phaseConfigurePlugin
				m.initConfigInputs(nil)
				return m, textinput.Blink
			}
		}
	}
	return m, nil
}

func (m configureModel) updateConfigInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.focusIndex++
			if m.focusIndex >= len(m.inputs) {
				m.focusIndex = 0
			}
			return m, m.updateFocus()

		case "shift+tab", "up":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}
			return m, m.updateFocus()

		case "enter":
			if m.focusIndex == len(m.inputs)-1 {
				if err := m.validate(); err != nil {
					m.err = err
					return m, nil
				}
				m.done = true
				return m, tea.Quit
			}
			m.focusIndex++
			if m.focusIndex >= len(m.inputs) {
				m.focusIndex = 0
			}
			return m, m.updateFocus()

		case "ctrl+s":
			if err := m.validate(); err != nil {
				m.err = err
				return m, nil
			}
			m.done = true
			return m, tea.Quit

		case "ctrl+a":
			return m, m.addConfigField()

		case "ctrl+d":
			if m.focusIndex >= m.configStart && len(m.inputs) > m.configStart+1 {
				return m, m.deleteCurrentField()
			}
		}
	}

	cmd := m.updateInputsMsg(msg)
	return m, cmd
}

func (m *configureModel) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		if i == m.focusIndex {
			cmds[i] = m.inputs[i].Focus()
			m.inputs[i].PromptStyle = focusedStyle
			m.inputs[i].TextStyle = focusedStyle
		} else {
			m.inputs[i].Blur()
			m.inputs[i].PromptStyle = noStyle
			m.inputs[i].TextStyle = noStyle
		}
	}
	return tea.Batch(cmds...)
}

func (m *configureModel) updateInputsMsg(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m *configureModel) addConfigField() tea.Cmd {
	input := textinput.New()
	input.Placeholder = "field_name=value"
	input.CharLimit = 200
	input.Width = 40

	m.inputs = append(m.inputs, input)
	m.labels = append(m.labels, "New Field")
	m.focusIndex = len(m.inputs) - 1

	return m.updateFocus()
}

func (m *configureModel) deleteCurrentField() tea.Cmd {
	if m.focusIndex < m.configStart {
		return nil
	}

	m.inputs = append(m.inputs[:m.focusIndex], m.inputs[m.focusIndex+1:]...)
	m.labels = append(m.labels[:m.focusIndex], m.labels[m.focusIndex+1:]...)

	if m.focusIndex >= len(m.inputs) {
		m.focusIndex = len(m.inputs) - 1
	}

	return m.updateFocus()
}

func (m configureModel) validate() error {
	if strings.TrimSpace(m.inputs[0].Value()) == "" {
		return fmt.Errorf("configuration name is required")
	}
	return nil
}

func (m configureModel) View() string {
	switch m.phase {
	case phaseSelectPlugin:
		return m.viewPluginSelection()
	case phaseInstallPlugin:
		return m.viewInstallPlugin()
	default:
		return m.viewConfigInputs()
	}
}

func (m configureModel) viewPluginSelection() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Add Plugin Configuration"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("● = installed, ○ = not installed (will be installed automatically)"))
	b.WriteString("\n\n")
	b.WriteString(m.pluginList.View())

	return b.String()
}

func (m configureModel) viewInstallPlugin() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("Installing %s Plugin", toTitleCase(m.pluginType))))
	b.WriteString("\n\n")

	if m.installErr != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("✗ Installation failed: %v", m.installErr)))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("[r] Retry • [s] Skip (configure anyway) • [Esc] Back"))
	} else {
		b.WriteString(fmt.Sprintf("%s Installing plugin via 'go install'...\n", m.spinner.View()))
		b.WriteString("\n")
		b.WriteString(blurredStyle.Render("This may take a moment while Go downloads and builds the plugin."))
	}

	return b.String()
}

func (m configureModel) viewConfigInputs() string {
	var b strings.Builder

	title := fmt.Sprintf("Configure %s Plugin", toTitleCase(m.pluginType))
	if m.editing {
		title = fmt.Sprintf("Edit %s Configuration", m.pluginName)
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Show installation status
	if !m.editing {
		installed := isPluginInstalled(m.pluginType)
		if installed {
			b.WriteString(successStyle.Render(fmt.Sprintf("✓ Plugin installed: %s", m.binaryPath)))
		} else {
			b.WriteString(warningStyle.Render("⚠ Plugin not installed - sync will fail until installed"))
		}
		b.WriteString("\n\n")
	}

	for i, input := range m.inputs {
		label := m.labels[i]
		if i == m.focusIndex {
			b.WriteString(focusedStyle.Render(fmt.Sprintf("› %s:", label)))
		} else {
			b.WriteString(blurredStyle.Render(fmt.Sprintf("  %s:", label)))
		}
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(input.View())
		b.WriteString("\n\n")
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	help := "[Tab/↓] Next • [Shift+Tab/↑] Prev • [Enter/Ctrl+S] Save • [Ctrl+A] Add Field • [Esc] Back"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m configureModel) getConfig() (string, plugin.PluginConfig) {
	name := strings.TrimSpace(m.inputs[0].Value())

	config := make(map[string]string)
	for i := m.configStart; i < len(m.inputs); i++ {
		label := m.labels[i]
		value := strings.TrimSpace(m.inputs[i].Value())

		key := strings.ToLower(strings.ReplaceAll(label, " ", "_"))

		if label == "New Field" && strings.Contains(value, "=") {
			parts := strings.SplitN(value, "=", 2)
			key = strings.TrimSpace(parts[0])
			value = strings.TrimSpace(parts[1])
		}

		if value != "" {
			config[key] = value
		}
	}

	return name, plugin.PluginConfig{
		Binary: m.binaryPath,
		Config: config,
	}
}

// --- CLI Commands ---

var syncAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new plugin configuration interactively",
	Long: `Add a new plugin configuration using an interactive TUI.

The wizard will guide you through:
  1. Selecting the plugin type (Jira, GitHub, Linear, etc.)
  2. Automatically installing the plugin if needed
  3. Entering your configuration name
  4. Filling in the required credentials and settings

The configuration will be saved to .roady/plugins.yaml.`,
	Example: `  # Add a new plugin configuration (interactive)
  roady sync add

  # Add with a preset name
  roady sync add my-jira-prod`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		var name string
		if len(args) > 0 {
			name = args[0]
			if _, err := services.Sync.GetPluginConfig(name); err == nil {
				return fmt.Errorf("plugin '%s' already exists. Use 'roady sync edit %s' to modify it", name, name)
			}
		}

		m := newConfigureModel(name, nil)
		p := tea.NewProgram(m, tea.WithAltScreen())

		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		result := finalModel.(configureModel)
		if result.cancelled {
			fmt.Println("Cancelled.")
			return nil
		}

		if !result.done {
			return nil
		}

		configName, config := result.getConfig()
		if err := services.Sync.SetPluginConfig(configName, config); err != nil {
			return fmt.Errorf("failed to save plugin config: %w", err)
		}

		fmt.Printf("✓ Plugin '%s' configured successfully.\n", configName)
		fmt.Printf("  Type: %s\n", result.pluginType)
		fmt.Printf("  Binary: %s\n", config.Binary)
		fmt.Printf("  Config: %d fields\n", len(config.Config))
		fmt.Printf("\nRun sync with: roady sync --name %s\n", configName)

		return nil
	},
}

var syncEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit an existing plugin configuration",
	Long: `Edit an existing plugin configuration using an interactive TUI.

All existing values will be pre-filled for editing.`,
	Example: `  roady sync edit my-jira`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		name := args[0]
		existingConfig, err := services.Sync.GetPluginConfig(name)
		if err != nil {
			return fmt.Errorf("plugin '%s' not found: %w", name, err)
		}

		m := newConfigureModel(name, existingConfig)
		p := tea.NewProgram(m, tea.WithAltScreen())

		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		result := finalModel.(configureModel)
		if result.cancelled {
			fmt.Println("Cancelled.")
			return nil
		}

		if !result.done {
			return nil
		}

		newName, config := result.getConfig()

		if newName != name {
			repo := services.Workspace.Repo
			_ = repo.RemovePluginConfig(name)
			fmt.Printf("Note: Configuration renamed from '%s' to '%s'\n", name, newName)
		}

		if err := services.Sync.SetPluginConfig(newName, config); err != nil {
			return fmt.Errorf("failed to save plugin config: %w", err)
		}

		fmt.Printf("✓ Plugin '%s' updated successfully.\n", newName)

		return nil
	},
}

var syncRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a plugin configuration",
	Long:    `Remove a plugin configuration from .roady/plugins.yaml.`,
	Example: `  roady sync remove my-jira
  roady sync rm my-jira`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		name := args[0]

		if _, err := services.Sync.GetPluginConfig(name); err != nil {
			return fmt.Errorf("plugin '%s' not found", name)
		}

		repo := services.Workspace.Repo
		if err := repo.RemovePluginConfig(name); err != nil {
			return fmt.Errorf("failed to remove plugin: %w", err)
		}

		fmt.Printf("✓ Plugin '%s' removed.\n", name)
		return nil
	},
}

var syncInstallCmd = &cobra.Command{
	Use:   "install <plugin-type>",
	Short: "Install a sync plugin",
	Long: `Install a sync plugin binary using 'go install'.

Available plugins:
  - jira     Atlassian Jira
  - github   GitHub Issues
  - linear   Linear
  - asana    Asana
  - notion   Notion
  - trello   Trello

The plugin will be installed to your GOPATH/bin directory.
Note: Plugins are also installed automatically when using 'roady sync add'.`,
	Example: `  # Install the Jira plugin
  roady sync install jira

  # Install all available plugins
  roady sync install --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		installAll, _ := cmd.Flags().GetBool("all")

		if installAll {
			fmt.Println("Installing all plugins...")
			for _, p := range availablePlugins {
				if err := installPluginSync(p); err != nil {
					fmt.Printf("✗ Failed to install %s: %v\n", p.Name, err)
				} else {
					fmt.Printf("✓ Installed %s\n", p.Name)
				}
			}
			return nil
		}

		if len(args) == 0 {
			// Show available plugins
			fmt.Println("Available plugins:")
			for _, p := range availablePlugins {
				status := "○"
				if isPluginInstalled(p.Name) {
					status = "●"
				}
				fmt.Printf("  %s %-10s %s\n", status, p.Name, p.Description)
			}
			fmt.Println("\n● = installed, ○ = not installed")
			fmt.Println("\nUsage: roady sync install <plugin-type>")
			fmt.Println("       roady sync install --all")
			fmt.Println("\nNote: 'roady sync add' will also install plugins automatically.")
			return nil
		}

		pluginType := strings.ToLower(args[0])
		var pluginInfo *PluginInfo
		for _, p := range availablePlugins {
			if p.Name == pluginType {
				pluginInfo = &p
				break
			}
		}

		if pluginInfo == nil {
			return fmt.Errorf("unknown plugin type: %s", pluginType)
		}

		fmt.Printf("Installing %s plugin...\n", pluginInfo.Name)
		if err := installPluginSync(*pluginInfo); err != nil {
			return fmt.Errorf("installation failed: %w", err)
		}

		fmt.Printf("✓ Plugin '%s' installed successfully.\n", pluginInfo.Name)
		fmt.Printf("\nConfigure with: roady sync add\n")

		return nil
	},
}

func installPluginSync(p PluginInfo) error {
	// #nosec G204 -- Package path is from trusted internal list
	cmd := exec.Command("go", "install", p.GoPackage)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func init() {
	syncInstallCmd.Flags().Bool("all", false, "Install all available plugins")

	syncCmd.AddCommand(syncAddCmd)
	syncCmd.AddCommand(syncEditCmd)
	syncCmd.AddCommand(syncRemoveCmd)
	syncCmd.AddCommand(syncInstallCmd)
}
