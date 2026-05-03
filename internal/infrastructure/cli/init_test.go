package cli

import (
	"bytes"
	"os"
	"testing"
)

func TestShouldRunInteractive(t *testing.T) {
	origTTY := stdinIsTTY
	t.Cleanup(func() {
		stdinIsTTY = origTTY
		initInteractive = false
		initNonInteractive = false
		initTemplate = ""
	})

	cases := []struct {
		name              string
		interactive       bool
		nonInteractive    bool
		template          string
		tty               bool
		want              bool
	}{
		{"explicit interactive wins over non-tty", true, false, "", false, true},
		{"explicit non-interactive even on tty", false, true, "", true, false},
		{"template skips wizard", false, false, "minimal", true, false},
		{"tty + no flags = wizard", false, false, "", true, true},
		{"piped (non-tty) + no flags = no wizard", false, false, "", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			initInteractive = tc.interactive
			initNonInteractive = tc.nonInteractive
			initTemplate = tc.template
			stdinIsTTY = func() bool { return tc.tty }

			if got := shouldRunInteractive(); got != tc.want {
				t.Errorf("shouldRunInteractive() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestInitCmd_Internal(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-cli-internal-*")
	defer func() { _ = os.RemoveAll(tempDir) }()

	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	_ = os.Chdir(tempDir)

	// Force non-interactive so the test is deterministic regardless of how
	// `go test` is invoked (TTY vs. piped stdin).
	origTTY := stdinIsTTY
	stdinIsTTY = func() bool { return false }
	t.Cleanup(func() {
		stdinIsTTY = origTTY
		initInteractive = false
		initNonInteractive = false
		initTemplate = ""
	})

	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	t.Cleanup(func() {
		RootCmd.SetOut(nil)
		RootCmd.SetErr(nil)
		RootCmd.SetArgs(nil)
	})

	RootCmd.SetArgs([]string{"init", "test"})
	if err := RootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Double init should fail
	if err := RootCmd.Execute(); err == nil {
		t.Error("expected error on re-init")
	}
}
