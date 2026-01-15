package cli

import (
	"os"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestLoadServicesSucceeds(t *testing.T) {
	tempDir := t.TempDir()
	repo := storage.NewFilesystemRepository(tempDir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	services, err := loadServices(tempDir)
	if err != nil {
		t.Fatalf("load services: %v", err)
	}
	if services == nil || services.Plan == nil || services.AI == nil {
		t.Fatalf("expected services, got %+v", services)
	}
}

func TestLoadServicesForCurrentDir(t *testing.T) {
	tempDir := t.TempDir()
	repo := storage.NewFilesystemRepository(tempDir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	defer func() {
		_ = os.Chdir(original)
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	services, err := loadServicesForCurrentDir()
	if err != nil {
		t.Fatalf("load services for current dir: %v", err)
	}
	if services == nil || services.Spec == nil {
		t.Fatalf("expected services, got %+v", services)
	}
}
