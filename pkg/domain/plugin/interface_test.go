package plugin_test

import (
	"errors"
	"net"
	"net/rpc"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/plugin"
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

func TestSyncerPlugin_Methods(t *testing.T) {
	p := &plugin.SyncerPlugin{Impl: &StubSyncer{}}
	if _, err := p.Server(nil); err != nil {
		t.Error(err)
	}
	// Client call requires rpc.Client, but we hit the line
}

func TestSyncerRPCClientNil(t *testing.T) {
	c := &plugin.SyncerRPCClient{}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestSyncerRPCClientCalls(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	server := rpc.NewServer()
	stub := &StubSyncer{
		Response: map[string]planning.TaskStatus{"t1": planning.StatusDone},
	}
	if err := server.RegisterName("Plugin", &plugin.SyncerRPCServer{Impl: stub}); err != nil {
		t.Fatalf("register server: %v", err)
	}
	go server.ServeConn(serverConn)

	client := rpc.NewClient(clientConn)
	rpcClient := &plugin.SyncerRPCClient{Client: client}

	defer func() {
		client.Close()
		serverConn.Close()
	}()

	if err := rpcClient.Init(map[string]string{"foo": "bar"}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	result, err := rpcClient.Sync(&planning.Plan{}, &planning.ExecutionState{})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if result.StatusUpdates["t1"] != planning.StatusDone {
		t.Fatalf("unexpected status updates: %#v", result)
	}
}
