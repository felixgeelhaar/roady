package cli

import (
	"bytes"
	"testing"
)

func TestCompletionCmd(t *testing.T) {
	shells := []string{"bash", "zsh", "fish", "powershell"}

	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			buf := new(bytes.Buffer)
			RootCmd.SetOut(buf)
			RootCmd.SetErr(buf)
			RootCmd.SetArgs([]string{"completion", shell})

			if err := RootCmd.Execute(); err != nil {
				t.Fatalf("completion %s failed: %v", shell, err)
			}

			if buf.Len() == 0 {
				t.Errorf("completion %s produced no output", shell)
			}
		})
	}
}
