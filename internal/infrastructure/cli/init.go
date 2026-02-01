package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new roady project",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		workspace := wiring.NewWorkspace(cwd)
		repo := workspace.Repo
		audit := workspace.Audit
		service := application.NewInitService(repo, audit)

		projectName := "new-project"
		if len(args) > 0 {
			projectName = args[0]
		}

		if initInteractive {
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
		return nil
	},
}

var (
	initInteractive bool
	initTemplate    string
)

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
	fmt.Println("  1. Edit your spec:       roady spec show")
	fmt.Println("  2. Generate a plan:       roady plan generate")
	fmt.Println("  3. Approve the plan:      roady plan approve")
	fmt.Println("  4. Start your first task: roady task start <task-id>")
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
	initCmd.Flags().BoolVar(&initInteractive, "interactive", false, "Guided onboarding with template selection")
	initCmd.Flags().StringVar(&initTemplate, "template", "", "Starter template (minimal, web-api, cli-tool, library)")
	RootCmd.AddCommand(initCmd)
}
