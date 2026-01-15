package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAllCmds_Internal(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-cli-all-*")
	defer os.RemoveAll(tempDir)
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(tempDir)

	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SilenceUsage = true
	RootCmd.SilenceErrors = true

	// 1. Init
	RootCmd.SetArgs([]string{"init", "test"})
	RootCmd.Execute()

	// 2. Plan
	RootCmd.SetArgs([]string{"plan", "generate"})
	RootCmd.Execute()

	// 3. Status
	RootCmd.SetArgs([]string{"status"})
	RootCmd.Execute()

	// Status empty
	os.Remove(filepath.Join(tempDir, ".roady", "plan.json"))
	RootCmd.SetArgs([]string{"status"})
	RootCmd.Execute()

	// 4. Drift
	RootCmd.SetArgs([]string{"drift", "detect"})
	RootCmd.Execute()

	// Drift detected
	os.WriteFile(filepath.Join(tempDir, ".roady", "spec.yaml"), []byte("id: test\ntitle: test\nfeatures: [{id: f1, title: F1}, {id: f2, title: F2}]"), 0600)
	os.WriteFile(filepath.Join(tempDir, ".roady", "plan.json"), []byte("{\"id\":\"p1\",\"tasks\":[{\"id\":\"t1\",\"feature_id\":\"f1\"}]}"), 0600)
	RootCmd.SetArgs([]string{"drift", "detect"})
	RootCmd.Execute()

	// Drift with JSON and issues
	os.WriteFile(filepath.Join(tempDir, ".roady", "spec.yaml"), []byte("id: test\ntitle: test\nfeatures:\n  - id: f1\n    title: F1"), 0600)
	RootCmd.SetArgs([]string{"drift", "detect", "--output", "json"})
	RootCmd.Execute()

	// 5. Policy
	RootCmd.SetArgs([]string{"policy", "check"})
	RootCmd.Execute()

	// 6. Spec
	RootCmd.SetArgs([]string{"spec", "validate"})
	RootCmd.Execute()

	// Spec Validate failure
	os.WriteFile(filepath.Join(tempDir, ".roady", "spec.yaml"), []byte("id: test\ntitle: test\nfeatures: [{id: f1}, {id: f1}]"), 0600)
	RootCmd.SetArgs([]string{"spec", "validate"})
	RootCmd.Execute()

	// 7. Task
	RootCmd.SetArgs([]string{"task", "start", "task-core-foundation"})
	RootCmd.Execute()

	// Task block
	RootCmd.SetArgs([]string{"task", "block", "task-core-foundation"})
	RootCmd.Execute()

	// Task unblock
	RootCmd.SetArgs([]string{"task", "unblock", "task-core-foundation"})
	RootCmd.Execute()

	// Task stop
	RootCmd.SetArgs([]string{"task", "stop", "task-core-foundation"})
	RootCmd.Execute()

	// Task complete
	RootCmd.SetArgs([]string{"task", "start", "task-core-foundation"})
	RootCmd.Execute()
	RootCmd.SetArgs([]string{"task", "complete", "task-core-foundation"})
	RootCmd.Execute()

	RootCmd.SetArgs([]string{"task", "block", "task-core-foundation"})
	RootCmd.Execute()

	RootCmd.SetArgs([]string{"task", "unblock", "task-core-foundation"})
	RootCmd.Execute()

	RootCmd.SetArgs([]string{"task", "stop", "task-core-foundation"})
	RootCmd.Execute()

	RootCmd.SetArgs([]string{"task", "start", "task-core-foundation"})
	RootCmd.Execute()

	RootCmd.SetArgs([]string{"task", "complete", "task-core-foundation"})
	RootCmd.Execute()

	// 8. Doctor
	RootCmd.SetArgs([]string{"doctor"})
	RootCmd.Execute()

	// 8.1 Dashboard
	os.Setenv("ROADY_SKIP_DASHBOARD_RUN", "true")
	defer os.Unsetenv("ROADY_SKIP_DASHBOARD_RUN")
	RootCmd.SetArgs([]string{"dashboard"})
	RootCmd.Execute()

	// 9. Spec Import
	md := filepath.Join(tempDir, "test.md")
	os.WriteFile(md, []byte("# Test\n## F1"), 0600)
	RootCmd.SetArgs([]string{"spec", "import", md})
	RootCmd.Execute()

	// Import non-markdown (still works because we just scan lines)
	txt := filepath.Join(tempDir, "test.txt")
	os.WriteFile(txt, []byte("plain text"), 0600)
	RootCmd.SetArgs([]string{"spec", "import", txt})
	RootCmd.Execute()

	// 10. Sync
	pluginBin := filepath.Join(tempDir, "roady-plugin-mock")
	exec.Command("go", "build", "-o", pluginBin, "../../../cmd/roady-plugin-mock/main.go").Run()
	RootCmd.SetArgs([]string{"sync", pluginBin})
	RootCmd.Execute()

	// 11. MCP (Internal)
	os.Setenv("ROADY_SKIP_MCP_START", "true")
	defer os.Unsetenv("ROADY_SKIP_MCP_START")
	RootCmd.SetArgs([]string{"mcp"})
	RootCmd.Execute()
}
