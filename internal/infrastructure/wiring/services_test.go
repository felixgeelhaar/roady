package wiring

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/roady/internal/infrastructure/config"
	domainai "github.com/felixgeelhaar/roady/pkg/domain/ai"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
)

func TestBuildAppServicesDefaults(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, ".roady"), 0700); err != nil {
		t.Fatalf("mkdir roady: %v", err)
	}

	services, err := BuildAppServices(tempDir)
	if err != nil {
		t.Fatalf("build services failed: %v", err)
	}
	if services.Workspace == nil || services.Init == nil || services.Plan == nil || services.AI == nil {
		t.Fatalf("expected non-nil services, got %+v", services)
	}
	if services.Provider.ID() != "ollama:llama3" {
		t.Fatalf("expected default provider id, got %s", services.Provider.ID())
	}
}

func TestBuildAppServicesFallbackOnInvalidProvider(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, ".roady"), 0700); err != nil {
		t.Fatalf("mkdir roady: %v", err)
	}

	cfg := &config.AIConfig{Provider: "unknown", Model: "nope"}
	if err := config.SaveAIConfig(tempDir, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	services, err := BuildAppServices(tempDir)
	if err == nil {
		t.Fatalf("expected error when provider is invalid")
	}
	if services == nil {
		t.Fatal("expected services even when fallback error occurs")
	}
	if services.Provider.ID() != "ollama:llama3" {
		t.Fatalf("expected fallback provider id, got %s", services.Provider.ID())
	}
}

type stubProvider struct{}

func (stubProvider) ID() string { return "stub:provider" }
func (stubProvider) Complete(_ context.Context, _ domainai.CompletionRequest) (*domainai.CompletionResponse, error) {
	return &domainai.CompletionResponse{Model: "stub"}, nil
}

func TestBuildAppServicesWithCustomResolver(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, ".roady"), 0700); err != nil {
		t.Fatalf("mkdir roady: %v", err)
	}

	resolver := func(root string) (domainai.Provider, error) {
		return stubProvider{}, nil
	}

	services, err := BuildAppServicesWithProvider(tempDir, resolver)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if services.Provider.ID() != "stub:provider" {
		t.Fatalf("expected stub provider, got %s", services.Provider.ID())
	}
}

func TestBuildAppServicesWithResolverError(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, ".roady"), 0700); err != nil {
		t.Fatalf("mkdir roady: %v", err)
	}

	resolver := func(root string) (domainai.Provider, error) {
		return nil, errors.New("boom")
	}

	services, err := BuildAppServicesWithProvider(tempDir, resolver)
	if err == nil {
		t.Fatal("expected error when resolver fails")
	}
	if services == nil {
		t.Fatal("expected services even when resolver fails")
	}
	if services.Provider.ID() != "ollama:llama3" {
		t.Fatalf("expected fallback provider, got %s", services.Provider.ID())
	}
}

func TestBuildAppServices_PlanEvents(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, ".roady"), 0700); err != nil {
		t.Fatalf("mkdir roady: %v", err)
	}

	services, err := BuildAppServices(tempDir)
	if err != nil {
		t.Fatalf("build services: %v", err)
	}

	spec := &spec.ProductSpec{
		ID:    "wire",
		Title: "Wire",
		Features: []spec.Feature{
			{ID: "feature-wire", Title: "Wire Feature"},
		},
	}
	if err := services.Workspace.Repo.SaveSpec(spec); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	if _, err := services.Plan.GeneratePlan(context.Background()); err != nil {
		t.Fatalf("generate plan failed: %v", err)
	}
	if err := services.Plan.ApprovePlan(); err != nil {
		t.Fatalf("approve plan failed: %v", err)
	}

	events, err := services.Workspace.Repo.LoadEvents()
	if err != nil {
		t.Fatalf("load events: %v", err)
	}

	found := map[string]bool{}
	for _, ev := range events {
		found[ev.Action] = true
	}

	for _, want := range []string{"plan.generate", "plan.approved"} {
		if !found[want] {
			t.Fatalf("expected %s event via wiring, got %v", want, events)
		}
	}
}
