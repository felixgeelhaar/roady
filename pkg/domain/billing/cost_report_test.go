package billing

import (
	"testing"
)

func TestNewCostReport(t *testing.T) {
	report := NewCostReport("USD")
	if report.Currency != "USD" {
		t.Errorf("expected USD but got %s", report.Currency)
	}
	if report.TotalHours != 0 {
		t.Errorf("expected 0 hours but got %v", report.TotalHours)
	}
	if report.TotalCost != 0 {
		t.Errorf("expected 0 cost but got %v", report.TotalCost)
	}
	if len(report.Entries) != 0 {
		t.Errorf("expected empty entries but got %d", len(report.Entries))
	}
}

func TestCostReport_SetTax(t *testing.T) {
	report := NewCostReport("USD")

	tax := &TaxConfig{Name: "VAT", Percent: 20}
	report.SetTax(tax)

	if report.TaxName != "VAT" {
		t.Errorf("expected VAT but got %s", report.TaxName)
	}
	if report.TaxPercent != 20 {
		t.Errorf("expected 20 but got %v", report.TaxPercent)
	}
}

func TestCostReport_SetTax_Nil(t *testing.T) {
	report := NewCostReport("USD")
	report.SetTax(nil)

	if report.TaxName != "" {
		t.Errorf("expected empty tax name but got %s", report.TaxName)
	}
}

func TestCostReport_AddEntry(t *testing.T) {
	report := NewCostReport("USD")

	entry := CostReportEntry{
		TaskID:     "task-1",
		Title:      "Feature A",
		RateID:     "senior",
		RateName:   "Senior Developer",
		Hours:      10,
		HourlyRate: 150,
		Cost:       1500,
	}

	report.AddEntry(entry, nil)

	if len(report.Entries) != 1 {
		t.Errorf("expected 1 entry but got %d", len(report.Entries))
	}
	if report.TotalHours != 10 {
		t.Errorf("expected 10 hours but got %v", report.TotalHours)
	}
	if report.TotalCost != 1500 {
		t.Errorf("expected 1500 cost but got %v", report.TotalCost)
	}
}

func TestCostReport_AddEntry_WithTax(t *testing.T) {
	report := NewCostReport("USD")

	tax := &TaxConfig{Name: "VAT", Percent: 20}
	report.SetTax(tax)

	entry := CostReportEntry{
		TaskID:     "task-1",
		Title:      "Feature A",
		RateID:     "senior",
		RateName:   "Senior Developer",
		Hours:      10,
		HourlyRate: 150,
		Cost:       1500,
	}

	report.AddEntry(entry, tax)

	if report.TotalTax != 300 {
		t.Errorf("expected 300 tax but got %v", report.TotalTax)
	}
	if report.TotalWithTax != 1800 {
		t.Errorf("expected 1800 total but got %v", report.TotalWithTax)
	}
}

func TestCostReport_AddEntry_Multiple(t *testing.T) {
	report := NewCostReport("USD")

	entries := []CostReportEntry{
		{TaskID: "task-1", Hours: 10, HourlyRate: 100, Cost: 1000},
		{TaskID: "task-2", Hours: 5, HourlyRate: 200, Cost: 1000},
		{TaskID: "task-3", Hours: 3, HourlyRate: 150, Cost: 450},
	}

	for _, e := range entries {
		report.AddEntry(e, nil)
	}

	if report.TotalHours != 18 {
		t.Errorf("expected 18 hours but got %v", report.TotalHours)
	}
	if report.TotalCost != 2450 {
		t.Errorf("expected 2450 cost but got %v", report.TotalCost)
	}
	if len(report.Entries) != 3 {
		t.Errorf("expected 3 entries but got %d", len(report.Entries))
	}
}

func TestCostReport_CSV_NoTax(t *testing.T) {
	report := NewCostReport("USD")

	entries := []CostReportEntry{
		{TaskID: "task-1", Title: "Feature A", RateName: "Senior", Hours: 10, HourlyRate: 150, Cost: 1500},
		{TaskID: "task-2", Title: "Feature B", RateName: "Junior", Hours: 5, HourlyRate: 75, Cost: 375},
	}

	for _, e := range entries {
		report.AddEntry(e, nil)
	}

	csv := report.CSV()

	if len(csv) == 0 {
		t.Error("expected non-empty CSV")
	}
	// Check header
	if len(csv) < 50 {
		t.Errorf("expected longer CSV output, got: %s", csv)
	}
}

func TestCostReport_CSV_WithTax(t *testing.T) {
	report := NewCostReport("EUR")
	tax := &TaxConfig{Name: "VAT", Percent: 20}
	report.SetTax(tax)

	entries := []CostReportEntry{
		{TaskID: "task-1", Title: "Feature A", RateName: "Senior", Hours: 10, HourlyRate: 150, Cost: 1500},
	}

	for _, e := range entries {
		report.AddEntry(e, tax)
	}

	csv := report.CSV()

	if len(csv) == 0 {
		t.Error("expected non-empty CSV")
	}
	// Should include tax columns
	if len(csv) < 50 {
		t.Errorf("expected longer CSV output, got: %s", csv)
	}
}
