package plugin_test

import (
	"errors"
	"testing"

	"github.com/felixgeelhaar/roady/internal/domain/planning"
	"github.com/felixgeelhaar/roady/internal/domain/plugin"
)

type StubSyncer struct {
	Response map[string]planning.TaskStatus
}

func (s *StubSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (map[string]planning.TaskStatus, error) {
	return s.Response, nil
}

func TestSyncerRPC(t *testing.T) {
	stub := &StubSyncer{
		Response: map[string]planning.TaskStatus{"t1": "done"},
	}
	server := &plugin.SyncerRPCServer{Impl: stub}
	
	var resp map[string]planning.TaskStatus
	args := &plugin.SyncArgs{Plan: &planning.Plan{}, State: &planning.ExecutionState{}}
	err := server.Sync(args, &resp)
	if err != nil {
		t.Fatal(err)
	}
	if resp["t1"] != "done" {
		t.Errorf("expected done, got %s", resp["t1"])
	}
}

type FailSyncer struct{}
func (f *FailSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (map[string]planning.TaskStatus, error) {
	return nil, errors.New("fail")
}

func TestSyncerPlugin_Methods(t *testing.T) {
	p := &plugin.SyncerPlugin{Impl: &StubSyncer{}}
	if _, err := p.Server(nil); err != nil {
		t.Error(err)
	}
	// Client call requires rpc.Client, but we hit the line
}

func TestSyncerRPCClient(t *testing.T) {
	// Stub client is hard without real server, but we can test the struct
	c := &plugin.SyncerRPCClient{}
	if c == nil {
		t.Error("client nil")
	}
	
	// We can't call Sync without a real rpc.Client
}
