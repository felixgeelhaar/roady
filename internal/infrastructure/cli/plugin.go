package cli

import (
	"encoding/json"
	"fmt"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage syncer plugins",
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered plugins",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		svc := application.NewPluginService(services.Workspace.Repo)
		plugins, err := svc.ListPlugins()
		if err != nil {
			return err
		}

		if len(plugins) == 0 {
			fmt.Println("No plugins registered.")
			return nil
		}

		data, err := json.MarshalIndent(plugins, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	},
}

var pluginRegisterCmd = &cobra.Command{
	Use:   "register <name> <binary-path>",
	Short: "Register a syncer plugin",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		svc := application.NewPluginService(services.Workspace.Repo)
		if err := svc.RegisterPlugin(args[0], args[1]); err != nil {
			return err
		}

		fmt.Printf("Plugin %q registered: %s\n", args[0], args[1])
		return nil
	},
}

var pluginUnregisterCmd = &cobra.Command{
	Use:   "unregister <name>",
	Short: "Unregister a syncer plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		svc := application.NewPluginService(services.Workspace.Repo)
		if err := svc.UnregisterPlugin(args[0]); err != nil {
			return err
		}

		fmt.Printf("Plugin %q unregistered.\n", args[0])
		return nil
	},
}

var pluginValidateCmd = &cobra.Command{
	Use:   "validate <name>",
	Short: "Validate a registered plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}

		svc := application.NewPluginService(services.Workspace.Repo)
		result, err := svc.ValidatePlugin(args[0])
		if err != nil {
			return err
		}

		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	},
}

func init() {
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginRegisterCmd)
	pluginCmd.AddCommand(pluginUnregisterCmd)
	pluginCmd.AddCommand(pluginValidateCmd)
	RootCmd.AddCommand(pluginCmd)
}
