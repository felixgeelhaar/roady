package billing

import (
	"testing"
)

func TestNewRate(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		rateName   string
		hourlyRate float64
		isDefault  bool
		wantErr    bool
	}{
		{name: "valid rate", id: "senior", rateName: "Senior Dev", hourlyRate: 150, wantErr: false},
		{name: "zero hourly rate", id: "intern", rateName: "Intern", hourlyRate: 0, wantErr: false},
		{name: "default rate", id: "std", rateName: "Standard", hourlyRate: 100, isDefault: true, wantErr: false},
		{name: "empty ID", id: "", rateName: "Foo", hourlyRate: 100, wantErr: true},
		{name: "empty name", id: "x", rateName: "", hourlyRate: 100, wantErr: true},
		{name: "negative hourly rate", id: "x", rateName: "X", hourlyRate: -1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRate(tt.id, tt.rateName, tt.hourlyRate, tt.isDefault)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r.ID != tt.id {
				t.Errorf("expected ID %s, got %s", tt.id, r.ID)
			}
			if r.IsDefault != tt.isDefault {
				t.Errorf("expected IsDefault %v, got %v", tt.isDefault, r.IsDefault)
			}
		})
	}
}

func TestRateConfig_AddRate(t *testing.T) {
	tests := []struct {
		name      string
		initial   []Rate
		add       Rate
		wantErr   bool
		wantCount int
	}{
		{
			name:      "add to empty",
			initial:   []Rate{},
			add:       Rate{ID: "a", Name: "A", HourlyRate: 100},
			wantCount: 1,
		},
		{
			name:    "duplicate ID",
			initial: []Rate{{ID: "a", Name: "A", HourlyRate: 100}},
			add:     Rate{ID: "a", Name: "B", HourlyRate: 200},
			wantErr: true,
		},
		{
			name:      "new default clears old",
			initial:   []Rate{{ID: "a", Name: "A", HourlyRate: 100, IsDefault: true}},
			add:       Rate{ID: "b", Name: "B", HourlyRate: 200, IsDefault: true},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &RateConfig{Rates: tt.initial}
			err := cfg.AddRate(tt.add)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cfg.Rates) != tt.wantCount {
				t.Errorf("expected %d rates, got %d", tt.wantCount, len(cfg.Rates))
			}
			// Verify single default
			if tt.add.IsDefault {
				defaultCount := 0
				for _, r := range cfg.Rates {
					if r.IsDefault {
						defaultCount++
					}
				}
				if defaultCount != 1 {
					t.Errorf("expected 1 default, got %d", defaultCount)
				}
			}
		})
	}
}

func TestRateConfig_RemoveRate(t *testing.T) {
	tests := []struct {
		name      string
		initial   []Rate
		removeID  string
		wantErr   bool
		wantCount int
	}{
		{
			name:      "remove existing",
			initial:   []Rate{{ID: "a", Name: "A", HourlyRate: 100}},
			removeID:  "a",
			wantCount: 0,
		},
		{
			name:    "remove nonexistent",
			initial: []Rate{{ID: "a", Name: "A", HourlyRate: 100}},
			removeID: "z",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &RateConfig{Rates: tt.initial}
			err := cfg.RemoveRate(tt.removeID)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cfg.Rates) != tt.wantCount {
				t.Errorf("expected %d rates, got %d", tt.wantCount, len(cfg.Rates))
			}
		})
	}
}

func TestRateConfig_SetDefault(t *testing.T) {
	tests := []struct {
		name    string
		initial []Rate
		setID   string
		wantErr bool
	}{
		{
			name: "set existing as default",
			initial: []Rate{
				{ID: "a", Name: "A", HourlyRate: 100, IsDefault: true},
				{ID: "b", Name: "B", HourlyRate: 200},
			},
			setID: "b",
		},
		{
			name:    "set nonexistent",
			initial: []Rate{{ID: "a", Name: "A", HourlyRate: 100}},
			setID:   "z",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &RateConfig{Rates: tt.initial}
			err := cfg.SetDefault(tt.setID)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, r := range cfg.Rates {
				if r.ID == tt.setID && !r.IsDefault {
					t.Errorf("rate %s should be default", tt.setID)
				}
				if r.ID != tt.setID && r.IsDefault {
					t.Errorf("rate %s should not be default", r.ID)
				}
			}
		})
	}
}

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
