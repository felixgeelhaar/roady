package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/roady/pkg/domain/dependency"
	"github.com/spf13/cobra"
)

var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Manage cross-repository dependencies",
}

var depsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered dependencies",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFormat, _ := cmd.Flags().GetString("output")

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		deps, err := services.Dependency.ListDependencies()
		if err != nil {
			return fmt.Errorf("failed to list dependencies: %w", err)
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(deps, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(deps) == 0 {
			fmt.Println("No dependencies registered.")
			fmt.Println("Use 'roady deps add --repo <path> --type <type>' to add dependencies.")
			return nil
		}

		fmt.Printf("Dependencies (%d):\n\n", len(deps))
		for _, dep := range deps {
			fmt.Printf("  %s\n", dep.TargetRepo)
			fmt.Printf("    ID:   %s\n", dep.ID)
			fmt.Printf("    Type: %s\n", dep.Type)
			if dep.Description != "" {
				fmt.Printf("    Desc: %s\n", dep.Description)
			}
			if len(dep.FeatureIDs) > 0 {
				fmt.Printf("    Features: %s\n", strings.Join(dep.FeatureIDs, ", "))
			}
			fmt.Println()
		}

		return nil
	},
}

var depsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new dependency to the project",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, _ := cmd.Flags().GetString("repo")
		depType, _ := cmd.Flags().GetString("type")
		description, _ := cmd.Flags().GetString("description")

		if repoPath == "" {
			return fmt.Errorf("--repo is required")
		}
		if depType == "" {
			return fmt.Errorf("--type is required (runtime, data, build, intent)")
		}

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		dep, err := services.Dependency.AddDependency(repoPath, dependency.DependencyType(depType), description)
		if err != nil {
			return fmt.Errorf("failed to add dependency: %w", err)
		}

		fmt.Printf("Added dependency: %s\n", dep.TargetRepo)
		fmt.Printf("  ID:   %s\n", dep.ID)
		fmt.Printf("  Type: %s\n", dep.Type)
		if dep.Description != "" {
			fmt.Printf("  Desc: %s\n", dep.Description)
		}

		return nil
	},
}

var depsRemoveCmd = &cobra.Command{
	Use:   "remove <dependency-id>",
	Short: "Remove a dependency by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		depID := args[0]

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		if err := services.Dependency.RemoveDependency(depID); err != nil {
			return fmt.Errorf("failed to remove dependency: %w", err)
		}

		fmt.Printf("Removed dependency: %s\n", depID)
		return nil
	},
}

var depsScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan health status of all dependent repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFormat, _ := cmd.Flags().GetString("output")

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		// Use nil scanner for basic health check (checks reachability)
		result, err := services.Dependency.ScanDependentRepos(nil)
		if err != nil {
			return fmt.Errorf("failed to scan dependencies: %w", err)
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Dependency Health Scan\n")
		fmt.Printf("======================\n\n")
		fmt.Printf("Total repositories: %d\n", result.TotalRepos)
		fmt.Printf("Healthy:           %d\n", result.HealthyRepos)
		fmt.Printf("Unhealthy:         %d\n", result.UnhealthyRepos)
		fmt.Printf("Unreachable:       %d\n", result.Unreachable)
		fmt.Println()

		if len(result.Details) > 0 {
			fmt.Println("Details:")
			for repoPath, health := range result.Details {
				status := "healthy"
				if !health.IsReachable {
					status = "unreachable"
				} else if !health.IsHealthy() {
					status = "unhealthy"
				}
				fmt.Printf("  %s: %s\n", repoPath, status)
				if health.ErrorMessage != "" {
					fmt.Printf("    Error: %s\n", health.ErrorMessage)
				}
				if health.HasDrift {
					fmt.Printf("    Warning: has drift\n")
				}
			}
		}

		if !result.AllHealthy() {
			return fmt.Errorf("some dependencies are unhealthy")
		}

		return nil
	},
}

var depsGraphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Show dependency graph summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFormat, _ := cmd.Flags().GetString("output")
		checkCycles, _ := cmd.Flags().GetBool("check-cycles")

		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		summary, err := services.Dependency.GetDependencySummary()
		if err != nil {
			return fmt.Errorf("failed to get dependency summary: %w", err)
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(summary, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Dependency Graph Summary\n")
		fmt.Printf("========================\n\n")
		fmt.Printf("Total dependencies: %d\n", summary.TotalDependencies)
		fmt.Println()

		if len(summary.ByType) > 0 {
			fmt.Println("By type:")
			for depType, count := range summary.ByType {
				fmt.Printf("  %s: %d\n", depType, count)
			}
			fmt.Println()
		}

		if checkCycles {
			hasCycle, err := services.Dependency.CheckForCycles()
			if err != nil {
				return fmt.Errorf("failed to check for cycles: %w", err)
			}
			if hasCycle {
				fmt.Println("Warning: Cyclic dependency detected!")
				return fmt.Errorf("cyclic dependency detected")
			}
			fmt.Println("No cyclic dependencies detected.")
		}

		// Show dependency order if requested
		showOrder, _ := cmd.Flags().GetBool("order")
		if showOrder {
			order, err := services.Dependency.GetDependencyOrder()
			if err != nil {
				fmt.Printf("\nCould not determine dependency order: %v\n", err)
			} else {
				fmt.Println("\nDependency order (dependencies first):")
				for i, repo := range order {
					fmt.Printf("  %d. %s\n", i+1, repo)
				}
			}
		}

		return nil
	},
}

func init() {
	// List command flags
	depsListCmd.Flags().StringP("output", "o", "text", "Output format (text, json)")

	// Add command flags
	depsAddCmd.Flags().StringP("repo", "r", "", "Path to the dependent repository")
	depsAddCmd.Flags().StringP("type", "t", "", "Dependency type (runtime, data, build, intent)")
	depsAddCmd.Flags().StringP("description", "d", "", "Description of the dependency")

	// Scan command flags
	depsScanCmd.Flags().StringP("output", "o", "text", "Output format (text, json)")

	// Graph command flags
	depsGraphCmd.Flags().StringP("output", "o", "text", "Output format (text, json)")
	depsGraphCmd.Flags().Bool("check-cycles", false, "Check for cyclic dependencies")
	depsGraphCmd.Flags().Bool("order", false, "Show topological dependency order")

	// Add subcommands
	depsCmd.AddCommand(depsListCmd)
	depsCmd.AddCommand(depsAddCmd)
	depsCmd.AddCommand(depsRemoveCmd)
	depsCmd.AddCommand(depsScanCmd)
	depsCmd.AddCommand(depsGraphCmd)

	RootCmd.AddCommand(depsCmd)
}
