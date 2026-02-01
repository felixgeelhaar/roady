package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/pkg/domain/team"
	"github.com/spf13/cobra"
)

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Manage team members and roles",
}

var teamJSONOutput bool

var teamListCmd = &cobra.Command{
	Use:   "list",
	Short: "List team members and their roles",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		cfg, err := services.Team.ListMembers()
		if err != nil {
			return MapError(fmt.Errorf("list team: %w", err))
		}

		if teamJSONOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(cfg)
		}

		if len(cfg.Members) == 0 {
			fmt.Println("No team members configured.")
			return nil
		}

		fmt.Printf("Team Members (%d)\n", len(cfg.Members))
		for _, m := range cfg.Members {
			fmt.Printf("  %-20s %s\n", m.Name, m.Role)
		}
		return nil
	},
}

var teamAddCmd = &cobra.Command{
	Use:   "add <name> <role>",
	Short: "Add or update a team member (roles: admin, member, viewer)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		role := team.Role(args[1])
		if err := services.Team.AddMember(args[0], role); err != nil {
			return MapError(fmt.Errorf("add member: %w", err))
		}

		fmt.Printf("Member %s added with role %s\n", args[0], args[1])
		return nil
	},
}

var teamRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a team member",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		if err := services.Team.RemoveMember(args[0]); err != nil {
			return MapError(fmt.Errorf("remove member: %w", err))
		}

		fmt.Printf("Member %s removed\n", args[0])
		return nil
	},
}

func init() {
	teamListCmd.Flags().BoolVar(&teamJSONOutput, "json", false, "Output in JSON format")
	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamAddCmd)
	teamCmd.AddCommand(teamRemoveCmd)
	RootCmd.AddCommand(teamCmd)
}
