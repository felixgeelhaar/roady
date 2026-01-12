package main

import (
	"log"

	"github.com/felixgeelhaar/roady/internal/domain/planning"
	domainPlugin "github.com/felixgeelhaar/roady/internal/domain/plugin"
	infraPlugin "github.com/felixgeelhaar/roady/internal/infrastructure/plugin"
	"github.com/hashicorp/go-plugin"
)

type MockSyncer struct{}

func (m *MockSyncer) Init(config map[string]string) error {
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

		if status == planning.StatusInProgress {
			log.Printf("Simulating external completion of task: %s", t.ID)
			updates[t.ID] = planning.StatusDone
		} else if status == planning.StatusPending {
			log.Printf("Simulating external start of task: %s", t.ID)
			updates[t.ID] = planning.StatusInProgress
		}
	}

	return &domainPlugin.SyncResult{StatusUpdates: updates}, nil
}
func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: infraPlugin.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"syncer": &domainPlugin.SyncerPlugin{Impl: &MockSyncer{}},
		},
	})
}
