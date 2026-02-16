package mcp

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/internal/infrastructure/config"
	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestServerHandlersExercise(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := config.SaveAIConfig(root, &config.AIConfig{Provider: "mock", Model: "coverage-test"}); err != nil {
		t.Fatalf("save ai config: %v", err)
	}

	specFile := &spec.ProductSpec{
		ID:    "project",
		Title: "Project",
		Features: []spec.Feature{
			{
				ID:    "feature",
				Title: "Feature",
				Requirements: []spec.Requirement{
					{ID: "req-alpha", Title: "Alpha", Description: "Desc"},
					{ID: "req-beta", Title: "Beta", Description: "Desc"},
				},
			},
		},
	}
	if err := repo.SaveSpec(specFile); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	plan := &planning.Plan{
		ID:             "plan-1",
		SpecID:         specFile.ID,
		ApprovalStatus: planning.ApprovalPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Tasks: []planning.Task{
			{ID: "task-req-alpha", Title: "Alpha Task", FeatureID: "feature"},
		},
	}
	if err := repo.SavePlan(plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	state := planning.NewExecutionState(plan.ID)
	state.TaskStates["task-req-alpha"] = planning.TaskResult{Status: planning.StatusPending}
	if err := repo.SaveState(state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 3, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	if err := repo.UpdateUsage(domain.UsageStats{ProviderStats: map[string]int{}}); err != nil {
		t.Fatalf("update usage: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o700); err != nil {
		t.Fatalf("create docs dir: %v", err)
	}

	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(original)
	})
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// Initialize dummy git repo
	exec.Command("git", "init").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "Test").Run()
	exec.Command("git", "config", "commit.gpgsign", "false").Run()
	os.WriteFile(filepath.Join(root, "README.md"), []byte("test"), 0644)
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "Initial commit").Run()

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	ctx := context.Background()

	if _, err := server.handleGetSpec(ctx, struct{}{}); err != nil {
		t.Fatalf("get spec: %v", err)
	}
	if _, err := server.handleGetPlan(ctx, struct{}{}); err != nil {
		t.Fatalf("get plan: %v", err)
	}
	if _, err := server.handleGetState(ctx, struct{}{}); err != nil {
		t.Fatalf("get state: %v", err)
	}

	if _, err := server.handleGeneratePlan(ctx, struct{}{}); err != nil {
		t.Fatalf("generate plan: %v", err)
	}

	if _, err := server.handleUpdatePlan(ctx, UpdatePlanArgs{
		Tasks: []planning.Task{
			{ID: "task-req-alpha", Title: "Alpha Task", FeatureID: "feature"},
		},
	}); err != nil {
		t.Fatalf("update plan: %v", err)
	}

	if _, err := server.handleDetectDrift(ctx, struct{}{}); err != nil {
		t.Fatalf("detect drift: %v", err)
	}

	if _, err := server.handleStatus(ctx, StatusArgs{}); err != nil {
		t.Fatalf("status: %v", err)
	}

	if _, err := server.handleCheckPolicy(ctx, struct{}{}); err != nil {
		t.Fatalf("check policy: %v", err)
	}

	if _, err := server.handleGetUsage(ctx, struct{}{}); err != nil {
		t.Fatalf("get usage: %v", err)
	}

	if _, err := server.handleExplainSpec(ctx, struct{}{}); err != nil {
		t.Fatalf("explain spec: %v", err)
	}

	if _, err := server.handleExplainDrift(ctx, struct{}{}); err != nil {
		t.Fatalf("explain drift: %v", err)
	}

	if _, err := server.handleAcceptDrift(ctx, struct{}{}); err != nil {
		t.Fatalf("accept drift: %v", err)
	}

	if _, err := server.handleForecast(ctx, struct{}{}); err != nil {
		t.Fatalf("forecast: %v", err)
	}

	if _, err := server.handleOrgStatus(ctx, struct{}{}); err != nil {
		t.Fatalf("org status: %v", err)
	}

	if _, err := server.handleGitSync(ctx, struct{}{}); err != nil {
		t.Fatalf("git sync: %v", err)
	}

	if _, err := server.handleApprovePlan(ctx, struct{}{}); err != nil {
		t.Fatalf("approve plan: %v", err)
	}

	if _, err := server.handleTransitionTask(ctx, TransitionTaskArgs{
		TaskID:   "task-req-alpha",
		Event:    "start",
		Evidence: "coverage",
	}); err != nil {
		t.Fatalf("transition task: %v", err)
	}

	if _, err := server.handleAddFeature(ctx, AddFeatureArgs{
		Title:       "extra",
		Description: "details",
	}); err != nil {
		t.Fatalf("add feature: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(root, "docs", "backlog.md"))
	if err != nil {
		t.Fatalf("read backlog: %v", err)
	}
	if !strings.Contains(string(content), "extra") {
		t.Fatalf("backlog missing feature: %s", content)
	}

	if _, err := server.handleDetectDrift(ctx, struct{}{}); err != nil {
		t.Fatalf("detect drift after feature: %v", err)
	}

	if _, err := server.handleStatus(ctx, StatusArgs{}); err != nil {
		t.Fatalf("status after transition: %v", err)
	}

	if _, err := server.handleExplainSpec(ctx, struct{}{}); err != nil {
		t.Fatalf("explain spec post-change: %v", err)
	}
}

func TestServerPlanEventsLogged(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	specFile := &spec.ProductSpec{
		ID:    "event-spec",
		Title: "Eventful Project",
		Features: []spec.Feature{
			{ID: "feature-a", Title: "Feature A"},
		},
	}
	if err := repo.SaveSpec(specFile); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	ctx := context.Background()

	if _, err := server.handleGeneratePlan(ctx, struct{}{}); err != nil {
		t.Fatalf("generate plan failed: %v", err)
	}
	if _, err := server.handleApprovePlan(ctx, struct{}{}); err != nil {
		t.Fatalf("approve plan failed: %v", err)
	}

	events, err := repo.LoadEvents()
	if err != nil {
		t.Fatalf("load events: %v", err)
	}

	found := map[string]bool{}
	for _, ev := range events {
		found[ev.Action] = true
	}

	for _, want := range []string{"plan.generate", "plan.approved"} {
		if !found[want] {
			t.Fatalf("expected event %s, got %v", want, events)
		}
	}
}

func TestInitHandlerCreatesProject(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	if _, err := server.handleInit(context.Background(), InitArgs{Name: "demo"}); err != nil {
		t.Fatalf("init handler failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".roady", "spec.yaml")); err != nil {
		t.Fatalf("spec was not created: %v", err)
	}
}

func TestGRPCServerStartsAndStops(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	specFile := &spec.ProductSpec{
		ID:    "grpc-test",
		Title: "gRPC Test Project",
	}
	if err := repo.SaveSpec(specFile); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		errCh <- server.ServeGRPC(ctx, ":0") // Use :0 for random available port
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}
