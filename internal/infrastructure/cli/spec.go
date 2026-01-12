package cli

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/internal/application"
	"github.com/felixgeelhaar/roady/internal/infrastructure/ai"
	"github.com/felixgeelhaar/roady/internal/infrastructure/storage"
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
		cwd, _ := os.Getwd()
		repo := storage.NewFilesystemRepository(cwd)
		audit := application.NewAuditService(repo)

		cfg, _ := repo.LoadPolicy()
		pName, mName := "ollama", "llama3"
		if cfg != nil {
			pName = cfg.AIProvider
			mName = cfg.AIModel
		}

		baseProvider, err := ai.GetDefaultProvider(pName, mName)
		if err != nil {
			return err
		}
		provider := ai.NewResilientProvider(baseProvider)
		service := application.NewAIPlanningService(repo, provider, audit)

		explanation, err := service.ExplainSpec(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to explain spec: %w", err)
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
		cwd, _ := os.Getwd()
		repo := storage.NewFilesystemRepository(cwd)
		service := application.NewSpecService(repo)
		filePath := args[0]

		spec, err := service.ImportFromMarkdown(filePath)
		if err != nil {
			return fmt.Errorf("failed to import spec: %w", err)
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
		cwd, _ := os.Getwd()
		repo := storage.NewFilesystemRepository(cwd)
		spec, err := repo.LoadSpec()
		if err != nil {
			return fmt.Errorf("failed to load/parse spec: %w", err)
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

		cwd, _ := os.Getwd()
		repo := storage.NewFilesystemRepository(cwd)
		service := application.NewSpecService(repo)

		spec, err := service.AnalyzeDirectory(dir)
		if err != nil {
			return fmt.Errorf("failed to analyze directory: %w", err)
		}

		if reconcileSpec {
			audit := application.NewAuditService(repo)
			cfg, _ := repo.LoadPolicy()
			pName, mName := "ollama", "llama3"
			if cfg != nil {
				pName = cfg.AIProvider
				mName = cfg.AIModel
			}
			baseProvider, err := ai.GetDefaultProvider(pName, mName)
			if err != nil {
				return err
			}
			provider := ai.NewResilientProvider(baseProvider)
			aiSvc := application.NewAIPlanningService(repo, provider, audit)

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
		cwd, _ := os.Getwd()
		repo := storage.NewFilesystemRepository(cwd)
		service := application.NewSpecService(repo)

		title, desc := args[0], args[1]
		spec, err := service.AddFeature(title, desc)
		if err != nil {
			return fmt.Errorf("failed to add feature: %w", err)
		}

		fmt.Printf("Successfully added feature '%s'. (Total features: %d)\n", title, len(spec.Features))
		fmt.Println("Intent synced to docs/backlog.md")
		return nil
	},
}

func init() {
	specCmd.AddCommand(specAddCmd)
	specAnalyzeCmd.Flags().BoolVar(&reconcileSpec, "reconcile", false, "Use AI to semanticly deduplicate and reconcile the spec")
	specCmd.AddCommand(specImportCmd)
	specCmd.AddCommand(specValidateCmd)
	specCmd.AddCommand(specExplainCmd)
	specCmd.AddCommand(specAnalyzeCmd)
	RootCmd.AddCommand(specCmd)
}