package billing

import "fmt"

// NewRate creates a validated Rate value object.
func NewRate(id, name string, hourlyRate float64, isDefault bool) (Rate, error) {
	if id == "" {
		return Rate{}, fmt.Errorf("rate ID must not be empty")
	}
	if name == "" {
		return Rate{}, fmt.Errorf("rate name must not be empty")
	}
	if hourlyRate < 0 {
		return Rate{}, fmt.Errorf("hourly rate must be >= 0")
	}
	return Rate{
		ID:         id,
		Name:       name,
		HourlyRate: hourlyRate,
		IsDefault:  isDefault,
	}, nil
}

type Rate struct {
	ID         string  `yaml:"id" json:"id"`
	Name       string  `yaml:"name" json:"name"`
	HourlyRate float64 `yaml:"hourly_rate" json:"hourly_rate"`
	IsDefault  bool    `yaml:"default" json:"default"`
}

type TaxConfig struct {
	Name     string  `yaml:"name" json:"name"`         // e.g., "VAT", "Sales Tax"
	Percent  float64 `yaml:"percent" json:"percent"`   // e.g., 20.0 for 20%
	Included bool    `yaml:"included" json:"included"` // true if tax is included in rate
}

type RateConfig struct {
	Currency string     `yaml:"currency" json:"currency"`
	Tax      *TaxConfig `yaml:"tax,omitempty" json:"tax,omitempty"`
	Rates    []Rate     `yaml:"rates" json:"rates"`
}

// AddRate adds a rate to the config, enforcing uniqueness and single-default.
func (rc *RateConfig) AddRate(rate Rate) error {
	for _, r := range rc.Rates {
		if r.ID == rate.ID {
			return fmt.Errorf("rate %s already exists", rate.ID)
		}
	}
	if rate.IsDefault {
		for i := range rc.Rates {
			rc.Rates[i].IsDefault = false
		}
	}
	rc.Rates = append(rc.Rates, rate)
	return nil
}

// RemoveRate removes a rate by ID.
func (rc *RateConfig) RemoveRate(rateID string) error {
	newRates := make([]Rate, 0, len(rc.Rates))
	found := false
	for _, r := range rc.Rates {
		if r.ID == rateID {
			found = true
			continue
		}
		newRates = append(newRates, r)
	}
	if !found {
		return fmt.Errorf("rate %s not found", rateID)
	}
	rc.Rates = newRates
	return nil
}

// SetDefault marks the given rate as default, clearing default from all others.
func (rc *RateConfig) SetDefault(rateID string) error {
	found := false
	for i, r := range rc.Rates {
		if r.ID == rateID {
			found = true
			rc.Rates[i].IsDefault = true
		} else {
			rc.Rates[i].IsDefault = false
		}
	}
	if !found {
		return fmt.Errorf("rate %s not found", rateID)
	}
	return nil
}

func (rc *RateConfig) GetDefault() *Rate {
	for _, r := range rc.Rates {
		if r.IsDefault {
			return &r
		}
	}
	if len(rc.Rates) > 0 {
		return &rc.Rates[0]
	}
	return nil
}

func (rc *RateConfig) GetByID(id string) *Rate {
	for _, r := range rc.Rates {
		if r.ID == id {
			return &r
		}
	}
	return nil
}
