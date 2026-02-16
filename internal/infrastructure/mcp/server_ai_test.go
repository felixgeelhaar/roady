package mcp

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain"
	ai "github.com/felixgeelhaar/roady/pkg/domain/ai"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

type mockAIProvider struct {
	response string
}

func (m *mockAIProvider) ID() string { return "mock" }

func (m *mockAIProvider) Complete(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResponse, error) {
	return &ai.CompletionResponse{Text: m.response}, nil
}

func setupAIRepo(t *testing.T) *storage.FilesystemRepository {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	productSpec := &spec.ProductSpec{
		ID:          "spec-1",
		Title:       "AI Project",
		Description: "AI test project",
		Features: []spec.Feature{
			{ID: "feat-1", Title: "Feature 1"},
		},
	}
	if err := repo.SaveSpec(productSpec); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	plan := &planning.Plan{
		ID:             "plan-1",
		SpecID:         productSpec.ID,
		ApprovalStatus: planning.ApprovalApproved,
		Tasks: []planning.Task{
			{ID: "task-1", Title: "Task 1", FeatureID: "feat-1", Priority: planning.PriorityLow},
		},
	}
	if err := repo.SavePlan(plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	state := planning.NewExecutionState(plan.ID)
	if err := repo.SaveState(state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	return repo
}

func TestServer_HandleQuery(t *testing.T) {
	repo := setupAIRepo(t)
	provider := &mockAIProvider{response: "Answer from AI"}
	auditSvc := application.NewAuditService(repo)
	planSvc := application.NewPlanService(repo, auditSvc)
	aiSvc := application.NewAIPlanningService(repo, provider, auditSvc, planSvc)

	server := &Server{aiSvc: aiSvc}

	result, err := server.handleQuery(context.Background(), QueryArgs{Question: "What is this project?"})
	if err != nil {
		t.Fatalf("handleQuery failed: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty answer")
	}
}

func TestServer_HandleSuggestPriorities(t *testing.T) {
	repo := setupAIRepo(t)
	provider := &mockAIProvider{response: `{"suggestions":[{"task_id":"task-1","current_priority":"low","suggested_priority":"high","reason":"deps"}],"summary":"ok"}`}
	auditSvc := application.NewAuditService(repo)
	planSvc := application.NewPlanService(repo, auditSvc)
	aiSvc := application.NewAIPlanningService(repo, provider, auditSvc, planSvc)

	server := &Server{aiSvc: aiSvc}

	result, err := server.handleSuggestPriorities(context.Background(), SuggestPrioritiesArgs{})
	if err != nil {
		t.Fatalf("handleSuggestPriorities failed: %v", err)
	}

	if result == nil {
		t.Error("expected non-nil suggestions")
	}
}

func TestServer_HandleReviewSpec(t *testing.T) {
	repo := setupAIRepo(t)
	provider := &mockAIProvider{response: `{"score":90,"summary":"ok","findings":[]}`}
	auditSvc := application.NewAuditService(repo)
	planSvc := application.NewPlanService(repo, auditSvc)
	aiSvc := application.NewAIPlanningService(repo, provider, auditSvc, planSvc)

	server := &Server{aiSvc: aiSvc}

	result, err := server.handleReviewSpec(context.Background(), ReviewSpecArgs{})
	if err != nil {
		t.Fatalf("handleReviewSpec failed: %v", err)
	}

	if result == nil {
		t.Error("expected non-nil review")
	}
}
