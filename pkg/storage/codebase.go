package storage

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CodebaseInspector implements CodeInspector using os and exec calls.
type CodebaseInspector struct{}

func NewCodebaseInspector() *CodebaseInspector {
	return &CodebaseInspector{}
}

func (i *CodebaseInspector) FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func (i *CodebaseInspector) FileNotEmpty(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.Size() > 0, nil
}

func (i *CodebaseInspector) GitStatus(path string) (string, error) {
	// check if git is installed/available
	if _, err := exec.LookPath("git"); err != nil {
		return "unknown", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain", path)
	var out bytes.Buffer
	cmd.Stdout = &out
	// ignore stderr, if git fails (e.g. not a repo), we return unknown/error
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "timeout", nil
		}
		return "error", nil
	}

	output := strings.TrimSpace(out.String())
	if output == "" {
		// If file exists but output is empty, it's tracked and clean.
		// BUT we must check if file exists first, otherwise git status --porcelain on non-existent file is also empty (if checked out) or specific error?
		// "git status --porcelain non_existent_file" returns nothing if it was never tracked.
		// If it was tracked and deleted, it returns "D".
		return "clean", nil
	}

	// Parse the first two chars
	// XY PATH
	// X = staging, Y = worktree
	// ?? = untracked
	// !! = ignored
	//  M = modified
	// M  = modified (staged)
	// D  = deleted

	if strings.HasPrefix(output, "??") {
		return "untracked", nil
	}
	if strings.HasPrefix(output, "!!") {
		return "ignored", nil
	}
	if strings.Contains(output, "D") { // Matches " D" or "D "
		return "missing", nil
	}
	if strings.Contains(output, "M") {
		return "modified", nil
	}

	return "unknown", nil
}
