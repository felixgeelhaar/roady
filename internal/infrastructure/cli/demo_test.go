package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDemoCmd_ScaffoldsAndDetectsDrift(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sample")

	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	t.Cleanup(func() {
		RootCmd.SetOut(nil)
		RootCmd.SetErr(nil)
		RootCmd.SetArgs(nil)
	})
	RootCmd.SetArgs([]string{"demo", target})

	err := RootCmd.Execute()
	if err != nil {
		t.Fatalf("demo failed: %v\noutput:\n%s", err, buf.String())
	}

	for _, name := range []string{"spec.yaml", "spec.lock.json", "plan.json", "state.json"} {
		path := filepath.Join(target, ".roady", name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
		}
	}

	out := buf.String()
	for _, want := range []string{
		"Scaffolding Roady demo",
		"drift detect",
		"Try next:",
		"roady drift accept",
	} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Errorf("expected output to contain %q\noutput:\n%s", want, out)
		}
	}
}

func TestDemoCmd_RefusesExistingProject(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sample")
	if err := os.MkdirAll(filepath.Join(target, ".roady"), 0o755); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	t.Cleanup(func() {
		RootCmd.SetOut(nil)
		RootCmd.SetErr(nil)
		RootCmd.SetArgs(nil)
	})
	RootCmd.SetArgs([]string{"demo", target})

	err := RootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when .roady already exists")
	}
}

func TestDemoCmd_ForceFlagOverwrites(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sample")
	if err := os.MkdirAll(filepath.Join(target, ".roady"), 0o755); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	t.Cleanup(func() {
		RootCmd.SetOut(nil)
		RootCmd.SetErr(nil)
		RootCmd.SetArgs(nil)
	})
	RootCmd.SetArgs([]string{"demo", target, "--force"})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("demo --force failed: %v\noutput:\n%s", err, buf.String())
	}
}
