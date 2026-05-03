package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmptyStateHintForRoot(t *testing.T) {
	t.Run("no_roady_dir", func(t *testing.T) {
		dir := t.TempDir()
		step, ok := emptyStateHintForRoot(dir)
		if !ok {
			t.Fatal("expected hint for empty dir")
		}
		if step.Stage != "uninitialised" {
			t.Errorf("stage = %q, want uninitialised", step.Stage)
		}
		if step.Command != "roady init" {
			t.Errorf("command = %q, want 'roady init'", step.Command)
		}
	})

	t.Run("roady_dir_no_spec", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, ".roady"), 0o755); err != nil {
			t.Fatal(err)
		}
		step, ok := emptyStateHintForRoot(dir)
		if !ok {
			t.Fatal("expected hint when spec missing")
		}
		if step.Stage != "no-spec" {
			t.Errorf("stage = %q, want no-spec", step.Stage)
		}
	})

	t.Run("spec_no_plan", func(t *testing.T) {
		dir := t.TempDir()
		roady := filepath.Join(dir, ".roady")
		if err := os.MkdirAll(roady, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(roady, "spec.yaml"), []byte("id: x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		step, ok := emptyStateHintForRoot(dir)
		if !ok {
			t.Fatal("expected hint when plan missing")
		}
		if step.Stage != "no-plan" {
			t.Errorf("stage = %q, want no-plan", step.Stage)
		}
	})

	t.Run("spec_and_plan_present_returns_no_hint", func(t *testing.T) {
		dir := t.TempDir()
		roady := filepath.Join(dir, ".roady")
		if err := os.MkdirAll(roady, 0o755); err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(roady, "spec.yaml"), []byte("id: x\n"), 0o644)
		_ = os.WriteFile(filepath.Join(roady, "plan.json"), []byte("{}"), 0o644)
		if _, ok := emptyStateHintForRoot(dir); ok {
			t.Error("expected no hint when both spec and plan exist")
		}
	})
}
