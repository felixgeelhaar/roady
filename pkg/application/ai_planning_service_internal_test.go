package application

import (
	"context"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/ai"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestSummarizeTextAndHumanizeHelpers(t *testing.T) {
	if got := summarizeText("Short sentence."); got != "Short sentence" {
		t.Fatalf("unexpected summary: %q", got)
	}
	if got := summarizeText("Long text without a period " +
		"that certainly exceeds the eighty character limit by a little bit"); !strings.HasSuffix(got, "…") {
		t.Fatalf("expected ellipsis for long summary, got %q", got)
	}

	if human := humanizeID("task-alpha_beta"); human != "Alpha Beta" {
		t.Fatalf("expected humanized id, got %q", human)
	}
	if slug := slugify("  Some Title! 🌟 "); slug != "some-title" {
		t.Fatalf("unexpected slug output: %q", slug)
	}
}

func TestNormalizeTaskMapFallbacksAndHelpers(t *testing.T) {
	raw := map[string]interface{}{
		"description": "Detailed summary. Follow-up details",
		"feature":     "feat-1",
	}

	task := normalizeTaskMap(raw, 0)
	if task.ID != "task-detailed-summary" {
		t.Fatalf("expected generated id, got %q", task.ID)
	}
	if task.FeatureID != "feat-1" || task.Title != "Detailed summary" {
		t.Fatalf("unexpected feature/title: %+v", task)
	}

	if !hasAnyKey(raw, "description", "missing") {
		t.Fatal("expected hasAnyKey to detect description")
	}

	payload := extractJSONPayload("\nSome text ```json[{\"id\":\"t1\"}]```more")
	if payload != "[{\"id\":\"t1\"}]" {
		t.Fatalf("unexpected payload: %q", payload)
	}
}

type stubProvider struct {
	response ai.CompletionResponse
}

func (s *stubProvider) ID() string { return "stub" }
func (s *stubProvider) Complete(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResponse, error) {
	s.response.Usage.InputTokens = 1
	s.response.Usage.OutputTokens = 1
	return &s.response, nil
}

func TestCompleteDecompositionLogsAudit(t *testing.T) {
	tempDir := t.TempDir()
	repo := storage.NewFilesystemRepository(tempDir)
	_ = repo.Initialize()
	provider := &stubProvider{
		response: ai.CompletionResponse{
			Text:  `[{"id":"task-1","title":"Task 1","feature_id":"feat"}]`,
			Model: "stub-model",
		},
	}
	audit := NewAuditService(repo)
	planSvc := NewPlanService(repo, audit)
	service := NewAIPlanningService(repo, provider, audit, planSvc)

	ctx := context.Background()
	resp, err := service.completeDecomposition(ctx, "prompt", 1)
	if err != nil {
		t.Fatalf("completeDecomposition failed: %v", err)
	}
	if resp.Model != "stub-model" {
		t.Fatalf("expected stub-model, got %s", resp.Model)
	}

	events, err := repo.LoadEvents()
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Action != "plan.ai_decomposition" {
		t.Fatalf("unexpected action: %s", events[0].Action)
	}
}

func TestParseTasksFromResponseWrapper(t *testing.T) {
	service := &AIPlanningService{}
	tasks, err := service.parseTasksFromResponse(`{"tasks":[{"id":"foo","title":"Foo","feature_id":"feat"}]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != "foo" {
		t.Fatalf("unexpected tasks: %+v", tasks)
	}
}

func TestParseTasksFromResponseMap(t *testing.T) {
	service := &AIPlanningService{}
	tasks, err := service.parseTasksFromResponse(`{"foo":{"description":"desc","feature_id":"feat","title":"Foo"}}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 1 || tasks[0].FeatureID != "feat" {
		t.Fatalf("unexpected map result: %+v", tasks)
	}
}

func TestParseTasksFromResponseError(t *testing.T) {
	service := &AIPlanningService{}
	if _, err := service.parseTasksFromResponse(`{"error":"boom"}`); err == nil {
		t.Fatal("expected error for response error field")
	}
}

func TestHasAnyKeyMatchesAlternateNames(t *testing.T) {
	payload := map[string]interface{}{
		"task_id": "exists",
		"feature": "f",
	}
	if !hasAnyKey(payload, "task-id", "task_id", "taskId") {
		t.Fatal("expected hasAnyKey to find task_id")
	}
	if !hasAnyKey(payload, "feature-id", "feature") {
		t.Fatal("expected hasAnyKey to find feature")
	}
}

func TestNormalizeTaskMapGeneratesSlugFallback(t *testing.T) {
	payload := map[string]interface{}{
		"description": "A long description that should generate a slug when no id is provided",
		"feature":     "feat",
	}
	task := normalizeTaskMap(payload, 0)
	if task.ID == "" || task.Title == "" {
		t.Fatalf("expected generated id/title, got %+v", task)
	}
	if task.FeatureID != "feat" {
		t.Fatalf("expected feature retainment, got %s", task.FeatureID)
	}
}
