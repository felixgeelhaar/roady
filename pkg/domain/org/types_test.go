package org_test

import (
	"encoding/json"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/org"
)

func TestSharedPolicy_Fields(t *testing.T) {
	p := org.SharedPolicy{
		MaxWIP:      5,
		AllowAI:     true,
		TokenLimit:  10000,
		BudgetHours: 40,
	}

	if p.MaxWIP != 5 {
		t.Errorf("MaxWIP = %d, want 5", p.MaxWIP)
	}
	if !p.AllowAI {
		t.Error("AllowAI = false, want true")
	}
	if p.TokenLimit != 10000 {
		t.Errorf("TokenLimit = %d, want 10000", p.TokenLimit)
	}
	if p.BudgetHours != 40 {
		t.Errorf("BudgetHours = %d, want 40", p.BudgetHours)
	}

	// JSON round-trip
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded org.SharedPolicy
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded != p {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, p)
	}
}

func TestSharedPolicy_Fields_OmitEmpty(t *testing.T) {
	p := org.SharedPolicy{}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	// All fields have omitempty, so zero-value struct should produce minimal JSON.
	expected := `{}`
	if string(data) != expected {
		t.Errorf("zero-value JSON = %s, want %s", string(data), expected)
	}
}

func TestOrgConfig_Fields(t *testing.T) {
	policy := &org.SharedPolicy{MaxWIP: 3, AllowAI: true}
	cfg := org.OrgConfig{
		Name:         "my-org",
		Repos:        []string{"repo-a", "repo-b"},
		SharedPolicy: policy,
	}

	if cfg.Name != "my-org" {
		t.Errorf("Name = %q, want %q", cfg.Name, "my-org")
	}
	if len(cfg.Repos) != 2 {
		t.Errorf("len(Repos) = %d, want 2", len(cfg.Repos))
	}
	if cfg.SharedPolicy == nil {
		t.Fatal("SharedPolicy is nil")
	}
	if cfg.SharedPolicy.MaxWIP != 3 {
		t.Errorf("SharedPolicy.MaxWIP = %d, want 3", cfg.SharedPolicy.MaxWIP)
	}

	// JSON round-trip
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded org.OrgConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Name != cfg.Name {
		t.Errorf("Name mismatch after round-trip: %q vs %q", decoded.Name, cfg.Name)
	}
	if len(decoded.Repos) != len(cfg.Repos) {
		t.Errorf("Repos length mismatch: %d vs %d", len(decoded.Repos), len(cfg.Repos))
	}
}

func TestOrgConfig_Fields_NilPolicy(t *testing.T) {
	cfg := org.OrgConfig{
		Name:  "bare-org",
		Repos: []string{},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded org.OrgConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.SharedPolicy != nil {
		t.Error("expected nil SharedPolicy after round-trip")
	}
}

func TestProjectMetrics_Fields(t *testing.T) {
	m := org.ProjectMetrics{
		Name:       "web-app",
		Path:       "/projects/web-app",
		Progress:   0.75,
		Verified:   3,
		WIP:        2,
		Done:       5,
		Pending:    4,
		Blocked:    1,
		Total:      15,
		HasDrift:   true,
		DriftCount: 2,
	}

	if m.Name != "web-app" {
		t.Errorf("Name = %q, want %q", m.Name, "web-app")
	}
	if m.Progress != 0.75 {
		t.Errorf("Progress = %f, want 0.75", m.Progress)
	}
	if m.Total != 15 {
		t.Errorf("Total = %d, want 15", m.Total)
	}
	if !m.HasDrift {
		t.Error("HasDrift = false, want true")
	}

	// JSON round-trip
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded org.ProjectMetrics
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded != m {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, m)
	}
}

