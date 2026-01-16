package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/felixgeelhaar/roady/pkg/domain/plugin"
	"github.com/spf13/cobra"
)

// Plugin templates with their required configuration fields
var pluginTemplates = map[string][]string{
	"jira":   {"domain", "project_key", "email", "api_token"},
	"github": {"owner", "repo", "token"},
	"linear": {"api_key", "team_id"},
	"custom": {}, // No predefined fields
}

// Sensitive field names that should be masked
var sensitiveFields = map[string]bool{
	"token":     true,
	"api_token": true,
	"api_key":   true,
	"password":  true,
	"secret":    true,
}

type configureModel struct {
	inputs      []textinput.Model
	labels      []string
	focusIndex  int
	pluginType  string
	pluginName  string
	editing     bool // true if editing existing config
	done        bool
	cancelled   bool
	err         error
	width       int
	configStart int // index where config fields start
}

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle  = focusedStyle
	noStyle      = lipgloss.NewStyle()

	focusedButton = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Background(lipgloss.Color("236")).
			Padding(0, 2)

	blurredButton = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Background(lipgloss.Color("235")).
			Padding(0, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginBottom(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			MarginTop(1)
)

func newConfigureModel(existingName string, existingConfig *plugin.PluginConfig) configureModel {
	m := configureModel{
		editing:    existingConfig != nil,
		pluginName: existingName,
		width:      60,
	}

	// Determine plugin type from existing config or name
	if existingConfig != nil {
		m.pluginType = detectPluginType(existingConfig.Binary)
	}

	// Create base inputs: name and binary
	var inputs []textinput.Model
	var labels []string

	// Plugin name input
	nameInput := textinput.New()
	nameInput.Placeholder = "my-jira-prod"
	nameInput.CharLimit = 50
	nameInput.Width = 40
	if existingName != "" {
		nameInput.SetValue(existingName)
	}
	inputs = append(inputs, nameInput)
	labels = append(labels, "Plugin Name")

	// Binary path input
	binaryInput := textinput.New()
	binaryInput.Placeholder = "./roady-plugin-jira"
	binaryInput.CharLimit = 200
	binaryInput.Width = 40
	if existingConfig != nil {
		binaryInput.SetValue(existingConfig.Binary)
	}
	inputs = append(inputs, binaryInput)
	labels = append(labels, "Binary Path")

	m.configStart = len(inputs)

	// Add config fields based on plugin type or existing config
	configFields := []string{}
	if existingConfig != nil && len(existingConfig.Config) > 0 {
		// Use existing config fields
		for k := range existingConfig.Config {
			configFields = append(configFields, k)
		}
	} else if m.pluginType != "" && m.pluginType != "custom" {
		// Use template fields
		configFields = pluginTemplates[m.pluginType]
	} else {
		// Default to common fields
		configFields = []string{"domain", "token"}
	}

	for _, field := range configFields {
		input := textinput.New()
		input.Placeholder = fmt.Sprintf("Enter %s", field)
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

	return m
}

func detectPluginType(binary string) string {
	lower := strings.ToLower(binary)
	if strings.Contains(lower, "jira") {
		return "jira"
	}
	if strings.Contains(lower, "github") {
		return "github"
	}
	if strings.Contains(lower, "linear") {
		return "linear"
	}
	return "custom"
}

func formatLabel(field string) string {
	// Convert snake_case to Title Case
	parts := strings.Split(field, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

func (m configureModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m configureModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit

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
			// If on last field or pressing enter, try to save
			if m.focusIndex == len(m.inputs)-1 || msg.String() == "ctrl+s" {
				// Validate
				if err := m.validate(); err != nil {
					m.err = err
					return m, nil
				}
				m.done = true
				return m, tea.Quit
			}
			// Otherwise move to next field
			m.focusIndex++
			if m.focusIndex >= len(m.inputs) {
				m.focusIndex = 0
			}
			return m, m.updateFocus()

		case "ctrl+a":
			// Add a new config field
			return m, m.addConfigField()

		case "ctrl+d":
			// Delete current config field (if it's a config field, not name/binary)
			if m.focusIndex >= m.configStart && len(m.inputs) > m.configStart+1 {
				return m, m.deleteCurrentField()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
	}

	// Update current input
	cmd := m.updateInputs(msg)
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

func (m *configureModel) updateInputs(msg tea.Msg) tea.Cmd {
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

	// Remove current field
	m.inputs = append(m.inputs[:m.focusIndex], m.inputs[m.focusIndex+1:]...)
	m.labels = append(m.labels[:m.focusIndex], m.labels[m.focusIndex+1:]...)

	if m.focusIndex >= len(m.inputs) {
		m.focusIndex = len(m.inputs) - 1
	}

	return m.updateFocus()
}

func (m configureModel) validate() error {
	if strings.TrimSpace(m.inputs[0].Value()) == "" {
		return fmt.Errorf("plugin name is required")
	}
	if strings.TrimSpace(m.inputs[1].Value()) == "" {
		return fmt.Errorf("binary path is required")
	}
	return nil
}

func (m configureModel) View() string {
	var b strings.Builder

	title := "Add Plugin Configuration"
	if m.editing {
		title = "Edit Plugin Configuration"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

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

	help := "[Tab/↓] Next • [Shift+Tab/↑] Previous • [Enter] Save • [Ctrl+A] Add Field • [Ctrl+D] Delete Field • [Esc] Cancel"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m configureModel) getConfig() (string, plugin.PluginConfig) {
	name := strings.TrimSpace(m.inputs[0].Value())
	binary := strings.TrimSpace(m.inputs[1].Value())

	config := make(map[string]string)
	for i := m.configStart; i < len(m.inputs); i++ {
		label := m.labels[i]
		value := strings.TrimSpace(m.inputs[i].Value())

		// Convert label back to snake_case for config key
		key := strings.ToLower(strings.ReplaceAll(label, " ", "_"))

		// Handle "New Field" entries with field_name=value format
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
		Binary: binary,
		Config: config,
	}
}

var syncAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new plugin configuration interactively",
	Long: `Add a new plugin configuration using an interactive TUI.

The TUI will guide you through setting up:
  - Plugin name (identifier for this configuration)
  - Binary path (path to the plugin executable)
  - Configuration values (API tokens, domains, etc.)

The configuration will be saved to .roady/plugins.yaml.`,
	Example: `  # Add a new plugin configuration
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

			// Check if already exists
			if _, err := services.Sync.GetPluginConfig(name); err == nil {
				return fmt.Errorf("plugin '%s' already exists. Use 'roady sync edit %s' to modify it", name, name)
			}
		}

		m := newConfigureModel(name, nil)
		p := tea.NewProgram(m)

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
		fmt.Printf("  Binary: %s\n", config.Binary)
		fmt.Printf("  Config: %d fields\n", len(config.Config))
		fmt.Printf("\nRun with: roady sync --name %s\n", configName)

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
		p := tea.NewProgram(m)

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

		// If name changed, remove old config
		if newName != name {
			// We need to add RemovePluginConfig to the service
			// For now, just save under new name
			fmt.Printf("Note: Plugin renamed from '%s' to '%s'\n", name, newName)
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

		// Check if exists
		if _, err := services.Sync.GetPluginConfig(name); err != nil {
			return fmt.Errorf("plugin '%s' not found", name)
		}

		// Use the repository directly for removal
		repo := services.Workspace.Repo
		if err := repo.RemovePluginConfig(name); err != nil {
			return fmt.Errorf("failed to remove plugin: %w", err)
		}

		fmt.Printf("✓ Plugin '%s' removed.\n", name)
		return nil
	},
}

func init() {
	syncCmd.AddCommand(syncAddCmd)
	syncCmd.AddCommand(syncEditCmd)
	syncCmd.AddCommand(syncRemoveCmd)
}
