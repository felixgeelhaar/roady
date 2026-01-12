package plugin

import (
	"net/rpc"

	"github.com/felixgeelhaar/roady/internal/domain/planning"
	"github.com/hashicorp/go-plugin"
)

// Syncer is the interface that plugins must implement.
type Syncer interface {
	// Sync pushes the Roady plan to the external system and returns any status updates.
	Sync(plan *planning.Plan, state *planning.ExecutionState) (map[string]planning.TaskStatus, error)
}

// SyncerPlugin is the implementation of plugin.Plugin so we can serve/consume this.
type SyncerPlugin struct {
	Impl Syncer
}

func (p *SyncerPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &SyncerRPCServer{Impl: p.Impl}, nil
}

func (p *SyncerPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &SyncerRPCClient{Client: c}, nil
}

// RPC Client/Server wrappers
type SyncArgs struct {
	Plan  *planning.Plan
	State *planning.ExecutionState
}

type SyncerRPCClient struct{ Client *rpc.Client }

func (g *SyncerRPCClient) Sync(plan *planning.Plan, state *planning.ExecutionState) (map[string]planning.TaskStatus, error) {
	var resp map[string]planning.TaskStatus
	args := &SyncArgs{Plan: plan, State: state}
	err := g.Client.Call("Plugin.Sync", args, &resp)
	return resp, err
}

type SyncerRPCServer struct{ Impl Syncer }

func (s *SyncerRPCServer) Sync(args *SyncArgs, resp *map[string]planning.TaskStatus) error {
	var err error
	*resp, err = s.Impl.Sync(args.Plan, args.State)
	return err
}
