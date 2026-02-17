package billing

import (
	"fmt"
	"strings"
	"time"
)

type CostReportEntry struct {
	TaskID         string  `json:"task_id"`
	Title          string  `json:"title"`
	RateID         string  `json:"rate_id"`
	RateName       string  `json:"rate_name"`
	Hours          float64 `json:"hours"`
	HourlyRate     float64 `json:"hourly_rate"`
	Cost           float64 `json:"cost"`
	Currency       string  `json:"currency"`
	Description    string  `json:"description,omitempty"`
	Tax            float64 `json:"tax,omitempty"`
	TotalWithTax   float64 `json:"total_with_tax,omitempty"`
	EstimatedHours float64 `json:"estimated_hours"`
	EstimatedCost  float64 `json:"estimated_cost"`
	CostVariance   float64 `json:"cost_variance"`
	HoursVariance  float64 `json:"hours_variance"`
}

type CostReport struct {
	GeneratedAt        time.Time         `json:"generated_at"`
	Currency           string            `json:"currency"`
	Period             string            `json:"period,omitempty"`
	TaxName            string            `json:"tax_name,omitempty"`
	TaxPercent         float64           `json:"tax_percent,omitempty"`
	Entries            []CostReportEntry `json:"entries"`
	TotalHours         float64           `json:"total_hours"`
	TotalCost          float64           `json:"total_cost"`
	TotalTax           float64           `json:"total_tax"`
	TotalWithTax       float64           `json:"total_with_tax"`
	TotalEstimatedHours float64          `json:"total_estimated_hours"`
	TotalEstimatedCost  float64          `json:"total_estimated_cost"`
	TotalCostVariance   float64          `json:"total_cost_variance"`
	TotalHoursVariance  float64          `json:"total_hours_variance"`
	EstimateCoverage    float64          `json:"estimate_coverage"`
}

func NewCostReport(currency string) *CostReport {
	return &CostReport{
		GeneratedAt: time.Now(),
		Currency:    currency,
		Entries:     []CostReportEntry{},
	}
}

func (cr *CostReport) SetTax(tax *TaxConfig) {
	if tax == nil {
		return
	}
	cr.TaxName = tax.Name
	cr.TaxPercent = tax.Percent
}

func (cr *CostReport) AddEntry(entry CostReportEntry, tax *TaxConfig) {
	entry.Currency = cr.Currency
	cr.TotalHours += entry.Hours
	cr.TotalCost += entry.Cost
	cr.TotalEstimatedHours += entry.EstimatedHours
	cr.TotalEstimatedCost += entry.EstimatedCost
	cr.TotalCostVariance += entry.CostVariance
	cr.TotalHoursVariance += entry.HoursVariance

	if tax != nil && tax.Percent > 0 {
		taxAmount := entry.Cost * (tax.Percent / 100)
		entry.Tax = taxAmount
		entry.TotalWithTax = entry.Cost + taxAmount
		cr.TotalTax += taxAmount
		cr.TotalWithTax += entry.TotalWithTax
	}

	cr.Entries = append(cr.Entries, entry)
}

// ComputeCoverage calculates the percentage of entries that have estimates.
func (cr *CostReport) ComputeCoverage() {
	if len(cr.Entries) == 0 {
		cr.EstimateCoverage = 0
		return
	}
	withEstimates := 0
	for _, e := range cr.Entries {
		if e.EstimatedHours > 0 {
			withEstimates++
		}
	}
	cr.EstimateCoverage = (float64(withEstimates) / float64(len(cr.Entries))) * 100
}

func (cr *CostReport) CSV() string {
	hasTax := cr.TaxPercent > 0
	if hasTax {
		lines := []string{"Task ID,Title,Rate,Hours,Rate Amount,Cost,Tax,Total,Est.Hours,Est.Cost,Variance"}
		for _, e := range cr.Entries {
			lines = append(lines, fmt.Sprintf("%s,%s,%s,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f",
				e.TaskID, e.Title, e.RateName, e.Hours, e.HourlyRate, e.Cost, e.Tax, e.TotalWithTax,
				e.EstimatedHours, e.EstimatedCost, e.CostVariance))
		}
		lines = append(lines, fmt.Sprintf("TOTAL,,,%.2f,,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f",
			cr.TotalHours, cr.TotalCost, cr.TotalTax, cr.TotalWithTax,
			cr.TotalEstimatedHours, cr.TotalEstimatedCost, cr.TotalCostVariance))
		return strings.Join(lines, "\n")
	}
	lines := []string{"Task ID,Title,Rate,Hours,Rate Amount,Cost,Est.Hours,Est.Cost,Variance"}
	for _, e := range cr.Entries {
		lines = append(lines, fmt.Sprintf("%s,%s,%s,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f",
			e.TaskID, e.Title, e.RateName, e.Hours, e.HourlyRate, e.Cost,
			e.EstimatedHours, e.EstimatedCost, e.CostVariance))
	}
	lines = append(lines, fmt.Sprintf("TOTAL,,,%.2f,,%.2f,,,%.2f", cr.TotalHours, cr.TotalCost, cr.TotalCostVariance))
	return strings.Join(lines, "\n")
}
