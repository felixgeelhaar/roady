package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new roady project",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, cErr := getProjectRoot()
		if cErr != nil {
			return fmt.Errorf("resolve project path: %w", cErr)
		}
		workspace := wiring.NewWorkspace(cwd)
		repo := workspace.Repo
		audit := workspace.Audit
		service := application.NewInitService(repo, audit)

		projectName := "new-project"
		if len(args) > 0 {
			projectName = args[0]
		}

		if shouldRunInteractive() {
			return runOnboarding(service, projectName)
		}

		if initTemplate != "" {
			service.SetTemplate(initTemplate)
		}

		err := service.InitializeProject(projectName)
		if err != nil {
			return MapError(fmt.Errorf("failed to initialize project: %w", err))
		}

		fmt.Printf("Successfully initialized roady project: %s\n", projectName)
		printNextSteps()
		return nil
	},
}

var (
	initInteractive    bool
	initNonInteractive bool
	initTemplate       string
)

// stdinIsTTY is overridable in tests. Uses go-isatty so /dev/null and
// pipes correctly report as non-TTY (the file-mode ModeCharDevice check
// previously used returned true for /dev/null on Linux, which broke CI
// e2e tests by silently triggering the interactive wizard).
var stdinIsTTY = func() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

// shouldRunInteractive picks the wizard when the user clearly wants it
// (--interactive), or when running in a TTY without an explicit template or
// --non-interactive flag. CI and piped invocations stay non-interactive.
func shouldRunInteractive() bool {
	if initInteractive {
		return true
	}
	if initNonInteractive {
		return false
	}
	if initTemplate != "" {
		return false
	}
	return stdinIsTTY()
}

func printNextSteps() {
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Connect your AI tool:    roady setup claude-code")
	fmt.Println("  2. Edit your spec:           roady spec show")
	fmt.Println("  3. Generate a plan:          roady plan generate")
	fmt.Println("  4. Approve the plan:         roady plan approve")
	fmt.Println("  5. Start your first task:    roady task start <task-id>")
	fmt.Println()
}

func runOnboarding(service *application.InitService, projectName string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("Welcome to Roady!")
	fmt.Println("Let's set up your project step by step.")
	fmt.Println()

	// Step 1: Project name
	fmt.Printf("  Project name [%s]: ", projectName)
	if input := readLine(reader); input != "" {
		projectName = input
	}

	// Step 2: Template selection
	fmt.Println()
	fmt.Println("Choose a starter template:")
	templates := application.BuiltinTemplates()
	for i, t := range templates {
		fmt.Printf("  %d) %-12s  %s\n", i+1, t.Name, t.Description)
	}
	fmt.Printf("  Template [1]: ")
	choice := readLine(reader)
	templateIdx := 0
	if choice != "" {
		for i, t := range templates {
			if choice == fmt.Sprintf("%d", i+1) || choice == t.Name {
				templateIdx = i
				break
			}
		}
	}
	service.SetTemplate(templates[templateIdx].Name)

	// Step 3: Initialize
	if err := service.InitializeProject(projectName); err != nil {
		return MapError(fmt.Errorf("failed to initialize project: %w", err))
	}

	fmt.Println()
	fmt.Printf("Project '%s' initialized with template '%s'.\n", projectName, templates[templateIdx].Name)
	fmt.Println()

	// Step 4: Next steps walkthrough
	fmt.Println("Next steps:")
	fmt.Println("  1. Connect your AI tool:    roady setup claude-code")
	fmt.Println("  2. Edit your spec:           roady spec show")
	fmt.Println("  3. Generate a plan:          roady plan generate")
	fmt.Println("  4. Approve the plan:         roady plan approve")
	fmt.Println("  5. Start your first task:    roady task start <task-id>")
	fmt.Println()
	fmt.Println("Optional:")
	fmt.Println("  - Configure AI:           roady config wizard")
	fmt.Println("  - Enable shell completion: roady completion bash/zsh/fish")
	fmt.Println()

	return nil
}

func readLine(reader *bufio.Reader) string {
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func init() {
	initCmd.Flags().BoolVar(&initInteractive, "interactive", false,
		"Force the guided onboarding wizard")
	initCmd.Flags().BoolVar(&initNonInteractive, "non-interactive", false,
		"Skip the wizard even when stdin is a TTY (use in CI/scripts)")
	initCmd.Flags().StringVar(&initTemplate, "template", "",
		"Starter template (minimal, web-api, cli-tool, library)")
	RootCmd.AddCommand(initCmd)
}
