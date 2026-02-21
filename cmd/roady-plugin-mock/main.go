package main

import (
	"fmt"
	"log"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	domainPlugin "github.com/felixgeelhaar/roady/pkg/domain/plugin"
	infraPlugin "github.com/felixgeelhaar/roady/pkg/plugin"
	goplugin "github.com/hashicorp/go-plugin"
)

type MockSyncer struct{}

func (m *MockSyncer) Init(config map[string]string) error {
	if config["fail"] == "true" {
		return fmt.Errorf("mock init failure (fail=true)")
	}
	return nil
}

func (m *MockSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (*domainPlugin.SyncResult, error) {
	log.Printf("Received plan with %d tasks", len(plan.Tasks))

	// Simulate updates: Mark all "in_progress" tasks as "done", and "pending" as "in_progress"
	updates := make(map[string]planning.TaskStatus)
	for _, t := range plan.Tasks {
		status := planning.StatusPending
		if res, ok := state.TaskStates[t.ID]; ok {
			status = res.Status
		}

		switch status {
		case planning.StatusInProgress:
			log.Printf("Simulating external completion of task: %s", t.ID)
			updates[t.ID] = planning.StatusDone
		case planning.StatusPending:
			log.Printf("Simulating external start of task: %s", t.ID)
			updates[t.ID] = planning.StatusInProgress
		}
	}

	return &domainPlugin.SyncResult{StatusUpdates: updates}, nil
}

func (m *MockSyncer) Push(taskID string, status planning.TaskStatus) error {
	log.Printf("Mock push: task %s -> status %s", taskID, status)
	return nil
}
func main() {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: infraPlugin.HandshakeConfig,
		Plugins: map[string]goplugin.Plugin{
			"syncer": &domainPlugin.SyncerPlugin{Impl: &MockSyncer{}},
		},
	})
}
