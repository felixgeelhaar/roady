package spec

// ReviewFinding represents a single quality finding in a spec review.
type ReviewFinding struct {
	Category   string `json:"category"`   // completeness, clarity, ambiguity, dependency, priority, testability
	Severity   string `json:"severity"`   // info, warning, critical
	FeatureID  string `json:"feature_id"` // empty if spec-level
	Title      string `json:"title"`
	Suggestion string `json:"suggestion"`
}

// SpecReview represents the result of an AI quality review of a spec.
type SpecReview struct {
	Score    int              `json:"score"`    // 0-100
	Summary  string           `json:"summary"`
	Findings []ReviewFinding  `json:"findings"`
}
