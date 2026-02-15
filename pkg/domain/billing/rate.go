package billing

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
