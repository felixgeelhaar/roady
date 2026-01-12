package application_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/internal/application"
	"github.com/felixgeelhaar/roady/internal/infrastructure/storage"
)

func TestAuditService_Log(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-audit-test-*")
	defer os.RemoveAll(tempDir)

	repo := storage.NewFilesystemRepository(tempDir)
	repo.Initialize()
	service := application.NewAuditService(repo)

	// 1. Log Event
	if err := service.Log("test.action", "tester", map[string]interface{}{"key": "val"}); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// 2. Verify File
	content, err := os.ReadFile(filepath.Join(tempDir, ".roady", "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "test.action") {
		t.Error("Event not logged")
	}

	// 3. Verify Usage
	usageContent, err := os.ReadFile(filepath.Join(tempDir, ".roady", "usage.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(usageContent), "total_commands") {
		t.Error("Usage stats not updated")
	}
}

func TestAuditService_Error(t *testing.T) {
	repo := &MockRepo{SaveError: errors.New("audit fail")}
	service := application.NewAuditService(repo)

	if err := service.Log("act", "actor", nil); err == nil {
		t.Error("expected error on save fail")
	}
}
