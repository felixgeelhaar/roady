package planning

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// estimatePattern matches estimate strings like "4h", "2d", "1w", "30m"
var estimatePattern = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(m|h|d|w)$`)

// Duration constants for estimates
const (
	MinutesPerHour = 60
	HoursPerDay    = 8 // Assume 8-hour work day
	DaysPerWeek    = 5 // Assume 5-day work week
)

// Estimate represents a time estimate for a task.
type Estimate struct {
	raw      string
	duration time.Duration
}

// ParseEstimate parses a string estimate into an Estimate value object.
// Supported formats: "30m", "4h", "2d", "1w"
func ParseEstimate(s string) (Estimate, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return Estimate{}, nil // Empty estimate is valid
	}

	matches := estimatePattern.FindStringSubmatch(s)
	if matches == nil {
		return Estimate{}, fmt.Errorf("invalid estimate format: %s (expected: 30m, 4h, 2d, or 1w)", s)
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return Estimate{}, fmt.Errorf("invalid estimate value: %s", matches[1])
	}

	unit := matches[2]
	var duration time.Duration

	switch unit {
	case "m":
		duration = time.Duration(value * float64(time.Minute))
	case "h":
		duration = time.Duration(value * float64(time.Hour))
	case "d":
		duration = time.Duration(value * float64(HoursPerDay) * float64(time.Hour))
	case "w":
		duration = time.Duration(value * float64(DaysPerWeek) * float64(HoursPerDay) * float64(time.Hour))
	}

	return Estimate{raw: s, duration: duration}, nil
}

// MustParseEstimate parses an estimate or panics. Use only in tests.
func MustParseEstimate(s string) Estimate {
	e, err := ParseEstimate(s)
	if err != nil {
		panic(err)
	}
	return e
}

// String returns the original string representation of the estimate.
func (e Estimate) String() string {
	return e.raw
}

// Duration returns the duration of the estimate.
func (e Estimate) Duration() time.Duration {
	return e.duration
}

// Hours returns the estimate in hours.
func (e Estimate) Hours() float64 {
	return e.duration.Hours()
}

// Days returns the estimate in work days (8-hour days).
func (e Estimate) Days() float64 {
	return e.duration.Hours() / float64(HoursPerDay)
}

// IsZero returns true if the estimate is empty.
func (e Estimate) IsZero() bool {
	return e.raw == ""
}

// Add adds two estimates together.
func (e Estimate) Add(other Estimate) Estimate {
	total := e.duration + other.duration
	// Format as hours if less than a day, else as days
	hours := total.Hours()
	if hours < float64(HoursPerDay) {
		return Estimate{raw: fmt.Sprintf("%.1fh", hours), duration: total}
	}
	days := hours / float64(HoursPerDay)
	return Estimate{raw: fmt.Sprintf("%.1fd", days), duration: total}
}

// Compare compares two estimates.
// Returns -1 if e < other, 0 if equal, 1 if e > other.
func (e Estimate) Compare(other Estimate) int {
	if e.duration < other.duration {
		return -1
	}
	if e.duration > other.duration {
		return 1
	}
	return 0
}
