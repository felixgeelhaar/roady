package mcp_test

import (
	"os"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/mcp"
)

func TestNewServer_Initialization(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-mcp-init-*")
	defer os.RemoveAll(tempDir)

	s := mcp.NewServer(tempDir)
	if s == nil {
		t.Fatal("expected server instance")
	}
	// We can't easily call Start() or tools without complex mocking of stdio and context,
	// but reaching this point covers the registration logic in registerTools().
}