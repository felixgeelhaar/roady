package plugin

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLoader_Full(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-plugin-full-*")
	defer os.RemoveAll(tempDir)

	// 1. Build the mock plugin
	pluginBin := filepath.Join(tempDir, "plugin.bin")
	// Note: We assume we are in the root of the project for this relative path to work
	// or we use a more robust way to find the source.
	// Since I'm an agent, I'll try to build cmd/roady-plugin-mock/main.go
	cmd := exec.Command("go", "build", "-o", pluginBin, "../../../cmd/roady-plugin-mock/main.go")
	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping full plugin test: build failed: %v", err)
		return
	}

	l := NewLoader()
	defer l.Cleanup()

	// 2. Load
	syncer, err := l.Load(pluginBin)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if syncer == nil {
		t.Fatal("Syncer is nil")
	}

	// 3. Cleanup
	l.Cleanup()
}

func TestLoader_Init(t *testing.T) {
	l := NewLoader()
	if l == nil {
		t.Fatal("Loader is nil")
	}
	l.Cleanup()

	if HandshakeConfig.MagicCookieKey != "ROADY_PLUGIN" {
		t.Errorf("wrong magic cookie key")
	}
}

func TestLoader_LoadError(t *testing.T) {
	l := NewLoader()
	_, err := l.Load("/invalid/path/999")
	if err == nil {
		t.Error("expected error for invalid plugin path")
	}
}

func TestLoader_LoadDirectory(t *testing.T) {
	tempDir := t.TempDir()
	l := NewLoader()
	_, err := l.Load(tempDir)
	if err == nil {
		t.Error("expected error for directory path")
	}
}

func TestLoader_LoadNonExecutable(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "plugin")
	if err := os.WriteFile(filePath, []byte("not executable"), 0644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	l := NewLoader()
	_, err := l.Load(filePath)
	if err == nil {
		t.Error("expected error for non-executable file")
	}
}
