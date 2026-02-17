package plugin_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	domainPlugin "github.com/felixgeelhaar/roady/pkg/domain/plugin"
	pb "github.com/felixgeelhaar/roady/pkg/domain/plugin/proto"
	infraPlugin "github.com/felixgeelhaar/roady/pkg/plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// fakeSyncer is a test implementation of the Syncer interface.
type fakeSyncer struct {
	initErr error
	pushErr error
	syncRes *domainPlugin.SyncResult
}

func (f *fakeSyncer) Init(config map[string]string) error {
	if f.initErr != nil {
		return f.initErr
	}
	return nil
}

func (f *fakeSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (*domainPlugin.SyncResult, error) {
	if f.syncRes != nil {
		return f.syncRes, nil
	}
	return &domainPlugin.SyncResult{
		StatusUpdates: map[string]planning.TaskStatus{},
	}, nil
}

func (f *fakeSyncer) Push(taskID string, status planning.TaskStatus) error {
	if f.pushErr != nil {
		return f.pushErr
	}
	return nil
}

// startGRPCServer starts an in-process gRPC server with bufconn.
func startGRPCServer(t *testing.T, impl domainPlugin.Syncer) (*grpc.ClientConn, func()) {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pb.RegisterSyncerServer(s, &infraPlugin.GRPCServer{Impl: impl})

	go func() {
		if err := s.Serve(lis); err != nil {
			// server stopped
		}
	}()

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}

	cleanup := func() {
		conn.Close()
		s.Stop()
		lis.Close()
	}

	return conn, cleanup
}

func TestGRPCClientServer_Init_Success(t *testing.T) {
	impl := &fakeSyncer{}
	conn, cleanup := startGRPCServer(t, impl)
	defer cleanup()

	client := infraPlugin.NewGRPCClient(conn)
	err := client.Init(map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
}

func TestGRPCClientServer_Init_Error(t *testing.T) {
	impl := &fakeSyncer{initErr: errors.New("init failed")}
	conn, cleanup := startGRPCServer(t, impl)
	defer cleanup()

	client := infraPlugin.NewGRPCClient(conn)
	err := client.Init(map[string]string{})
	if err == nil {
		t.Fatal("expected error from Init")
	}
	if err.Error() != "init failed" {
		t.Errorf("got error %q, want %q", err.Error(), "init failed")
	}
}

func TestGRPCClientServer_Sync_Success(t *testing.T) {
	impl := &fakeSyncer{
		syncRes: &domainPlugin.SyncResult{
			StatusUpdates: map[string]planning.TaskStatus{
				"task-1": planning.StatusDone,
				"task-2": planning.StatusInProgress,
			},
			LinkUpdates: map[string]planning.ExternalRef{
				"task-1": {
					ID:         "ext-1",
					Identifier: "GH-123",
					URL:        "https://github.com/test/issues/123",
				},
			},
			Errors: []string{"warning: task-3 not found"},
		},
	}
	conn, cleanup := startGRPCServer(t, impl)
	defer cleanup()

	client := infraPlugin.NewGRPCClient(conn)

	plan := &planning.Plan{
		ID:     "plan-1",
		SpecID: "spec-1",
		Tasks: []planning.Task{
			{ID: "task-1", Title: "Build UI", Priority: planning.PriorityHigh, DependsOn: []string{}},
			{ID: "task-2", Title: "Build API", Priority: planning.PriorityMedium, DependsOn: []string{"task-1"}},
		},
		ApprovalStatus: planning.ApprovalApproved,
	}
	state := &planning.ExecutionState{
		TaskStates: map[string]planning.TaskResult{
			"task-1": {Status: planning.StatusInProgress, Owner: "alice"},
		},
	}

	result, err := client.Sync(plan, state)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(result.StatusUpdates) != 2 {
		t.Errorf("got %d status updates, want 2", len(result.StatusUpdates))
	}
	if result.StatusUpdates["task-1"] != planning.StatusDone {
		t.Errorf("task-1 status: got %q, want %q", result.StatusUpdates["task-1"], planning.StatusDone)
	}
	if len(result.LinkUpdates) != 1 {
		t.Errorf("got %d link updates, want 1", len(result.LinkUpdates))
	}
	if len(result.Errors) != 1 {
		t.Errorf("got %d errors, want 1", len(result.Errors))
	}
}

func TestGRPCClientServer_Sync_EmptyPlan(t *testing.T) {
	impl := &fakeSyncer{}
	conn, cleanup := startGRPCServer(t, impl)
	defer cleanup()

	client := infraPlugin.NewGRPCClient(conn)
	result, err := client.Sync(&planning.Plan{}, &planning.ExecutionState{})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestGRPCClientServer_Push_Success(t *testing.T) {
	impl := &fakeSyncer{}
	conn, cleanup := startGRPCServer(t, impl)
	defer cleanup()

	client := infraPlugin.NewGRPCClient(conn)
	err := client.Push("task-1", planning.StatusDone)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
}

func TestGRPCClientServer_Push_Error(t *testing.T) {
	impl := &fakeSyncer{pushErr: errors.New("push rejected")}
	conn, cleanup := startGRPCServer(t, impl)
	defer cleanup()

	client := infraPlugin.NewGRPCClient(conn)
	err := client.Push("task-1", planning.StatusDone)
	if err == nil {
		t.Fatal("expected error from Push")
	}
	if err.Error() != "push rejected" {
		t.Errorf("got error %q, want %q", err.Error(), "push rejected")
	}
}

func TestGRPCClientServer_Sync_NilPlanAndState(t *testing.T) {
	impl := &fakeSyncer{}
	conn, cleanup := startGRPCServer(t, impl)
	defer cleanup()

	client := infraPlugin.NewGRPCClient(conn)
	result, err := client.Sync(nil, nil)
	if err != nil {
		t.Fatalf("Sync with nil args failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}
