package application_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/ai"
	"github.com/felixgeelhaar/roady/pkg/domain/drift"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

type MockProvider struct {
	Fail bool
	Text string
}

func (m *MockProvider) ID() string { return "mock" }
func (m *MockProvider) Complete(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResponse, error) {
	if m.Fail {
		return nil, errors.New("ai fail")
	}
	text := m.Text
	if text == "" {
		text = `[{"id": "t1", "title": "Mock Task"}]`
	}
	return &ai.CompletionResponse{
		Text:  text,
		Model: "mock-model",
		Usage: ai.TokenUsage{
			InputTokens:  5,
			OutputTokens: 3,
		},
	}, nil
}

func TestAIPlanningService_Decompose(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-ai-test-*")
	defer os.RemoveAll(tempDir)

	repo := storage.NewFilesystemRepository(tempDir)
	repo.Initialize()
	audit := application.NewAuditService(repo)

	// Success Path
	repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 3, AllowAI: true})
	repo.SaveSpec(&spec.ProductSpec{ID: "test", Title: "AI Test"})

	planSvc := application.NewPlanService(repo, audit)
	service := application.NewAIPlanningService(repo, &MockProvider{}, audit, planSvc)

	plan, err := service.DecomposeSpec(context.Background())
	if err != nil {
		t.Fatalf("Expected success, got: %v", err)
	}
	if len(plan.Tasks) != 1 || plan.Tasks[0].ID != "t1" {
		t.Errorf("Unexpected tasks in plan: %+v", plan.Tasks)
	}

	// Policy DISABLES AI
	repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 3, AllowAI: false})
	_, err = service.DecomposeSpec(context.Background())
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Errorf("Expected policy disabled error, got: %v", err)
	}
}

func TestAIPlanningService_Errors_Mock(t *testing.T) {
	repo := &MockRepo{
		Policy: &domain.PolicyConfig{AllowAI: true},
		Spec:   &spec.ProductSpec{ID: "test"},
	}
	audit := application.NewAuditService(repo)
	planSvc := application.NewPlanService(repo, audit)

	// 1. AI Fail
	service := application.NewAIPlanningService(repo, &MockProvider{Fail: true}, audit, planSvc)
	_, err := service.DecomposeSpec(context.Background())
	if err == nil {
		t.Error("expected ai failure")
	}

	// 2. Policy load fail
	service2 := application.NewAIPlanningService(repo, &MockProvider{}, audit, planSvc)
	repo.LoadError = errors.New("policy fail")
	_, err = service2.DecomposeSpec(context.Background())
	if err == nil {
		t.Error("expected policy load error")
	}

	// 3. Spec load fail
	repo.LoadError = errors.New("spec fail")
	_, err = service2.DecomposeSpec(context.Background())
	if err == nil {
		t.Error("expected spec load error")
	}
}

func TestAIPlanningService_ReconcileSpec(t *testing.T) {
	repo := &MockRepo{
		Policy: &domain.PolicyConfig{AllowAI: true},
	}
	audit := application.NewAuditService(repo)
	planSvc := application.NewPlanService(repo, audit)
	provider := &MockProvider{
		Text: `{"id":"proj","title":"Reconciled","description":"Desc","version":"0.1.0","features":[{"id":"f1","title":"F1","description":"D","requirements":[{"id":"r1","title":"R1","description":"RD","priority":"low","estimate":"1h"}]}]}`,
	}
	service := application.NewAIPlanningService(repo, provider, audit, planSvc)

	out, err := service.ReconcileSpec(context.Background(), &spec.ProductSpec{
		Title: "Raw",
		Features: []spec.Feature{
			{ID: "f1", Title: "F1", Description: "D1"},
		},
	})
	if err != nil {
		t.Fatalf("ReconcileSpec failed: %v", err)
	}
	if out.Title != "Reconciled" {
		t.Fatalf("expected reconciled title, got %q", out.Title)
	}
}

