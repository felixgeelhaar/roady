package application_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain/plugin"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestPluginService_RegisterAndList(t *testing.T) {
	root := t.TempDir()
	roadyDir := filepath.Join(root, ".roady")
	_ = os.MkdirAll(roadyDir, 0700)

	// Create a fake binary
	binPath := filepath.Join(root, "fake-plugin")
	_ = os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0755)

	repo := storage.NewFilesystemRepository(root)
	svc := application.NewPluginService(repo)

	if err := svc.RegisterPlugin("test-plugin", binPath); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	plugins, err := svc.ListPlugins()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	if plugins[0].Name != "test-plugin" {
		t.Errorf("expected name 'test-plugin', got %q", plugins[0].Name)
	}
	if plugins[0].Status != "available" {
		t.Errorf("expected status 'available', got %q", plugins[0].Status)
	}
}

func TestPluginService_Unregister(t *testing.T) {
	root := t.TempDir()
	roadyDir := filepath.Join(root, ".roady")
	_ = os.MkdirAll(roadyDir, 0700)

	binPath := filepath.Join(root, "fake-plugin")
	_ = os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0755)

	repo := storage.NewFilesystemRepository(root)
	svc := application.NewPluginService(repo)

	if err := svc.RegisterPlugin("test-plugin", binPath); err != nil {
		t.Fatal(err)
	}

	if err := svc.UnregisterPlugin("test-plugin"); err != nil {
		t.Fatalf("unregister failed: %v", err)
	}

	plugins, _ := svc.ListPlugins()
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins after unregister, got %d", len(plugins))
	}
}

func TestPluginService_UnregisterNotFound(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, ".roady"), 0700)

	repo := storage.NewFilesystemRepository(root)
	svc := application.NewPluginService(repo)

	err := svc.UnregisterPlugin("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plugin")
	}
}

func TestPluginService_Validate(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, ".roady"), 0700)

	binPath := filepath.Join(root, "fake-plugin")
	_ = os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0755)

	repo := storage.NewFilesystemRepository(root)
	svc := application.NewPluginService(repo)
	if err := svc.RegisterPlugin("test-plugin", binPath); err != nil {
		t.Fatal(err)
	}

	result, err := svc.ValidatePlugin("test-plugin")
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("expected valid=true, got false: %s", result.Error)
	}
}

func TestPluginService_CheckHealth(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, ".roady"), 0700)

	binPath := filepath.Join(root, "fake-plugin")
	_ = os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0755)

	repo := storage.NewFilesystemRepository(root)
	svc := application.NewPluginService(repo)
	if err := svc.RegisterPlugin("test-plugin", binPath); err != nil {
		t.Fatal(err)
	}

	result, err := svc.CheckHealth("test-plugin")
	if err != nil {
		t.Fatalf("check health failed: %v", err)
	}

	if result.Status != application.HealthStatusHealthy {
		t.Errorf("expected healthy, got %s: %s", result.Status, result.Error)
	}
}

func TestPluginService_CheckHealthMissing(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, ".roady"), 0700)

	repo := storage.NewFilesystemRepository(root)
	svc := application.NewPluginService(repo)

	// Register with nonexistent binary
	if err := repo.SetPluginConfig("missing", plugin.PluginConfig{Binary: "/nonexistent", Config: map[string]string{}}); err != nil {
		t.Fatal(err)
	}

	result, err := svc.CheckHealth("missing")
	if err != nil {
		t.Fatalf("check health failed: %v", err)
	}

	if result.Status != application.HealthStatusUnhealthy {
		t.Errorf("expected unhealthy, got %s", result.Status)
	}
}

func TestPluginService_CheckAllHealth(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, ".roady"), 0700)

	binPath := filepath.Join(root, "fake-plugin")
	_ = os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0755)

	repo := storage.NewFilesystemRepository(root)
	svc := application.NewPluginService(repo)
	_ = svc.RegisterPlugin("test-plugin", binPath)

	results, err := svc.CheckAllHealth()
	if err != nil {
		t.Fatalf("check all health failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}
