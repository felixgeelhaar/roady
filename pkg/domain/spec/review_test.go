package spec

import (
	"encoding/json"
	"testing"
)

func TestSpecReview_JSONRoundTrip(t *testing.T) {
	review := SpecReview{
		Score:   85,
		Summary: "Spec is well-structured with minor gaps.",
		Findings: []ReviewFinding{
			{
				Category:   "completeness",
				Severity:   "warning",
				FeatureID:  "feature-auth",
				Title:      "Missing error handling requirements",
				Suggestion: "Add requirements for authentication failure scenarios.",
			},
			{
				Category:   "clarity",
				Severity:   "info",
				FeatureID:  "",
				Title:      "Ambiguous performance targets",
				Suggestion: "Specify concrete latency and throughput targets.",
			},
		},
	}

	data, err := json.Marshal(review)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded SpecReview
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Score != 85 {
		t.Errorf("expected score 85, got %d", decoded.Score)
	}
	if len(decoded.Findings) != 2 {
		t.Errorf("expected 2 findings, got %d", len(decoded.Findings))
	}
	if decoded.Findings[0].Category != "completeness" {
		t.Errorf("expected category completeness, got %q", decoded.Findings[0].Category)
	}
}
