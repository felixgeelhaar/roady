package cli

import (
	"os"
	"testing"
)

func TestExecute(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-root-test-*")
	defer func() { _ = os.RemoveAll(tempDir) }()
	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	_ = os.Chdir(tempDir)

	// Help
	os.Args = []string{"roady", "--help"}
	if err := Execute(); err != nil {
		t.Errorf("Execute failed: %v", err)
	}
}
