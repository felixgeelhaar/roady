package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/infrastructure/dashboard"
	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Project dashboards (TUI and web-based)",
	Long: `The dashboard command provides both TUI and web-based interfaces for viewing
your Roady project status, tasks, and plan details.

Without subcommands, opens the interactive TUI dashboard.
Use 'roady dashboard serve' for the web-based dashboard.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Getenv("ROADY_SKIP_DASHBOARD_RUN") == "true" {
			return nil
		}
		p := tea.NewProgram(initialModel())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("dashboard run failed: %w", err)
		}
		return nil
	},
}

var dashboardPort int

var dashboardServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web dashboard server",
	Long: `Start a local web server to view the dashboard.

The dashboard provides:
  - Project overview with completion statistics
  - Task list with status indicators
  - Plan details and approval status

Access the dashboard in your browser at the displayed URL.`,
	Example: `  # Start on default port 3000
  roady dashboard serve

  # Start on custom port
  roady dashboard serve --port 8080`,
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		provider := &dashboardDataProvider{services: services}

		addr := fmt.Sprintf(":%d", dashboardPort)
		server, err := dashboard.NewServer(addr, provider)
		if err != nil {
			return fmt.Errorf("create server: %w", err)
		}

		// Handle graceful shutdown
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-stop
			fmt.Println("\nShutting down dashboard...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = server.Shutdown(ctx)
		}()

		fmt.Printf("Dashboard starting on http://localhost:%d\n", dashboardPort)
		fmt.Println("Press Ctrl+C to stop")

		if err := server.Start(); err != nil && err.Error() != "http: Server closed" {
			return fmt.Errorf("server error: %w", err)
		}

		return nil
	},
}

var dashboardOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "Start the web dashboard and open in browser",
	Long:  `Start the web dashboard server and automatically open it in your default browser.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		provider := &dashboardDataProvider{services: services}

		addr := fmt.Sprintf(":%d", dashboardPort)
		server, err := dashboard.NewServer(addr, provider)
		if err != nil {
			return fmt.Errorf("create server: %w", err)
		}

		// Handle graceful shutdown
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-stop
			fmt.Println("\nShutting down dashboard...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = server.Shutdown(ctx)
		}()

		// Start server in background
		go func() {
			if err := server.Start(); err != nil && err.Error() != "http: Server closed" {
				fmt.Printf("Server error: %v\n", err)
			}
		}()

		// Wait a moment for server to start
		time.Sleep(500 * time.Millisecond)

		// Open browser
		url := fmt.Sprintf("http://localhost:%d", dashboardPort)
		fmt.Printf("Opening %s in browser...\n", url)
		fmt.Println("Press Ctrl+C to stop")

		if err := openBrowser(url); err != nil {
			fmt.Printf("Could not open browser: %v\n", err)
			fmt.Printf("Please open %s manually\n", url)
		}

		// Wait for signal
		<-stop
		return nil
	},
}

func init() {
	RootCmd.AddCommand(dashboardCmd)
	dashboardCmd.AddCommand(dashboardServeCmd)
	dashboardCmd.AddCommand(dashboardOpenCmd)

	dashboardServeCmd.Flags().IntVarP(&dashboardPort, "port", "p", 3000, "Port to listen on")
	dashboardOpenCmd.Flags().IntVarP(&dashboardPort, "port", "p", 3000, "Port to listen on")
}

// dashboardDataProvider implements dashboard.DataProvider
type dashboardDataProvider struct {
	services *wiring.AppServices
}

func (p *dashboardDataProvider) GetPlan() (*planning.Plan, error) {
	return p.services.Plan.GetPlan()
}

func (p *dashboardDataProvider) GetState() (*planning.ExecutionState, error) {
	return p.services.Plan.GetState()
}

