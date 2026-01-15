package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHappyPath(t *testing.T) {
	// Setup
	distDir, _ := filepath.Abs("../../dist")
	roadyBin := filepath.Join(distDir, "roady")

	tempDir, err := os.MkdirTemp("", "roady-e2e-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Helper to run roady
	runRoady := func(args ...string) string {
		cmd := exec.Command(roadyBin, args...)
		cmd.Dir = tempDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("roady %v failed: %v\nOutput: %s", args, err, output)
		}
		return string(output)
	}

	// Helper that allows failure (for drift detect)
	runRoadyAllowFail := func(args ...string) string {
		cmd := exec.Command(roadyBin, args...)
		cmd.Dir = tempDir
		output, _ := cmd.CombinedOutput()
		return string(output)
	}

	// 1. Init
	t.Log("Running roady init...")
	out := runRoady("init")
	if !strings.Contains(out, "Successfully initialized roady project") {
		t.Errorf("Unexpected init output: %s", out)
	}

	// Verify .roady structure
	if _, err := os.Stat(filepath.Join(tempDir, ".roady", "spec.yaml")); os.IsNotExist(err) {
		t.Error(".roady/spec.yaml missing")
	}

	// 2. Spec (Add Feature)
	specPath := filepath.Join(tempDir, ".roady", "spec.yaml")
	specContent := `
id: "e2e-project"
title: "E2E Project"
features:
  - id: "feat-1"
    title: "Login"
    description: "User login flow"
    requirements:
      - id: "req-1"
        title: "Login Page"
        description: "HTML page for login"
`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Plan Generate
	t.Log("Running roady plan generate...")
	runRoady("plan", "generate")

	planPath := filepath.Join(tempDir, ".roady", "plan.json")
	planData, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal("plan.json missing")
	}
	if !strings.Contains(string(planData), "feat-1") {
		t.Error("Plan missing feature data")
	}

	// 3b. Accept Intent Drift (Bless the new Spec)
	// This should lock the spec so subsequent checks don't complain about spec changes
	t.Log("Running roady drift accept...")
	runRoady("drift", "accept")

	// 4. Status
	t.Log("Running roady status...")
	out = runRoady("status")
	if !strings.Contains(out, "E2E Project") {
		t.Errorf("Status output missing project title: %s", out)
	}

	// 5. Drift (Expect Code Drift/Missing)
	// Create dummy state to simulate task completion without code
	statePath := filepath.Join(tempDir, ".roady", "state.json")
	stateContent := `
{
  "project_id": "e2e-project",
  "task_states": {
    "task-req-1": {
      "status": "done",
      "path": "login.html"
    }
  },
  "updated_at": "2024-01-01T00:00:00Z"
}`
	if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Log("Running roady drift detect (expecting failure)...")
	out = runRoadyAllowFail("drift", "detect")

	// We expect "missing-code-task-req-1" because login.html doesn't exist
	if !strings.Contains(out, "missing-code-task-req-1") && !strings.Contains(out, "path 'login.html' is missing") {
		t.Errorf("Expected code drift reporting missing file. Output: %s", out)
	}

	// 6. Fix Drift
	os.WriteFile(filepath.Join(tempDir, "login.html"), []byte("<html></html>"), 0644)

	t.Log("Running roady drift detect (after fix)...")
	runRoady("drift", "detect") // Helper will fail if exit code != 0
}
