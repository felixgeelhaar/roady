package billing

import (
	"testing"
)

func TestRateConfig_GetDefault(t *testing.T) {
	tests := []struct {
		name     string
		rates    []Rate
		expected *Rate
	}{
		{
			name:     "empty rates",
			rates:    []Rate{},
			expected: nil,
		},
		{
			name: "no default returns first",
			rates: []Rate{
				{ID: "a", Name: "A", HourlyRate: 100},
				{ID: "b", Name: "B", HourlyRate: 200},
			},
			expected: &Rate{ID: "a", Name: "A", HourlyRate: 100},
		},
		{
			name: "returns default",
			rates: []Rate{
				{ID: "a", Name: "A", HourlyRate: 100},
				{ID: "b", Name: "B", HourlyRate: 200, IsDefault: true},
			},
			expected: &Rate{ID: "b", Name: "B", HourlyRate: 200, IsDefault: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &RateConfig{Rates: tt.rates}
			result := cfg.GetDefault()
			if result == nil && tt.expected != nil {
				t.Errorf("expected %v but got nil", tt.expected)
			}
			if result != nil && tt.expected == nil {
				t.Errorf("expected nil but got %v", result)
			}
			if result != nil && tt.expected != nil && result.ID != tt.expected.ID {
				t.Errorf("expected ID %s but got %s", tt.expected.ID, result.ID)
			}
		})
	}
}

func TestRateConfig_GetByID(t *testing.T) {
	cfg := &RateConfig{
		Rates: []Rate{
			{ID: "senior", Name: "Senior Developer", HourlyRate: 150},
			{ID: "junior", Name: "Junior Developer", HourlyRate: 75},
		},
	}

	tests := []struct {
		name     string
		id       string
		expected *Rate
	}{
		{
			name:     "found senior",
			id:       "senior",
			expected: &Rate{ID: "senior", Name: "Senior Developer", HourlyRate: 150},
		},
		{
			name:     "found junior",
			id:       "junior",
			expected: &Rate{ID: "junior", Name: "Junior Developer", HourlyRate: 75},
		},
		{
			name:     "not found",
			id:       "nonexistent",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.GetByID(tt.id)
			if result == nil && tt.expected != nil {
				t.Errorf("expected %v but got nil", tt.expected)
			}
			if result != nil && tt.expected == nil {
				t.Errorf("expected nil but got %v", result)
			}
			if result != nil && result.ID != tt.id {
				t.Errorf("expected ID %s but got %s", tt.id, result.ID)
			}
		})
	}
}
