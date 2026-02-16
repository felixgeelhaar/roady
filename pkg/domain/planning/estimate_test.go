package planning_test

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

func TestParseEstimate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantHrs float64
	}{
		{"30 minutes", "30m", false, 0.5},
		{"1 hour", "1h", false, 1},
		{"4 hours", "4h", false, 4},
		{"1 day", "1d", false, 8}, // 8-hour day
		{"2 days", "2d", false, 16},
		{"1 week", "1w", false, 40}, // 5 * 8 = 40 hours
		{"empty", "", false, 0},
		{"with spaces", "  4h  ", false, 4},
		{"uppercase", "4H", false, 4},
		{"invalid unit", "4x", true, 0},
		{"no number", "h", true, 0},
		{"just number", "4", true, 0},
		{"fraction", "1.5h", false, 1.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := planning.ParseEstimate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEstimate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && e.Hours() != tt.wantHrs {
				t.Errorf("Hours() = %v, want %v", e.Hours(), tt.wantHrs)
			}
		})
	}
}

func TestEstimate_Duration(t *testing.T) {
	e := planning.MustParseEstimate("4h")
	if e.Duration() != 4*time.Hour {
		t.Errorf("expected 4 hours duration, got %v", e.Duration())
	}
}

func TestEstimate_Days(t *testing.T) {
	e := planning.MustParseEstimate("2d")
	if e.Days() != 2 {
		t.Errorf("expected 2 days, got %v", e.Days())
	}
}

func TestEstimate_IsZero(t *testing.T) {
	empty, _ := planning.ParseEstimate("")
	if !empty.IsZero() {
		t.Error("expected empty estimate to be zero")
	}

	e := planning.MustParseEstimate("4h")
	if e.IsZero() {
		t.Error("expected non-empty estimate to not be zero")
	}
}

func TestEstimate_Add(t *testing.T) {
	e1 := planning.MustParseEstimate("4h")
	e2 := planning.MustParseEstimate("2h")
	sum := e1.Add(e2)

	if sum.Hours() != 6 {
		t.Errorf("expected 6 hours, got %v", sum.Hours())
	}

	// Adding day estimates
	d1 := planning.MustParseEstimate("1d")
	d2 := planning.MustParseEstimate("2d")
	daySum := d1.Add(d2)

	if daySum.Days() != 3 {
		t.Errorf("expected 3 days, got %v", daySum.Days())
	}
}

func TestEstimate_Compare(t *testing.T) {
	small := planning.MustParseEstimate("1h")
	medium := planning.MustParseEstimate("4h")
	large := planning.MustParseEstimate("1d")
	equal := planning.MustParseEstimate("1h")

	if small.Compare(medium) != -1 {
		t.Error("expected 1h < 4h")
	}
	if large.Compare(medium) != 1 {
		t.Error("expected 1d > 4h")
	}
	if small.Compare(equal) != 0 {
		t.Error("expected 1h == 1h")
	}
}

func TestEstimate_String(t *testing.T) {
	e := planning.MustParseEstimate("4h")
	if e.String() != "4h" {
		t.Errorf("expected '4h', got '%s'", e.String())
	}
}

func TestMustParseEstimate_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid estimate")
		}
	}()
	planning.MustParseEstimate("invalid")
}