func TestAIPlanningService_ExplainSpec(t *testing.T) {
	repo := &MockRepo{
		Policy: &domain.PolicyConfig{AllowAI: true},
		Spec: &spec.ProductSpec{
			ID:    "s1",
			Title: "Spec",
			Features: []spec.Feature{
				{ID: "f1", Title: "F1", Description: "Desc"},
			},
		},
	}
	audit := application.NewAuditService(repo)
	planSvc := application.NewPlanService(repo, audit)
	service := application.NewAIPlanningService(repo, &MockProvider{Text: "Explanation"}, audit, planSvc)

	out, err := service.ExplainSpec(context.Background())
	if err != nil {
		t.Fatalf("ExplainSpec failed: %v", err)
	}
	if out != "Explanation" {
		t.Fatalf("expected explanation, got %q", out)
	}
}

func TestAIPlanningService_QueryProject(t *testing.T) {
	repo := &MockRepo{
		Policy: &domain.PolicyConfig{AllowAI: true},
		Spec: &spec.ProductSpec{
			ID:    "s1",
			Title: "Test Spec",
			Features: []spec.Feature{
				{ID: "f1", Title: "Auth", Description: "Authentication system"},
			},
		},
		Plan: &planning.Plan{
			ID:    "plan-1",
			Tasks: []planning.Task{{ID: "t1", Title: "Setup DB", Priority: "medium"}},
		},
	}
	audit := application.NewAuditService(repo)
	planSvc := application.NewPlanService(repo, audit)
	service := application.NewAIPlanningService(repo, &MockProvider{Text: "There is 1 task pending."}, audit, planSvc)

	answer, err := service.QueryProject(context.Background(), "How many tasks are pending?")
	if err != nil {
		t.Fatalf("QueryProject failed: %v", err)
	}
	if answer != "There is 1 task pending." {
		t.Errorf("unexpected answer: %q", answer)
	}
}

func TestAIPlanningService_SuggestPriorities(t *testing.T) {
	repo := &MockRepo{
		Policy: &domain.PolicyConfig{AllowAI: true},
		Spec: &spec.ProductSpec{
			ID:    "s1",
			Title: "Test Spec",
		},
		Plan: &planning.Plan{
			ID: "plan-1",
			Tasks: []planning.Task{
				{ID: "t1", Title: "Setup DB", Priority: "low", FeatureID: "f1"},
				{ID: "t2", Title: "Build API", Priority: "medium", FeatureID: "f1", DependsOn: []string{"t1"}},
			},
		},
	}
	audit := application.NewAuditService(repo)
	planSvc := application.NewPlanService(repo, audit)

	suggestJSON := `{"suggestions":[{"task_id":"t1","current_priority":"low","suggested_priority":"high","reason":"Blocks t2"}],"summary":"t1 should be high priority as it blocks other work."}`
	service := application.NewAIPlanningService(repo, &MockProvider{Text: suggestJSON}, audit, planSvc)

	result, err := service.SuggestPriorities(context.Background())
	if err != nil {
		t.Fatalf("SuggestPriorities failed: %v", err)
	}
	if len(result.Suggestions) != 1 {
		t.Errorf("expected 1 suggestion, got %d", len(result.Suggestions))
	}
	if result.Suggestions[0].SuggestedPriority != "high" {
		t.Errorf("expected high, got %q", result.Suggestions[0].SuggestedPriority)
	}
}

func TestAIPlanningService_ReviewSpec(t *testing.T) {
	repo := &MockRepo{
		Policy: &domain.PolicyConfig{AllowAI: true},
		Spec: &spec.ProductSpec{
			ID:    "s1",
			Title: "Test Spec",
			Features: []spec.Feature{
				{ID: "f1", Title: "Auth", Description: "Authentication system"},
			},
		},
	}
	audit := application.NewAuditService(repo)
	planSvc := application.NewPlanService(repo, audit)

	reviewJSON := `{"score":78,"summary":"Good spec with minor gaps.","findings":[{"category":"completeness","severity":"warning","feature_id":"f1","title":"Missing error cases","suggestion":"Add error handling requirements."}]}`
	service := application.NewAIPlanningService(repo, &MockProvider{Text: reviewJSON}, audit, planSvc)

	review, err := service.ReviewSpec(context.Background())
	if err != nil {
		t.Fatalf("ReviewSpec failed: %v", err)
	}
	if review.Score != 78 {
		t.Errorf("expected score 78, got %d", review.Score)
	}
	if len(review.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(review.Findings))
	}
	if review.Findings[0].Category != "completeness" {
		t.Errorf("expected category completeness, got %q", review.Findings[0].Category)
	}
}

