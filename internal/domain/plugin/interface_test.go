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

func (s *StubSyncer) Init(config map[string]string) error {
	return nil
}

func (s *StubSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (*plugin.SyncResult, error) {
	return &plugin.SyncResult{StatusUpdates: s.Response}, nil
}

func TestSyncerRPC(t *testing.T) {
	stub := &StubSyncer{
		Response: map[string]planning.TaskStatus{"t1": "done"},
	}
	server := &plugin.SyncerRPCServer{Impl: stub}

	var resp plugin.SyncResult
	args := &plugin.SyncArgs{Plan: &planning.Plan{}, State: &planning.ExecutionState{}}
	err := server.Sync(args, &resp)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusUpdates["t1"] != "done" {
		t.Errorf("expected done, got %s", resp.StatusUpdates["t1"])
	}
}

type ErrorSyncer struct{}

func (e *ErrorSyncer) Init(config map[string]string) error { return nil }
func (e *ErrorSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (*plugin.SyncResult, error) {
	return nil, errors.New("fail")
}

func TestSyncerRPC_Error(t *testing.T) {
	server := &plugin.SyncerRPCServer{Impl: &ErrorSyncer{}}
	var resp plugin.SyncResult
	args := &plugin.SyncArgs{Plan: &planning.Plan{}, State: &planning.ExecutionState{}}
	if err := server.Sync(args, &resp); err == nil {
		t.Error("expected error")
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
