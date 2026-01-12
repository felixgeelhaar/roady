package application_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/internal/application"
	"github.com/felixgeelhaar/roady/internal/domain"
	"github.com/felixgeelhaar/roady/internal/domain/ai"
	"github.com/felixgeelhaar/roady/internal/domain/spec"
	"github.com/felixgeelhaar/roady/internal/infrastructure/storage"
)

type MockProvider struct{
	Fail bool
}

func (m *MockProvider) ID() string { return "mock" }
func (m *MockProvider) Complete(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResponse, error) {
	if m.Fail {
		return nil, errors.New("ai fail")
	}
	return &ai.CompletionResponse{
		Text: `[{"id": "t1", "title": "Mock Task"}]`,
		Model: "mock-model",
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
		Spec: &spec.ProductSpec{ID: "test"},
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
