package mcp

import infra "github.com/felixgeelhaar/roady/internal/infrastructure/mcp"

// Server exposes the MCP server implementation from the infrastructure layer.
type Server = infra.Server

// NewServer constructs an MCP server rooted at the provided path.
func NewServer(root string) (*Server, error) {
	return infra.NewServer(root)
}
