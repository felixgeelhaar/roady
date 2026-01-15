// Package plugin provides plugin communication infrastructure.
package plugin

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	domainPlugin "github.com/felixgeelhaar/roady/pkg/domain/plugin"
	pb "github.com/felixgeelhaar/roady/pkg/domain/plugin/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCClient is a client that implements the Syncer interface over gRPC.
type GRPCClient struct {
	client pb.SyncerClient
}

// NewGRPCClient creates a new gRPC client.
func NewGRPCClient(conn *grpc.ClientConn) *GRPCClient {
	return &GRPCClient{client: pb.NewSyncerClient(conn)}
}

// Init implements Syncer.Init over gRPC.
func (c *GRPCClient) Init(config map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := c.client.Init(ctx, &pb.InitRequest{Config: config})
	if err != nil {
		return err
	}
	if !resp.Success {
		return errors.New(resp.Error)
	}
	return nil
}

// Sync implements Syncer.Sync over gRPC.
func (c *GRPCClient) Sync(plan *planning.Plan, state *planning.ExecutionState) (*domainPlugin.SyncResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	resp, err := c.client.Sync(ctx, &pb.SyncRequest{
		Plan:  planToProto(plan),
		State: stateToProto(state),
	})
	if err != nil {
		return nil, err
	}

	return syncResultFromProto(resp), nil
}

// Push implements Syncer.Push over gRPC.
func (c *GRPCClient) Push(taskID string, status planning.TaskStatus) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := c.client.Push(ctx, &pb.PushRequest{
		TaskId: taskID,
		Status: string(status),
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return errors.New(resp.Error)
	}
	return nil
}

// GRPCServer wraps a Syncer implementation as a gRPC server.
type GRPCServer struct {
	pb.UnimplementedSyncerServer
	Impl domainPlugin.Syncer
}

