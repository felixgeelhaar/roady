package cli

import (
	"testing"
)

func TestDebtCmd_RootCommand(t *testing.T) {
	if debtCmd == nil {
		t.Fatal("Expected debt command to be defined")
	}
	if debtCmd.Use != "debt" {
		t.Errorf("Use = %s, want debt", debtCmd.Use)
	}
	if debtCmd.Short == "" {
		t.Error("Expected Short description to be set")
	}
}

func TestDebtCmdSubcommands(t *testing.T) {
	subcommands := map[string]bool{
		"report":  false,
		"score":   false,
		"sticky":  false,
		"summary": false,
		"history": false,
		"trend":   false,
	}

	for _, cmd := range debtCmd.Commands() {
		cmdName := cmd.Name()
		if _, ok := subcommands[cmdName]; ok {
			subcommands[cmdName] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("Missing subcommand: %s", name)
		}
	}
}

func TestDebtReportCmd_Structure(t *testing.T) {
	if debtReportCmd.Use != "report" {
		t.Errorf("Use = %s, want report", debtReportCmd.Use)
	}
	if debtReportCmd.Short == "" {
		t.Error("Expected Short description to be set")
	}
	if debtReportCmd.RunE == nil {
		t.Error("Expected RunE to be set")
	}

	// Check flags
	outputFlag := debtReportCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Error("Expected --output flag")
	}
}

func TestDebtScoreCmd_Structure(t *testing.T) {
	if debtScoreCmd.Use != "score <component-id>" {
		t.Errorf("Use = %s, want 'score <component-id>'", debtScoreCmd.Use)
	}
	if debtScoreCmd.Short == "" {
		t.Error("Expected Short description to be set")
	}
	if debtScoreCmd.RunE == nil {
		t.Error("Expected RunE to be set")
	}

	// Check flags
	outputFlag := debtScoreCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Error("Expected --output flag")
	}
}

func TestDebtStickyCmd_Structure(t *testing.T) {
	if debtStickyCmd.Use != "sticky" {
		t.Errorf("Use = %s, want sticky", debtStickyCmd.Use)
	}
	if debtStickyCmd.Short == "" {
		t.Error("Expected Short description to be set")
	}
	if debtStickyCmd.RunE == nil {
		t.Error("Expected RunE to be set")
	}

	// Check flags
	outputFlag := debtStickyCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Error("Expected --output flag")
	}
}

func TestDebtSummaryCmd_Structure(t *testing.T) {
	if debtSummaryCmd.Use != "summary" {
		t.Errorf("Use = %s, want summary", debtSummaryCmd.Use)
	}
	if debtSummaryCmd.Short == "" {
		t.Error("Expected Short description to be set")
	}
	if debtSummaryCmd.RunE == nil {
		t.Error("Expected RunE to be set")
	}

	// Check flags
	outputFlag := debtSummaryCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Error("Expected --output flag")
	}
}

func TestDebtHistoryCmd_Structure(t *testing.T) {
	if debtHistoryCmd.Use != "history" {
		t.Errorf("Use = %s, want history", debtHistoryCmd.Use)
	}
	if debtHistoryCmd.Short == "" {
		t.Error("Expected Short description to be set")
	}
	if debtHistoryCmd.RunE == nil {
		t.Error("Expected RunE to be set")
	}

	// Check flags
	outputFlag := debtHistoryCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Error("Expected --output flag")
	}
	daysFlag := debtHistoryCmd.Flags().Lookup("days")
	if daysFlag == nil {
		t.Error("Expected --days flag")
	}
}

func TestDebtTrendCmd_Structure(t *testing.T) {
	if debtTrendCmd.Use != "trend" {
		t.Errorf("Use = %s, want trend", debtTrendCmd.Use)
	}
	if debtTrendCmd.Short == "" {
		t.Error("Expected Short description to be set")
	}
	if debtTrendCmd.RunE == nil {
		t.Error("Expected RunE to be set")
	}

	// Check flags
	outputFlag := debtTrendCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Error("Expected --output flag")
	}
	daysFlag := debtTrendCmd.Flags().Lookup("days")
	if daysFlag == nil {
		t.Error("Expected --days flag")
	}
}

func TestDebtCmd_RegisteredToRoot(t *testing.T) {
	found := false
	for _, cmd := range RootCmd.Commands() {
		if cmd.Name() == "debt" {
			found = true
			break
		}
	}
	if !found {
		t.Error("debt command not registered to root command")
	}
}
