package cli

import (
	"os"
	"testing"
)

func TestExecute(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-root-test-*")
	defer os.RemoveAll(tempDir)
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(tempDir)

	// Help
	os.Args = []string{"roady", "--help"}
	if err := Execute(); err != nil {
		t.Errorf("Execute failed: %v", err)
	}
}