func TestAIPlanningService_ReviewSpec_PolicyDisabled(t *testing.T) {
	repo := &MockRepo{
		Policy: &domain.PolicyConfig{AllowAI: false},
		Spec:   &spec.ProductSpec{ID: "s1", Title: "Test"},
	}
	audit := application.NewAuditService(repo)
	planSvc := application.NewPlanService(repo, audit)
	service := application.NewAIPlanningService(repo, &MockProvider{}, audit, planSvc)

	_, err := service.ReviewSpec(context.Background())
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Errorf("expected policy disabled error, got: %v", err)
	}
}

func TestAIPlanningService_ExplainDrift(t *testing.T) {
	repo := &MockRepo{
		Policy: &domain.PolicyConfig{AllowAI: true},
		Spec: &spec.ProductSpec{
			ID:    "s1",
			Title: "Spec",
		},
	}
	audit := application.NewAuditService(repo)
	planSvc := application.NewPlanService(repo, audit)
	service := application.NewAIPlanningService(repo, &MockProvider{Text: "Drift explanation"}, audit, planSvc)

	report := &drift.Report{
		Issues: []drift.Issue{
			{Type: "spec", Severity: "medium", Message: "Mismatch", ComponentID: "spec"},
		},
	}
	out, err := service.ExplainDrift(context.Background(), report)
	if err != nil {
		t.Fatalf("ExplainDrift failed: %v", err)
	}
	if out != "Drift explanation" {
		t.Fatalf("expected explanation, got %q", out)
	}

	empty, err := service.ExplainDrift(context.Background(), &drift.Report{})
	if err != nil {
		t.Fatalf("ExplainDrift empty failed: %v", err)
	}
	if !strings.Contains(empty, "No drift") {
		t.Fatalf("expected no drift message, got %q", empty)
	}
}

func TestAIPlanningService_GetAuditLogger(t *testing.T) {
	repo := &MockRepo{}
	audit := application.NewAuditService(repo)
	planSvc := application.NewPlanService(repo, audit)
	service := application.NewAIPlanningService(repo, &MockProvider{}, audit, planSvc)

	if service.GetAuditLogger() != audit {
		t.Fatal("expected audit logger to match")
	}
}

func TestAIPlanningService_DecomposeSpec_TokenLimit(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-ai-limit-*")
	defer os.RemoveAll(tempDir)

	repo := storage.NewFilesystemRepository(tempDir)
	repo.Initialize()
	audit := application.NewAuditService(repo)

	repo.SavePolicy(&domain.PolicyConfig{AllowAI: true, TokenLimit: 1})
	repo.SaveSpec(&spec.ProductSpec{ID: "test", Title: "AI Test"})
	_ = repo.UpdateUsage(domain.UsageStats{
		ProviderStats: map[string]int{"mock:input": 2},
	})

	planSvc := application.NewPlanService(repo, audit)
	service := application.NewAIPlanningService(repo, &MockProvider{}, audit, planSvc)
	if _, err := service.DecomposeSpec(context.Background()); err == nil {
		t.Fatal("expected token limit error")
	}
}

