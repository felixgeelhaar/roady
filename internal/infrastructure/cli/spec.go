package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/spf13/cobra"
)

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Manage product specifications",
}

var specExplainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Provide an AI-generated explanation of the current spec",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := getProjectRoot()
		if err != nil {
			return fmt.Errorf("resolve project path: %w", err)
		}
		workspace := wiring.NewWorkspace(cwd)
		repo := workspace.Repo
		audit := workspace.Audit

		provider, err := wiring.LoadAIProvider(cwd)
		if err != nil {
			return err
		}
		planSvc := application.NewPlanService(repo, audit)
		service := application.NewAIPlanningService(repo, provider, audit, planSvc)

		explanation, err := service.ExplainSpec(cmd.Context())
		if err != nil {
			return MapError(fmt.Errorf("failed to explain spec: %w", err))
		}

		fmt.Println("\n--- Spec Explanation ---")
		fmt.Println(explanation)
		fmt.Println("-------------------------")
		return nil
	},
}

var specImportCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import a spec from a markdown file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := getProjectRoot()
		if err != nil {
			return fmt.Errorf("resolve project path: %w", err)
		}
		repo := wiring.NewWorkspace(cwd).Repo
		service := application.NewSpecService(repo)
		filePath := args[0]

		spec, err := service.ImportFromMarkdown(filePath)
		if err != nil {
			return MapError(fmt.Errorf("failed to import spec: %w", err))
		}

		fmt.Printf("Successfully imported spec '%s' with %d features.\n", spec.Title, len(spec.Features))
		return nil
	},
}

var specValidateCmd = &cobra.Command{
	Use:     "validate",
	Aliases: []string{"lint"},
	Short:   "Validate the current specification (alias: lint)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := getProjectRoot()
		if err != nil {
			return fmt.Errorf("resolve project path: %w", err)
		}
		repo := wiring.NewWorkspace(cwd).Repo
		spec, err := repo.LoadSpec()
		if err != nil {
			return MapError(fmt.Errorf("failed to load/parse spec: %w", err))
		}

		errs := spec.Validate()
		if len(errs) > 0 {
			fmt.Println("Spec validation failed:")
			for _, e := range errs {
				fmt.Printf("- %v\n", e)
			}
			return fmt.Errorf("spec validation failed")
		}

		fmt.Println("Spec is valid and correctly formatted.")
		return nil
	},
}

var reconcileSpec bool

var specAnalyzeCmd = &cobra.Command{
	Use:   "analyze [dir]",
	Short: "Analyze a directory for markdown files and infer a product specification",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}

		cwd, err := getProjectRoot()
		if err != nil {
			return fmt.Errorf("resolve project path: %w", err)
		}
		workspace := wiring.NewWorkspace(cwd)
		repo := workspace.Repo
		service := application.NewSpecService(repo)

		spec, err := service.AnalyzeDirectory(dir)
		if err != nil {
			return MapError(fmt.Errorf("failed to analyze directory: %w", err))
		}

		if reconcileSpec {
			audit := workspace.Audit
			provider, err := wiring.LoadAIProvider(cwd)
			if err != nil {
				return err
			}
			planSvc := application.NewPlanService(repo, audit)
			aiSvc := application.NewAIPlanningService(repo, provider, audit, planSvc)

			fmt.Println("Reconciling specification using AI...")
			spec, err = aiSvc.ReconcileSpec(cmd.Context(), spec)
			if err != nil {
				return fmt.Errorf("failed to reconcile spec: %w", err)
			}
		}

		fmt.Printf("Successfully analyzed directory and generated spec '%s' with %d features.\n", spec.Title, len(spec.Features))
		return nil
	},
}

var specAddCmd = &cobra.Command{
	Use:   "add [title] [description]",
	Short: "Quickly add a new feature to the specification",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := getProjectRoot()
		if err != nil {
			return fmt.Errorf("resolve project path: %w", err)
		}
		repo := wiring.NewWorkspace(cwd).Repo
		service := application.NewSpecService(repo)

		title, desc := args[0], args[1]
		spec, err := service.AddFeature(title, desc)
		if err != nil {
			return MapError(fmt.Errorf("failed to add feature: %w", err))
		}

		fmt.Printf("Successfully added feature '%s'. (Total features: %d)\n", title, len(spec.Features))
		fmt.Println("Intent synced to docs/backlog.md")
		return nil
	},
}

var specReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Perform an AI-powered quality review of the current spec",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := getProjectRoot()
		if err != nil {
			return fmt.Errorf("resolve project path: %w", err)
		}
		workspace := wiring.NewWorkspace(cwd)
		repo := workspace.Repo
		audit := workspace.Audit

		provider, err := wiring.LoadAIProvider(cwd)
		if err != nil {
			return err
		}
		planSvc := application.NewPlanService(repo, audit)
		service := application.NewAIPlanningService(repo, provider, audit, planSvc)

		review, err := service.ReviewSpec(cmd.Context())
		if err != nil {
			return MapError(fmt.Errorf("failed to review spec: %w", err))
		}

		fmt.Printf("\n--- Spec Quality Review (Score: %d/100) ---\n", review.Score)
		fmt.Println(review.Summary)
		if len(review.Findings) > 0 {
			fmt.Println("\nFindings:")
			for _, f := range review.Findings {
				featureTag := ""
				if f.FeatureID != "" {
					featureTag = fmt.Sprintf(" [%s]", f.FeatureID)
				}
				fmt.Printf("  [%s] %s%s: %s\n", f.Severity, f.Category, featureTag, f.Title)
				fmt.Printf("         → %s\n", f.Suggestion)
			}
		}
		fmt.Println("-------------------------------------------")
		return nil
	},
}

var specParseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Parse raw LLM output into a structured spec and plan in one operation",
	Long: `Parse unstructured LLM-generated content into a structured ProductSpec 
and execution Plan in a single AI call. 

This command accepts:
  - Text from stdin: cat file.txt | roady spec parse
  - A file path: roady spec parse path/to/llm-output.txt
  - Direct input: roady spec parse "Feature: Login..."`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := getProjectRoot()
		if err != nil {
			return fmt.Errorf("resolve project path: %w", err)
		}

		var rawText string
		switch {
		case len(args) > 0:
			rawText = args[0]
		case hasStdin():
			rawText = readStdin()
		default:
			return fmt.Errorf("no input provided: pipe text via stdin, provide a file path, or pass text directly as an argument")
		}

		if rawText == "" {
			return fmt.Errorf("empty input")
		}

		workspace := wiring.NewWorkspace(cwd)
		repo := workspace.Repo
		audit := workspace.Audit

		provider, err := wiring.LoadAIProvider(cwd)
		if err != nil {
			return err
		}
		planSvc := application.NewPlanService(repo, audit)
		aiSvc := application.NewAIPlanningService(repo, provider, audit, planSvc)

		fmt.Println("Parsing LLM output...")
		spec, plan, err := aiSvc.ImportFromLLM(cmd.Context(), rawText)
		if err != nil {
			return MapError(fmt.Errorf("failed to parse LLM output: %w", err))
		}

		fmt.Printf("\nSuccessfully created spec '%s' with %d features.\n", spec.Title, len(spec.Features))
		if plan != nil {
			fmt.Printf("Created plan with %d tasks.\n", len(plan.Tasks))
		}
		return nil
	},
}

func hasStdin() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func readStdin() string {
	reader := bufio.NewReader(os.Stdin)
	var lines []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}
		lines = append(lines, line)
		if err == io.EOF {
			break
		}
	}
	return ""
}

func init() {
	specCmd.AddCommand(specAddCmd)
	specAnalyzeCmd.Flags().BoolVar(&reconcileSpec, "reconcile", false, "Use AI to semanticly deduplicate and reconcile the spec")
	specCmd.AddCommand(specImportCmd)
	specCmd.AddCommand(specValidateCmd)
	specCmd.AddCommand(specExplainCmd)
	specCmd.AddCommand(specReviewCmd)
	specCmd.AddCommand(specAnalyzeCmd)
	specCmd.AddCommand(specParseCmd)
	RootCmd.AddCommand(specCmd)
}
