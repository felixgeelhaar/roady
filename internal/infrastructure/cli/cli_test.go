package cli

import (
	"bufio"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/billing"
)

func TestIntOrDefault(t *testing.T) {
	tests := []struct {
		val      int
		fallback string
		want     string
	}{
		{0, "default", "default"},
		{0, "", ""},
		{5, "default", "5"},
		{100, "", "100"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := intOrDefault(tt.val, tt.fallback)
			if got != tt.want {
				t.Errorf("intOrDefault(%d, %q) = %q, want %q", tt.val, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestBoolStr(t *testing.T) {
	tests := []struct {
		val  bool
		want string
	}{
		{true, "true"},
		{false, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := boolStr(tt.val)
			if got != tt.want {
				t.Errorf("boolStr(%v) = %q, want %q", tt.val, got, tt.want)
			}
		})
	}
}

func TestGenerateTextTable(t *testing.T) {
	entries := []billing.CostReportEntry{
		{TaskID: "task-1", Title: "Short", RateName: "Standard", Hours: 2.5, Cost: 250.00},
		{TaskID: "task-2", Title: "Very Long Task Title Here", RateName: "Premium", Hours: 1.0, Cost: 150.00},
	}

	got := generateTextTable(entries)

	if len(got) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestGenerateMarkdownTable(t *testing.T) {
	entries := []billing.CostReportEntry{
		{TaskID: "task-1", Title: "Task One", RateName: "Standard", Hours: 2.5, Cost: 250.00},
	}

	got := generateMarkdownTable(entries)

	if len(got) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestGenerateTextReport(t *testing.T) {
	report := &billing.CostReport{
		Currency:   "USD",
		TotalHours: 10.0,
		TotalCost:  1000.00,
		Entries: []billing.CostReportEntry{
			{TaskID: "task-1", Title: "Task One", RateName: "Standard", Hours: 10.0, Cost: 1000.00},
		},
	}

	got := generateTextReport(report)

	if len(got) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestGenerateMarkdownReport(t *testing.T) {
	report := &billing.CostReport{
		Currency:   "EUR",
		TotalHours: 5.0,
		TotalCost:  500.00,
		Entries: []billing.CostReportEntry{
			{TaskID: "task-1", Title: "Task One", RateName: "Standard", Hours: 5.0, Cost: 500.00},
		},
	}

	got := generateMarkdownReport(report)

	if len(got) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestPrintTextReport(t *testing.T) {
	report := &billing.CostReport{
		Currency:   "USD",
		TotalHours: 8.0,
		TotalCost:  800.00,
		Entries: []billing.CostReportEntry{
			{TaskID: "task-1", Title: "Task One", RateName: "Standard", Hours: 8.0, Cost: 800.00},
		},
	}

	printTextReport(report)
}

func TestPrintTextReportWithTax(t *testing.T) {
	report := &billing.CostReport{
		Currency:     "USD",
		TaxName:      "VAT",
		TaxPercent:   20.0,
		TotalTax:     160.00,
		TotalHours:   8.0,
		TotalCost:    800.00,
		TotalWithTax: 960.00,
		Entries: []billing.CostReportEntry{
			{TaskID: "task-1", Title: "Task One", RateName: "Standard", Hours: 8.0, Cost: 800.00, Tax: 160.00, TotalWithTax: 960.00},
		},
	}

	printTextReport(report)
}

func TestPrintMarkdownReport(t *testing.T) {
	report := &billing.CostReport{
		Currency:   "USD",
		TotalHours: 8.0,
		TotalCost:  800.00,
		Entries: []billing.CostReportEntry{
			{TaskID: "task-1", Title: "Task One", RateName: "Standard", Hours: 8.0, Cost: 800.00},
		},
	}

	printMarkdownReport(report)
}

func TestPrintMarkdownReportWithTax(t *testing.T) {
	report := &billing.CostReport{
		Currency:     "USD",
		TaxName:      "VAT",
		TaxPercent:   20.0,
		TotalTax:     160.00,
		TotalHours:   8.0,
		TotalCost:    800.00,
		TotalWithTax: 960.00,
		Entries: []billing.CostReportEntry{
			{TaskID: "task-1", Title: "Task One", RateName: "Standard", Hours: 8.0, Cost: 800.00, Tax: 160.00, TotalWithTax: 960.00},
		},
	}

	printMarkdownReport(report)
}

func TestPrompt_WithInput(t *testing.T) {
	input := strings.NewReader("user input\n")
	reader := bufio.NewReader(input)

	got := prompt(reader, "Label", "default")
	if got != "user input" {
		t.Errorf("prompt() = %q, want %q", got, "user input")
	}
}

func TestPrompt_WithDefault(t *testing.T) {
	input := strings.NewReader("\n")
	reader := bufio.NewReader(input)

	got := prompt(reader, "Label", "default")
	if got != "default" {
		t.Errorf("prompt() = %q, want %q", got, "default")
	}
}
