package contract

import (
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
