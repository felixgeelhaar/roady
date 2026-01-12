package cli

import (
	"bytes"
	"os"
	"testing"
)

func TestInitCmd_Internal(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-cli-internal-*")
	defer os.RemoveAll(tempDir)
	
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(tempDir)

	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)

	RootCmd.SetArgs([]string{"init", "test"})
	if err := RootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Double init should fail
	if err := RootCmd.Execute(); err == nil {
		t.Error("expected error on re-init")
	}
}
