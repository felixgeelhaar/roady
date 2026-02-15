package cli

import (
	"fmt"

	"github.com/felixgeelhaar/roady/pkg/domain/billing"
	"github.com/spf13/cobra"
)

var rateCmd = &cobra.Command{
	Use:   "rate",
	Short: "Manage billing rates",
}

var rateAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new billing rate",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}
		billingSvc := services.Billing

		rate := billing.Rate{
			ID:         rateID,
			Name:       rateName,
			HourlyRate: rateAmount,
			IsDefault:  rateDefault,
		}

		if err := billingSvc.AddRate(rate); err != nil {
			return fmt.Errorf("failed to add rate: %w", err)
		}

		fmt.Printf("Added rate: %s (%s) - $%.2f/hr\n", rateID, rateName, rateAmount)
		return nil
	},
}

var rateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all billing rates",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}
		billingSvc := services.Billing

		config, err := billingSvc.ListRates()
		if err != nil {
			return fmt.Errorf("failed to list rates: %w", err)
		}

		if len(config.Rates) == 0 {
			fmt.Println("No rates configured. Use 'roady rate add' to add a rate.")
			return nil
		}

		fmt.Printf("Currency: %s\n", config.Currency)
		if config.Tax != nil && config.Tax.Name != "" {
			fmt.Printf("Tax: %s (%.1f%%)\n", config.Tax.Name, config.Tax.Percent)
		}
		fmt.Println("\nRates:")
		for _, r := range config.Rates {
			defaultMark := ""
			if r.IsDefault {
				defaultMark = " (default)"
			}
			fmt.Printf("  %s: %s - $%.2f/hr%s\n", r.ID, r.Name, r.HourlyRate, defaultMark)
		}
		return nil
	},
}

var rateRemoveCmd = &cobra.Command{
	Use:   "rm [rate-id]",
	Short: "Remove a billing rate",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}
		billingSvc := services.Billing

		rateID := args[0]
		if err := billingSvc.RemoveRate(rateID); err != nil {
			return fmt.Errorf("failed to remove rate: %w", err)
		}

		fmt.Printf("Removed rate: %s\n", rateID)
		return nil
	},
}

var rateSetDefaultCmd = &cobra.Command{
	Use:   "default [rate-id]",
	Short: "Set the default billing rate",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}
		billingSvc := services.Billing

		rateID := args[0]
		if err := billingSvc.SetDefaultRate(rateID); err != nil {
			return fmt.Errorf("failed to set default rate: %w", err)
		}

		fmt.Printf("Set default rate to: %s\n", rateID)
		return nil
	},
}

var rateTaxCmd = &cobra.Command{
	Use:   "tax",
	Short: "Configure tax settings",
}

var rateTaxSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set tax configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := loadServicesForCurrentDir()
		if err != nil {
			return err
		}
		billingSvc := services.Billing

		if err := billingSvc.SetTax(taxName, taxPercent, taxIncluded); err != nil {
			return fmt.Errorf("failed to set tax: %w", err)
		}

		fmt.Printf("Tax configured: %s at %.1f%%\n", taxName, taxPercent)
		return nil
	},
}

var taxName string
var taxPercent float64
var taxIncluded bool

var rateID string
var rateName string
var rateAmount float64
var rateDefault bool

func init() {
	rateAddCmd.Flags().StringVarP(&rateID, "id", "", "", "Rate ID (e.g., senior, junior)")
	rateAddCmd.Flags().StringVarP(&rateName, "name", "", "", "Rate name (e.g., Senior Developer)")
	rateAddCmd.Flags().Float64VarP(&rateAmount, "rate", "", 0, "Hourly rate amount")
	rateAddCmd.Flags().BoolVarP(&rateDefault, "default", "", false, "Set as default rate")
	rateAddCmd.MarkFlagRequired("id")
	rateAddCmd.MarkFlagRequired("name")
	rateAddCmd.MarkFlagRequired("rate")

	rateCmd.AddCommand(rateAddCmd)
	rateCmd.AddCommand(rateListCmd)
	rateCmd.AddCommand(rateRemoveCmd)
	rateCmd.AddCommand(rateSetDefaultCmd)
	rateCmd.AddCommand(rateTaxCmd)
	rateTaxCmd.AddCommand(rateTaxSetCmd)

	rateTaxSetCmd.Flags().StringVar(&taxName, "name", "", "Tax name (e.g., VAT, Sales Tax)")
	rateTaxSetCmd.Flags().Float64Var(&taxPercent, "percent", 0, "Tax percentage (e.g., 20 for 20%)")
	rateTaxSetCmd.Flags().BoolVar(&taxIncluded, "included", false, "Tax is included in rate")
	rateTaxSetCmd.MarkFlagRequired("name")
	rateTaxSetCmd.MarkFlagRequired("percent")

	RootCmd.AddCommand(rateCmd)
}
