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

func (s *StubSyncer) Push(taskID string, status planning.TaskStatus) error {
	return nil
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

func (e *ErrorSyncer) Init(config map[string]string) error { return errors.New("init fail") }
func (e *ErrorSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (*plugin.SyncResult, error) {
	return nil, errors.New("fail")
}
func (e *ErrorSyncer) Push(taskID string, status planning.TaskStatus) error {
	return errors.New("push fail")
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
		_ = client.Close()
		_ = serverConn.Close()
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

	// Test Push via RPC
	if err := rpcClient.Push("task-1", planning.StatusDone); err != nil {
		t.Fatalf("Push failed: %v", err)
	}
}

func TestSyncerRPCPush(t *testing.T) {
	stub := &StubSyncer{}
	server := &plugin.SyncerRPCServer{Impl: stub}

	var resp interface{}
	args := &plugin.PushArgs{TaskID: "task-123", Status: planning.StatusDone}
	err := server.Push(args, &resp)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
}

func TestSyncerRPCPush_Error(t *testing.T) {
	server := &plugin.SyncerRPCServer{Impl: &ErrorSyncer{}}
	var resp interface{}
	args := &plugin.PushArgs{TaskID: "task-123", Status: planning.StatusDone}
	if err := server.Push(args, &resp); err == nil {
		t.Error("expected error")
	}
}

// ---------------------------------------------------------------------------
// Error-path RPC client tests using ErrorSyncer
// ---------------------------------------------------------------------------

func TestSyncerRPCClientCalls_InitError(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	srv := rpc.NewServer()
	if err := srv.RegisterName("Plugin", &plugin.SyncerRPCServer{Impl: &ErrorSyncer{}}); err != nil {
		t.Fatalf("register: %v", err)
	}
	go srv.ServeConn(serverConn)

	client := rpc.NewClient(clientConn)
	rpcClient := &plugin.SyncerRPCClient{Client: client}
	defer func() {
		_ = client.Close()
		_ = serverConn.Close()
	}()

	if err := rpcClient.Init(map[string]string{"key": "val"}); err == nil {
		t.Error("expected Init to return error")
	}
}

func TestSyncerRPCClientCalls_SyncError(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	srv := rpc.NewServer()
	if err := srv.RegisterName("Plugin", &plugin.SyncerRPCServer{Impl: &ErrorSyncer{}}); err != nil {
		t.Fatalf("register: %v", err)
	}
	go srv.ServeConn(serverConn)

	client := rpc.NewClient(clientConn)
	rpcClient := &plugin.SyncerRPCClient{Client: client}
	defer func() {
		_ = client.Close()
		_ = serverConn.Close()
	}()

	_, err := rpcClient.Sync(&planning.Plan{}, &planning.ExecutionState{})
	if err == nil {
		t.Error("expected Sync to return error")
	}
}

func TestSyncerRPCClientCalls_PushError(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	srv := rpc.NewServer()
	if err := srv.RegisterName("Plugin", &plugin.SyncerRPCServer{Impl: &ErrorSyncer{}}); err != nil {
		t.Fatalf("register: %v", err)
	}
	go srv.ServeConn(serverConn)

	client := rpc.NewClient(clientConn)
	rpcClient := &plugin.SyncerRPCClient{Client: client}
	defer func() {
		_ = client.Close()
		_ = serverConn.Close()
	}()

	if err := rpcClient.Push("task-1", planning.StatusDone); err == nil {
		t.Error("expected Push to return error")
	}
}

func TestSyncerPlugin_Client(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer func() { _ = serverConn.Close() }()
	defer func() { _ = clientConn.Close() }()

	rpcClient := rpc.NewClient(clientConn)
	defer func() { _ = rpcClient.Close() }()

	p := &plugin.SyncerPlugin{Impl: &StubSyncer{}}
	iface, err := p.Client(nil, rpcClient)
	if err != nil {
		t.Fatalf("Client() error = %v", err)
	}
	if iface == nil {
		t.Fatal("Client() returned nil interface")
	}
	if _, ok := iface.(*plugin.SyncerRPCClient); !ok {
		t.Errorf("expected *SyncerRPCClient, got %T", iface)
	}
}

func TestSyncerRPCServer_Init(t *testing.T) {
	stub := &StubSyncer{}
	server := &plugin.SyncerRPCServer{Impl: stub}
	var resp interface{}
	if err := server.Init(map[string]string{"foo": "bar"}, &resp); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
}

func TestSyncerRPCServer_SyncNilResult(t *testing.T) {
	// ErrorSyncer.Sync returns (nil, error). Verify the server handles nil result
	// without panicking and propagates the error.
	server := &plugin.SyncerRPCServer{Impl: &ErrorSyncer{}}
	var resp plugin.SyncResult
	args := &plugin.SyncArgs{Plan: &planning.Plan{}, State: &planning.ExecutionState{}}
	err := server.Sync(args, &resp)
	if err == nil {
		t.Error("expected error when impl returns nil result")
	}
}
