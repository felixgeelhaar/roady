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

	service := application.NewAIPlanningService(repo, &MockProvider{}, audit)

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

	// 1. AI Fail
	service := application.NewAIPlanningService(repo, &MockProvider{Fail: true}, audit)
	_, err := service.DecomposeSpec(context.Background())
	if err == nil {
		t.Error("expected ai failure")
	}

	// 2. Policy load fail
	service2 := application.NewAIPlanningService(repo, &MockProvider{}, audit)
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
	provider := &MockProvider{
		Text: `{"id":"proj","title":"Reconciled","description":"Desc","version":"0.1.0","features":[{"id":"f1","title":"F1","description":"D","requirements":[{"id":"r1","title":"R1","description":"RD","priority":"low","estimate":"1h"}]}]}`,
	}
	service := application.NewAIPlanningService(repo, provider, audit)

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
	service := application.NewAIPlanningService(repo, &MockProvider{Text: "Explanation"}, audit)

	out, err := service.ExplainSpec(context.Background())
	if err != nil {
		t.Fatalf("ExplainSpec failed: %v", err)
	}
	if out != "Explanation" {
		t.Fatalf("expected explanation, got %q", out)
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
	service := application.NewAIPlanningService(repo, &MockProvider{Text: "Drift explanation"}, audit)

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

func TestAIPlanningService_GetAuditService(t *testing.T) {
	repo := &MockRepo{}
	audit := application.NewAuditService(repo)
	service := application.NewAIPlanningService(repo, &MockProvider{}, audit)

	if service.GetAuditService() != audit {
		t.Fatal("expected audit service to match")
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

	service := application.NewAIPlanningService(repo, &MockProvider{}, audit)
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
	repo.SavePolicy(&domain.PolicyConfig{AllowAI: true})
	repo.SaveSpec(&spec.ProductSpec{ID: "test", Title: "AI Test", Features: []spec.Feature{{ID: "f1", Title: "F1"}}})

	service := application.NewAIPlanningService(repo, &MockProvider{Text: `{"tasks":[{"id":"t1","title":"Wrapped Task"}]}`}, audit)
	if _, err := service.DecomposeSpec(context.Background()); err != nil {
		t.Fatalf("expected wrapper JSON to parse, got %v", err)
	}

	service = application.NewAIPlanningService(repo, &MockProvider{Text: `{"t2":{"id":"t2","title":"Map Task"}}`}, audit)
	if _, err := service.DecomposeSpec(context.Background()); err != nil {
		t.Fatalf("expected map JSON to parse, got %v", err)
	}
}