func TestProjectMetrics_Fields_DriftCountOmitEmpty(t *testing.T) {
	m := org.ProjectMetrics{
		Name: "no-drift",
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	// DriftCount has omitempty, so zero value should be omitted.
	var raw map[string]interface{}
	_ = json.Unmarshal(data, &raw)
	if _, exists := raw["drift_count"]; exists {
		t.Error("drift_count should be omitted when zero")
	}
}

func TestOrgMetrics_Fields(t *testing.T) {
	m := org.OrgMetrics{
		OrgName: "acme-corp",
		Projects: []org.ProjectMetrics{
			{Name: "proj-a", Total: 10, Progress: 0.5},
			{Name: "proj-b", Total: 20, Progress: 0.8},
		},
		TotalProjects: 2,
		TotalTasks:    30,
		TotalVerified: 8,
		TotalWIP:      4,
		AvgProgress:   0.65,
	}

	if m.OrgName != "acme-corp" {
		t.Errorf("OrgName = %q, want %q", m.OrgName, "acme-corp")
	}
	if len(m.Projects) != 2 {
		t.Errorf("len(Projects) = %d, want 2", len(m.Projects))
	}
	if m.TotalProjects != 2 {
		t.Errorf("TotalProjects = %d, want 2", m.TotalProjects)
	}
	if m.AvgProgress != 0.65 {
		t.Errorf("AvgProgress = %f, want 0.65", m.AvgProgress)
	}

	// JSON round-trip
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded org.OrgMetrics
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.OrgName != m.OrgName {
		t.Errorf("OrgName mismatch: %q vs %q", decoded.OrgName, m.OrgName)
	}
	if decoded.TotalTasks != m.TotalTasks {
		t.Errorf("TotalTasks mismatch: %d vs %d", decoded.TotalTasks, m.TotalTasks)
	}
}

func TestOrgMetrics_Fields_OmitEmptyOrgName(t *testing.T) {
	m := org.OrgMetrics{}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var raw map[string]interface{}
	_ = json.Unmarshal(data, &raw)
	if _, exists := raw["org_name"]; exists {
		t.Error("org_name should be omitted when empty")
	}
}

func TestCrossDriftReport_Fields(t *testing.T) {
	r := org.CrossDriftReport{
		Projects: []org.ProjectDriftSummary{
			{Name: "api", Path: "/api", IssueCount: 3, HasDrift: true},
			{Name: "web", Path: "/web", IssueCount: 0, HasDrift: false},
		},
		TotalIssues: 3,
	}

	if len(r.Projects) != 2 {
		t.Errorf("len(Projects) = %d, want 2", len(r.Projects))
	}
	if r.TotalIssues != 3 {
		t.Errorf("TotalIssues = %d, want 3", r.TotalIssues)
	}
	if r.Projects[0].Name != "api" {
		t.Errorf("Projects[0].Name = %q, want %q", r.Projects[0].Name, "api")
	}
	if !r.Projects[0].HasDrift {
		t.Error("Projects[0].HasDrift = false, want true")
	}

	// JSON round-trip
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded org.CrossDriftReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.TotalIssues != r.TotalIssues {
		t.Errorf("TotalIssues mismatch: %d vs %d", decoded.TotalIssues, r.TotalIssues)
	}
	if len(decoded.Projects) != len(r.Projects) {
		t.Errorf("Projects length mismatch: %d vs %d", len(decoded.Projects), len(r.Projects))
	}
}

func TestProjectDriftSummary_Fields(t *testing.T) {
	s := org.ProjectDriftSummary{
		Name:       "service-x",
		Path:       "/projects/service-x",
		IssueCount: 7,
		HasDrift:   true,
	}

	if s.Name != "service-x" {
		t.Errorf("Name = %q, want %q", s.Name, "service-x")
	}
	if s.Path != "/projects/service-x" {
		t.Errorf("Path = %q, want %q", s.Path, "/projects/service-x")
	}
	if s.IssueCount != 7 {
		t.Errorf("IssueCount = %d, want 7", s.IssueCount)
	}
	if !s.HasDrift {
		t.Error("HasDrift = false, want true")
	}

	// JSON round-trip
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded org.ProjectDriftSummary
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded != s {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, s)
	}
}