func openBrowser(url string) error {
	// Validate URL to prevent command injection
	if !isValidBrowserURL(url) {
		return fmt.Errorf("invalid URL: must be http:// or https://")
	}

	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		return fmt.Errorf("unsupported platform")
	}

	return exec.Command(cmd, args...).Start() // #nosec G204 -- URL validated above
}

// isValidBrowserURL validates that the URL is a safe http/https URL.
func isValidBrowserURL(url string) bool {
	// Only allow http and https schemes
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return false
	}
	// Reject URLs with shell metacharacters
	dangerousChars := []string{";", "|", "&", "$", "`", "(", ")", "{", "}", "<", ">", "\n", "\r"}
	for _, char := range dangerousChars {
		if strings.Contains(url, char) {
			return false
		}
	}
	return true
}

// Styles
var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var headerStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#7D56F4")).
	PaddingLeft(1).
	PaddingRight(1)

var statusDone = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
var statusWIP = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
var statusErr = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

type model struct {
	table   table.Model
	project string
	version string
	usage   int
	limit   int
	drift   []string
	err     error
}

func initialModel() model {
	services, err := loadServicesForCurrentDir()
	if err != nil {
		return model{err: err}
	}
	repo := services.Workspace.Repo

	// Load Data
	spec, err := repo.LoadSpec()
	if err != nil {
		return model{err: err}
	}

	plan, err := repo.LoadPlan()
	if err != nil {
		return model{err: err}
	}

	state, err := repo.LoadState()
	if err != nil {
		return model{err: err}
	}

	stats, _ := repo.LoadUsage()
	usageCount := 0
	if stats != nil {
		for _, c := range stats.ProviderStats {
			usageCount += c
		}
	}

	cfg, _ := repo.LoadPolicy()
	tokenLimit := 0
	if cfg != nil {
		tokenLimit = cfg.TokenLimit
	}

	// Setup Table
	columns := []table.Column{
		{Title: "Status", Width: 10},
		{Title: "Owner", Width: 15},
		{Title: "Priority", Width: 8},
		{Title: "Task", Width: 40},
		{Title: "ID", Width: 20},
	}

	rows := []table.Row{}
	if plan != nil {
		for _, t := range plan.Tasks {
			status := "pending"
			owner := "-"
			if res, ok := state.TaskStates[t.ID]; ok {
				status = string(res.Status)
				if res.Owner != "" {
					owner = res.Owner
				}
			}
			rows = append(rows, table.Row{status, owner, string(t.Priority), t.Title, t.ID})
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229"))

	t.SetStyles(s)

	// Drift Detection (Quick check)
	report, _ := services.Drift.DetectDrift(context.Background())
	driftMsgs := []string{}
	if report != nil {
		for _, issue := range report.Issues {
			driftMsgs = append(driftMsgs, fmt.Sprintf("[%s] %s", issue.Severity, issue.Message))
		}
	}

	return model{
		table:   t,
		project: spec.Title,
		version: spec.Version,
		usage:   usageCount,
		limit:   tokenLimit,
		drift:   driftMsgs,
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error loading dashboard: %v\nPress q to quit.", m.err)
	}

	header := headerStyle.Render(fmt.Sprintf("%s v%s", m.project, m.version))

	budgetText := fmt.Sprintf("AI Budget: %d", m.usage)
	if m.limit > 0 {
		budgetText = fmt.Sprintf("AI Budget: %d / %d", m.usage, m.limit)
	}

	driftView := ""
	if len(m.drift) > 0 {
		driftView = statusErr.Render("\nDRIFT DETECTED:\n")
		for _, d := range m.drift {
			driftView += fmt.Sprintf("- %s\n", d)
		}
	} else {
		driftView = statusDone.Render("\nSystem Sync: OK")
	}

	return baseStyle.Render(

		lipgloss.JoinVertical(lipgloss.Left,

			header,

			budgetText,

			"\nTask Plan:",

			m.table.View(),

			driftView,

			"\n[q] Quit  [Up/Down] Navigate",
		),
	) + "\n"

}