// Init implements the gRPC Init method.
func (s *GRPCServer) Init(ctx context.Context, req *pb.InitRequest) (*pb.InitResponse, error) {
	err := s.Impl.Init(req.Config)
	if err != nil {
		return &pb.InitResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.InitResponse{Success: true}, nil
}

// Sync implements the gRPC Sync method.
func (s *GRPCServer) Sync(ctx context.Context, req *pb.SyncRequest) (*pb.SyncResponse, error) {
	plan := planFromProto(req.Plan)
	state := stateFromProto(req.State)

	result, err := s.Impl.Sync(plan, state)
	if err != nil {
		return nil, err
	}

	return syncResultToProto(result), nil
}

// Push implements the gRPC Push method.
func (s *GRPCServer) Push(ctx context.Context, req *pb.PushRequest) (*pb.PushResponse, error) {
	err := s.Impl.Push(req.TaskId, planning.TaskStatus(req.Status))
	if err != nil {
		return &pb.PushResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.PushResponse{Success: true}, nil
}

// Subscribe implements the gRPC Subscribe method for streaming events.
func (s *GRPCServer) Subscribe(req *pb.SubscribeRequest, stream pb.Syncer_SubscribeServer) error {
	// Streaming not implemented in this iteration
	return nil
}

// Conversion functions

func planToProto(plan *planning.Plan) *pb.Plan {
	if plan == nil {
		return nil
	}

	tasks := make([]*pb.Task, len(plan.Tasks))
	for i, t := range plan.Tasks {
		tasks[i] = taskToProto(t)
	}

	return &pb.Plan{
		Id:             plan.ID,
		SpecId:         plan.SpecID,
		Tasks:          tasks,
		ApprovalStatus: string(plan.ApprovalStatus),
		CreatedAt:      timestamppb.New(plan.CreatedAt),
		UpdatedAt:      timestamppb.New(plan.UpdatedAt),
	}
}

func taskToProto(t planning.Task) *pb.Task {
	return &pb.Task{
		Id:           t.ID,
		Title:        t.Title,
		Description:  t.Description,
		Phase:        string(t.Priority), // Map Priority to Phase field
		Dependencies: t.DependsOn,
	}
}

func stateToProto(state *planning.ExecutionState) *pb.ExecutionState {
	if state == nil {
		return nil
	}

	taskStates := make(map[string]*pb.TaskResult)
	for k, v := range state.TaskStates {
		taskStates[k] = taskResultToProto(v)
	}

	return &pb.ExecutionState{
		TaskStates: taskStates,
	}
}

func taskResultToProto(r planning.TaskResult) *pb.TaskResult {
	refs := make(map[string]*pb.ExternalRef)
	for k, v := range r.ExternalRefs {
		refs[k] = externalRefToProto(v)
	}

	return &pb.TaskResult{
		Status:       string(r.Status),
		Owner:        r.Owner,
		ExternalRefs: refs,
	}
}

func externalRefToProto(ref planning.ExternalRef) *pb.ExternalRef {
	return &pb.ExternalRef{
		Id:           ref.ID,
		Identifier:   ref.Identifier,
		Url:          ref.URL,
		LastSyncedAt: timestamppb.New(ref.LastSyncedAt),
	}
}

func planFromProto(p *pb.Plan) *planning.Plan {
	if p == nil {
		return nil
	}

	tasks := make([]planning.Task, len(p.Tasks))
	for i, t := range p.Tasks {
		tasks[i] = taskFromProto(t)
	}

	return &planning.Plan{
		ID:             p.Id,
		SpecID:         p.SpecId,
		Tasks:          tasks,
		ApprovalStatus: planning.ApprovalStatus(p.ApprovalStatus),
		CreatedAt:      p.CreatedAt.AsTime(),
		UpdatedAt:      p.UpdatedAt.AsTime(),
	}
}

func taskFromProto(t *pb.Task) planning.Task {
	return planning.Task{
		ID:          t.Id,
		Title:       t.Title,
		Description: t.Description,
		Priority:    planning.TaskPriority(t.Phase),
		DependsOn:   t.Dependencies,
	}
}

func stateFromProto(s *pb.ExecutionState) *planning.ExecutionState {
	if s == nil {
		return nil
	}

	taskStates := make(map[string]planning.TaskResult)
	for k, v := range s.TaskStates {
		taskStates[k] = taskResultFromProto(v)
	}

	return &planning.ExecutionState{
		TaskStates: taskStates,
	}
}

func taskResultFromProto(r *pb.TaskResult) planning.TaskResult {
	refs := make(map[string]planning.ExternalRef)
	for k, v := range r.ExternalRefs {
		refs[k] = externalRefFromProto(v)
	}

	return planning.TaskResult{
		Status:       planning.TaskStatus(r.Status),
		Owner:        r.Owner,
		ExternalRefs: refs,
	}
}

func externalRefFromProto(ref *pb.ExternalRef) planning.ExternalRef {
	return planning.ExternalRef{
		ID:           ref.Id,
		Identifier:   ref.Identifier,
		URL:          ref.Url,
		LastSyncedAt: ref.LastSyncedAt.AsTime(),
	}
}

func syncResultToProto(result *domainPlugin.SyncResult) *pb.SyncResponse {
	statusUpdates := make(map[string]string)
	for k, v := range result.StatusUpdates {
		statusUpdates[k] = string(v)
	}

	linkUpdates := make(map[string]*pb.ExternalRef)
	for k, v := range result.LinkUpdates {
		linkUpdates[k] = externalRefToProto(v)
	}

	return &pb.SyncResponse{
		StatusUpdates: statusUpdates,
		LinkUpdates:   linkUpdates,
		Errors:        result.Errors,
	}
}

func syncResultFromProto(resp *pb.SyncResponse) *domainPlugin.SyncResult {
	statusUpdates := make(map[string]planning.TaskStatus)
	for k, v := range resp.StatusUpdates {
		statusUpdates[k] = planning.TaskStatus(v)
	}

	linkUpdates := make(map[string]planning.ExternalRef)
	for k, v := range resp.LinkUpdates {
		linkUpdates[k] = externalRefFromProto(v)
	}

	return &domainPlugin.SyncResult{
		StatusUpdates: statusUpdates,
		LinkUpdates:   linkUpdates,
		Errors:        resp.Errors,
	}
}
