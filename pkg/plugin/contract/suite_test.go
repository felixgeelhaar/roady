package contract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	domainPlugin "github.com/felixgeelhaar/roady/pkg/domain/plugin"
)

// fakeSyncer is a minimal in-process syncer for testing the suite runner.
type fakeSyncer struct{}

func (f *fakeSyncer) Init(config map[string]string) error {
	if config["fail"] == "true" {
		return &initError{}
	}
	return nil
}

type initError struct{}

func (e *initError) Error() string { return "init failed" }

func (f *fakeSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (*domainPlugin.SyncResult, error) {
	return &domainPlugin.SyncResult{StatusUpdates: map[string]planning.TaskStatus{}}, nil
}

func (f *fakeSyncer) Push(taskID string, status planning.TaskStatus) error {
	return nil
}

func TestContractSuite_RunWithSyncer(t *testing.T) {
	suite := NewContractSuite()
	result := suite.RunWithSyncer(&fakeSyncer{})

	if result.Passed+result.Failed != len(result.Results) {
		t.Errorf("passed(%d) + failed(%d) != total(%d)", result.Passed, result.Failed, len(result.Results))
	}

	// All assertions should pass against a well-behaved fake
	for _, r := range result.Results {
		if !r.Passed {
			t.Errorf("assertion %s failed: %s", r.Name, r.Message)
		}
	}
}

// failingSyncer always returns errors for Init and Sync, testing assertion failure paths.
type failingSyncer struct{}

func (f *failingSyncer) Init(config map[string]string) error {
	return &initError{}
}

func (f *failingSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (*domainPlugin.SyncResult, error) {
	return nil, &initError{}
}

func (f *failingSyncer) Push(taskID string, status planning.TaskStatus) error {
	return &initError{}
}

// nilResultSyncer returns nil from Sync to test nil-result branches.
type nilResultSyncer struct {
	fakeSyncer
}

func (n *nilResultSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (*domainPlugin.SyncResult, error) {
	return nil, nil
}

func TestAssertInitSuccess_Failure(t *testing.T) {
	r := AssertInitSuccess(&failingSyncer{})
	if r.Passed {
		t.Error("expected InitSuccess to fail with failingSyncer")
	}
	if r.Name != "InitSuccess" {
		t.Errorf("expected name 'InitSuccess', got %q", r.Name)
	}
}

func TestAssertInitWithBadConfig_NoError(t *testing.T) {
	// Use a syncer that never fails Init — the "bad config" assertion should report failure
	neverFail := &neverFailSyncer{}
	r := AssertInitWithBadConfig(neverFail)
	if r.Passed {
		t.Error("expected InitWithBadConfig to fail when syncer doesn't error")
	}
}

// neverFailSyncer accepts any config without error.
type neverFailSyncer struct {
	fakeSyncer
}

func (n *neverFailSyncer) Init(config map[string]string) error {
	return nil // never fails, even with fail=true
}

func TestAssertSyncEmptyPlan_Error(t *testing.T) {
	r := AssertSyncEmptyPlan(&failingSyncer{})
	if r.Passed {
		t.Error("expected SyncEmptyPlan to fail with failingSyncer")
	}
}

func TestAssertSyncEmptyPlan_NilResult(t *testing.T) {
	r := AssertSyncEmptyPlan(&nilResultSyncer{})
	if r.Passed {
		t.Error("expected SyncEmptyPlan to fail with nil result")
	}
}

func TestAssertSyncWithTasks_Error(t *testing.T) {
	r := AssertSyncWithTasks(&failingSyncer{})
	if r.Passed {
		t.Error("expected SyncWithTasks to fail with failingSyncer")
	}
}

func TestAssertSyncWithTasks_NilResult(t *testing.T) {
	r := AssertSyncWithTasks(&nilResultSyncer{})
	if r.Passed {
		t.Error("expected SyncWithTasks to fail with nil result")
	}
}

func TestAssertPushValidTask_Error(t *testing.T) {
	r := AssertPushValidTask(&failingSyncer{})
	if r.Passed {
		t.Error("expected PushValidTask to fail with failingSyncer")
	}
}

func TestContractSuite_RunWithFailingSyncer(t *testing.T) {
	suite := NewContractSuite()
	result := suite.RunWithSyncer(&failingSyncer{})

	if result.Passed+result.Failed != len(result.Results) {
		t.Errorf("passed(%d) + failed(%d) != total(%d)", result.Passed, result.Failed, len(result.Results))
	}

	// failingSyncer should cause some assertions to fail
	if result.Failed == 0 {
		t.Error("expected some failures with failingSyncer")
	}
}

func TestRunBinary_NotFound(t *testing.T) {
	suite := NewContractSuite()
	_, err := suite.RunBinary("/nonexistent/path/to/plugin")
	if err == nil {
		t.Error("expected error for nonexistent binary")
	}
}

func TestRunBinary_NotExecutable(t *testing.T) {
	// Create a temp file that is NOT executable
	dir := t.TempDir()
	path := filepath.Join(dir, "not-executable")
	if err := os.WriteFile(path, []byte("not a real binary"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	suite := NewContractSuite()
	_, err := suite.RunBinary(path)
	if err == nil {
		t.Error("expected error for non-executable file")
	}
}
