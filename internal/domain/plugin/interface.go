package plugin

import (
	"net/rpc"

	"github.com/felixgeelhaar/roady/internal/domain/planning"
	"github.com/hashicorp/go-plugin"
)

// Syncer is the interface that plugins must implement.
type Syncer interface {
	// Init ensures the plugin can connect (auth check)
	Init(config map[string]string) error

	// Sync performs the bi-directional synchronization
	Sync(plan *planning.Plan, state *planning.ExecutionState) (*SyncResult, error)
}

// SyncResult captures the outcome of a sync operation
type SyncResult struct {
	StatusUpdates map[string]planning.TaskStatus `json:"status_updates"`
	LinkUpdates   map[string]planning.ExternalRef `json:"link_updates"`
	Errors        []string                       `json:"errors"`
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

func (g *SyncerRPCClient) Init(config map[string]string) error {
	var resp interface{}
	return g.Client.Call("Plugin.Init", config, &resp)
}

func (g *SyncerRPCClient) Sync(plan *planning.Plan, state *planning.ExecutionState) (*SyncResult, error) {
	var resp SyncResult
	args := &SyncArgs{Plan: plan, State: state}
	err := g.Client.Call("Plugin.Sync", args, &resp)
	return &resp, err
}

type SyncerRPCServer struct{ Impl Syncer }

func (s *SyncerRPCServer) Init(config map[string]string, resp *interface{}) error {
	return s.Impl.Init(config)
}

func (s *SyncerRPCServer) Sync(args *SyncArgs, resp *SyncResult) error {
	result, err := s.Impl.Sync(args.Plan, args.State)
	if result != nil {
		*resp = *result
	}
	return err
}
