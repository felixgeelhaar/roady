package mcp_test

import (
	"os"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/mcp"
)

func TestNewServer_Initialization(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-mcp-init-*")
	defer os.RemoveAll(tempDir)

	s, err := mcp.NewServer(tempDir)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	if s == nil {
		t.Fatal("expected server instance")
	}
	// We can't easily call Start() or tools without complex mocking of stdio and context,
	// but reaching this point covers the registration logic in registerTools().
}
