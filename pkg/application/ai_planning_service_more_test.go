package application

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain"
	domainai "github.com/felixgeelhaar/roady/pkg/domain/ai"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

type sequentialProvider struct {
	responses []domainai.CompletionResponse
	idx       int
}

func (p *sequentialProvider) ID() string { return "sequential" }

func (p *sequentialProvider) Complete(ctx context.Context, req domainai.CompletionRequest) (*domainai.CompletionResponse, error) {
	if p.idx >= len(p.responses) {
		return nil, fmt.Errorf("no more responses")
	}
	resp := p.responses[p.idx]
	p.idx++
	return &resp, nil
}

func (p *sequentialProvider) count() int {
	return p.idx
}

func TestAIPlanningService_DecomposeSpecRetriesOnInvalidJSON(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-ai-retry-*")
	defer os.RemoveAll(tempDir)

	repo := storage.NewFilesystemRepository(tempDir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}
	if err := repo.SaveSpec(&spec.ProductSpec{
		ID:    "retry-spec",
		Title: "Retry",
		Features: []spec.Feature{
			{
				ID:    "feature-ai",
				Title: "AI Feature",
				Requirements: []spec.Requirement{
					{ID: "req-ai", Title: "AI Req"},
				},
			},
		},
	}); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	provider := &sequentialProvider{
		responses: []domainai.CompletionResponse{
			{
				Text:  "invalid response",
				Model: "stub-model",
				Usage: domainai.TokenUsage{InputTokens: 10, OutputTokens: 2},
			},
			{
				Text:  `[{"id":"task-req-ai","feature_id":"feature-ai","title":"AI Task"}]`,
				Model: "stub-model",
				Usage: domainai.TokenUsage{InputTokens: 8, OutputTokens: 4},
			},
		},
	}

	audit := NewAuditService(repo)
	planSvc := NewPlanService(repo, audit)
	service := NewAIPlanningService(repo, provider, audit, planSvc)

	plan, err := service.DecomposeSpec(context.Background())
	if err != nil {
		t.Fatalf("DecomposeSpec failed: %v", err)
	}

	if provider.count() != 2 {
		t.Fatalf("expected 2 provider calls, got %d", provider.count())
	}
	if len(plan.Tasks) != 1 || plan.Tasks[0].ID != "task-req-ai" {
		t.Fatalf("unexpected plan tasks: %+v", plan.Tasks)
	}
}

func TestAIPlanningService_ParseTasksFromResponseDiverseFormats(t *testing.T) {
	service := &AIPlanningService{}
	tests := []struct {
		name    string
		payload string
		wantID  string
	}{
		{
			name:    "task-wrapper",
			payload: `{"task":{"id":"wrapped","feature_id":"feat","title":"Wrapped"}}`,
			wantID:  "wrapped",
		},
		{
			name:    "data-array",
			payload: `{"data":[{"id":"data-task","feature_id":"feat","title":"From Data"}]}`,
			wantID:  "data-task",
		},
		{
			name:    "task-id-style",
			payload: `{"task-id":"slug-task","feature":"feat","description":"With slug"}`,
			wantID:  "slug-task",
		},
		{
			name:    "json-enclosed",
			payload: "Look here ```json[{\"id\":\"code-task\",\"feature_id\":\"feat\",\"title\":\"Code\"}]``` end",
			wantID:  "code-task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks, err := service.parseTasksFromResponse(tt.payload)
			if err != nil {
				t.Fatalf("parse tasks failed: %v", err)
			}
			if len(tasks) == 0 || tasks[0].ID != tt.wantID {
				t.Fatalf("unexpected tasks: %+v", tasks)
			}
		})
	}
}

func TestAIPlanningService_DecomposeAddsFallbackTasks(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-ai-fallback-*")
	defer os.RemoveAll(tempDir)

	repo := storage.NewFilesystemRepository(tempDir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}
	if err := repo.SavePolicy(&domain.PolicyConfig{AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}
	if err := repo.SaveSpec(&spec.ProductSpec{
		ID:    "fallback-spec",
		Title: "Fallback",
		Features: []spec.Feature{
			{ID: "feature-a", Title: "Feature A"},
			{ID: "feature-b", Title: "Feature B"},
		},
	}); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	provider := &sequentialProvider{
		responses: []domainai.CompletionResponse{
			{
				Text:  `[{"id":"task-feature-a","feature_id":"feature-a","title":"AI Task A"}]`,
				Model: "stub-model",
				Usage: domainai.TokenUsage{InputTokens: 4, OutputTokens: 2},
			},
		},
	}

	audit := NewAuditService(repo)
	planSvc := NewPlanService(repo, audit)
	service := NewAIPlanningService(repo, provider, audit, planSvc)
	plan, err := service.DecomposeSpec(context.Background())
	if err != nil {
		t.Fatalf("DecomposeSpec failed: %v", err)
	}

	foundB := false
	for _, task := range plan.Tasks {
		if task.ID == "task-feature-b" {
			foundB = true
			break
		}
	}
	if !foundB {
		t.Fatalf("expected fallback task for feature-b, got %+v", plan.Tasks)
	}
}

type failingProvider struct{}

func (f *failingProvider) ID() string { return "fail" }

func (f *failingProvider) Complete(ctx context.Context, req domainai.CompletionRequest) (*domainai.CompletionResponse, error) {
	return nil, fmt.Errorf("boom")
}

func TestAIPlanningService_CompleteDecompositionProviderError(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-ai-error-*")
	defer os.RemoveAll(tempDir)

	repo := storage.NewFilesystemRepository(tempDir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	audit := NewAuditService(repo)
	planSvc := NewPlanService(repo, audit)
	service := NewAIPlanningService(repo, &failingProvider{}, audit, planSvc)

	if _, err := service.completeDecomposition(context.Background(), "prompt", 1); err == nil {
		t.Fatal("expected provider error")
	}
}

func TestAIPlanningService_DecomposeSpecMissingSpec(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-ai-nospec-*")
	defer os.RemoveAll(tempDir)

	repo := storage.NewFilesystemRepository(tempDir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	audit := NewAuditService(repo)
	planSvc := NewPlanService(repo, audit)
	service := NewAIPlanningService(repo, &sequentialProvider{}, audit, planSvc)
	if _, err := service.DecomposeSpec(context.Background()); err == nil {
		t.Fatalf("expected error when spec is missing")
	} else if !strings.Contains(err.Error(), "load spec") {
		t.Fatalf("unexpected error when spec missing: %v", err)
	}
}