func TestAIPlanningService_DecomposeSpec_JSONVariants(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-ai-json-*")
	defer os.RemoveAll(tempDir)

	repo := storage.NewFilesystemRepository(tempDir)
	repo.Initialize()
	audit := application.NewAuditService(repo)
	planSvc := application.NewPlanService(repo, audit)
	repo.SavePolicy(&domain.PolicyConfig{AllowAI: true})
	repo.SaveSpec(&spec.ProductSpec{ID: "test", Title: "AI Test", Features: []spec.Feature{{ID: "f1", Title: "F1"}}})

	service := application.NewAIPlanningService(repo, &MockProvider{Text: `{"tasks":[{"id":"t1","title":"Wrapped Task"}]}`}, audit, planSvc)
	if _, err := service.DecomposeSpec(context.Background()); err != nil {
		t.Fatalf("expected wrapper JSON to parse, got %v", err)
	}

	service = application.NewAIPlanningService(repo, &MockProvider{Text: `{"t2":{"id":"t2","title":"Map Task"}}`}, audit, planSvc)
	if _, err := service.DecomposeSpec(context.Background()); err != nil {
		t.Fatalf("expected map JSON to parse, got %v", err)
	}
}

func TestAIPlanningService_DecomposeSpec_MissingFeatureFallback(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-ai-fallback-*")
	defer os.RemoveAll(tempDir)

	repo := storage.NewFilesystemRepository(tempDir)
	repo.Initialize()
	audit := application.NewAuditService(repo)

	spec := &spec.ProductSpec{
		ID:    "fallback",
		Title: "Fallback Spec",
		Features: []spec.Feature{
			{ID: "feature-a", Title: "Feature A", Requirements: []spec.Requirement{{ID: "req-a", Title: "Req A"}}},
			{ID: "feature-b", Title: "Feature B", Requirements: []spec.Requirement{{ID: "req-b", Title: "Req B"}}},
		},
	}
	if err := repo.SaveSpec(spec); err != nil {
		t.Fatalf("save spec: %v", err)
	}
	repo.SavePolicy(&domain.PolicyConfig{AllowAI: true})

	provider := &MockProvider{
		Text: `[{"id":"task-req-a","title":"Task A","feature_id":"feature-a"}]`,
	}
	planSvc := application.NewPlanService(repo, audit)
	service := application.NewAIPlanningService(repo, provider, audit, planSvc)

	plan, err := service.DecomposeSpec(context.Background())
	if err != nil {
		t.Fatalf("fallback decomposition failed: %v", err)
	}

	foundFallback := false
	for _, t := range plan.Tasks {
		if t.ID == "task-feature-b" {
			foundFallback = true
		}
	}
	if !foundFallback {
		t.Fatalf("expected fallback task for missing feature, tasks: %+v", plan.Tasks)
	}
}

type retryProvider struct {
	attempt int
}

func (rp *retryProvider) ID() string { return "retry-provider" }
func (rp *retryProvider) Complete(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResponse, error) {
	rp.attempt++
	if rp.attempt == 1 {
		return &ai.CompletionResponse{
			Text:  "invalid response",
			Model: "retry-model",
			Usage: ai.TokenUsage{InputTokens: 1, OutputTokens: 1},
		}, nil
	}
	return &ai.CompletionResponse{
		Text:  `[{"id":"task-req-gov","title":"Governance","feature_id":"feature-gov"}]`,
		Model: "retry-model",
		Usage: ai.TokenUsage{InputTokens: 1, OutputTokens: 1},
	}, nil
}

func TestAIPlanningService_DecomposeSpec_RetryLogsEvent(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-ai-retry-*")
	defer os.RemoveAll(tempDir)

	repo := storage.NewFilesystemRepository(tempDir)
	repo.Initialize()
	audit := application.NewAuditService(repo)

	repo.SavePolicy(&domain.PolicyConfig{AllowAI: true})
	repo.SaveSpec(&spec.ProductSpec{
		ID:    "retry-spec",
		Title: "Retry Spec",
		Features: []spec.Feature{
			{ID: "feature-gov", Title: "Governance Feature"},
		},
	})

	planSvc := application.NewPlanService(repo, audit)
	service := application.NewAIPlanningService(repo, &retryProvider{}, audit, planSvc)

	if _, err := service.DecomposeSpec(context.Background()); err != nil {
		t.Fatalf("some retry failed: %v", err)
	}

	events, err := repo.LoadEvents()
	if err != nil {
		t.Fatalf("load events: %v", err)
	}

	found := false
	for _, ev := range events {
		if ev.Action == "plan.ai_decomposition_retry" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected retry event in events: %+v", events)
	}
}
