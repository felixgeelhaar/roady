package application_test

import (
	"os"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestUsageService_IncrementCommand(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-usage-*")
	defer func() { _ = os.RemoveAll(tempDir) }()

	repo := storage.NewFilesystemRepository(tempDir)
	_ = repo.Initialize()
	svc := application.NewUsageService(repo)

	// Increment twice
	if err := svc.IncrementCommand(); err != nil {
		t.Fatalf("increment: %v", err)
	}
	if err := svc.IncrementCommand(); err != nil {
		t.Fatalf("increment 2: %v", err)
	}

	stats, err := svc.GetUsage()
	if err != nil {
		t.Fatalf("get usage: %v", err)
	}
	if stats.TotalCommands != 2 {
		t.Errorf("expected 2 commands, got %d", stats.TotalCommands)
	}
}

func TestUsageService_RecordTokenUsage(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-usage-tokens-*")
	defer func() { _ = os.RemoveAll(tempDir) }()

	repo := storage.NewFilesystemRepository(tempDir)
	_ = repo.Initialize()
	svc := application.NewUsageService(repo)

	// Record tokens for gpt-4
	if err := svc.RecordTokenUsage("gpt-4", 100, 50); err != nil {
		t.Fatalf("record tokens: %v", err)
	}
	if err := svc.RecordTokenUsage("gpt-4", 200, 100); err != nil {
		t.Fatalf("record tokens 2: %v", err)
	}

	stats, err := svc.GetUsage()
	if err != nil {
		t.Fatalf("get usage: %v", err)
	}

	inputTotal := stats.ProviderStats["gpt-4:input"]
	outputTotal := stats.ProviderStats["gpt-4:output"]

	if inputTotal != 300 {
		t.Errorf("expected 300 input tokens, got %d", inputTotal)
	}
	if outputTotal != 150 {
		t.Errorf("expected 150 output tokens, got %d", outputTotal)
	}
}

func TestUsageService_GetTotalTokens(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-usage-total-*")
	defer func() { _ = os.RemoveAll(tempDir) }()

	repo := storage.NewFilesystemRepository(tempDir)
	_ = repo.Initialize()
	svc := application.NewUsageService(repo)

	// Record tokens across providers
	_ = svc.RecordTokenUsage("gpt-4", 100, 50)
	_ = svc.RecordTokenUsage("claude", 200, 75)

	total, err := svc.GetTotalTokens()
	if err != nil {
		t.Fatalf("get total: %v", err)
	}
	// 100 + 50 + 200 + 75 = 425
	if total != 425 {
		t.Errorf("expected 425 total tokens, got %d", total)
	}
}

func TestUsageService_EmptyRepo(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-usage-empty-*")
	defer func() { _ = os.RemoveAll(tempDir) }()

	repo := storage.NewFilesystemRepository(tempDir)
	_ = repo.Initialize()
	svc := application.NewUsageService(repo)

	total, err := svc.GetTotalTokens()
	if err != nil {
		t.Fatalf("get total: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 total tokens for empty repo, got %d", total)
	}
}
