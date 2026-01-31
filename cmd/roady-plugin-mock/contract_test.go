package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/plugin/contract"
)

func TestMockPluginContract(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping contract test in short mode")
	}

	// Build the mock plugin binary
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "roady-plugin-mock")

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = "."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build mock plugin: %v", err)
	}

	// Run contract suite
	suite := contract.NewContractSuite()
	result, err := suite.RunBinary(binPath)
	if err != nil {
		t.Fatalf("contract suite failed to run: %v", err)
	}

	for _, r := range result.Results {
		t.Logf("[%s] passed=%v: %s", r.Name, r.Passed, r.Message)
	}

	if result.Failed > 0 {
		t.Errorf("contract suite: %d passed, %d failed", result.Passed, result.Failed)
	}
}
