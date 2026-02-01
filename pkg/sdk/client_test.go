package sdk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/mcp-go/client"
)

func TestTextResult(t *testing.T) {
	t.Run("extracts text", func(t *testing.T) {
		r := &client.ToolResult{
			Content: []client.ContentItem{{Type: "text", Text: "hello"}},
		}
		got, err := textResult(r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "hello" {
			t.Fatalf("got %q, want %q", got, "hello")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		r := &client.ToolResult{}
		_, err := textResult(r)
		if err != ErrNoContent {
			t.Fatalf("got %v, want ErrNoContent", err)
		}
	})
}

func TestUnmarshalText(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		r := &client.ToolResult{
			Content: []client.ContentItem{{Type: "text", Text: `{"id":"s1","title":"My Spec"}`}},
		}
		spec, err := unmarshalText[Spec](r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if spec.ID != "s1" || spec.Title != "My Spec" {
			t.Fatalf("unexpected spec: %+v", spec)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		r := &client.ToolResult{
			Content: []client.ContentItem{{Type: "text", Text: "not json"}},
		}
		_, err := unmarshalText[Spec](r)
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		r := &client.ToolResult{}
		_, err := unmarshalText[Spec](r)
		if err != ErrNoContent {
			t.Fatalf("got %v, want ErrNoContent", err)
		}
	})
}

func TestMajorVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1.0.0", "1"},
		{"2.3.4", "2"},
		{"10.0.1", "10"},
		{"0.1.0", "0"},
		{"3", "3"},
	}
	for _, tt := range tests {
		got := majorVersion(tt.input)
		if got != tt.want {
			t.Errorf("majorVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToolError(t *testing.T) {
	e := &ToolError{Tool: "roady_init", Message: "bad"}
	if !strings.Contains(e.Error(), "roady_init") {
		t.Fatalf("error should contain tool name: %s", e.Error())
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := cwd
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return cwd
}

func TestIntegrationInitGetSpec(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	root := findRepoRoot(t)
	tempDir := t.TempDir()

	binPath := filepath.Join(tempDir, "roady")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/roady")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build roady: %v\n%s", err, out)
	}

	cmd := fmt.Sprintf("cd '%s' && '%s' mcp --transport stdio", tempDir, binPath)
	transport, err := client.NewStdioTransport("bash", "-lc", cmd)
	if err != nil {
		t.Fatalf("stdio transport: %v", err)
	}
	defer transport.Close()

	c := NewClient(transport, WithTimeout(60*time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	info, err := c.Initialize(ctx)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if !info.Capabilities.Tools {
		t.Fatalf("expected tools capability")
	}

	msg, err := c.Init(ctx, "test-sdk")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if !strings.Contains(msg, "test-sdk") {
		t.Fatalf("unexpected init result: %s", msg)
	}

	spec, err := c.GetSpec(ctx)
	if err != nil {
		t.Fatalf("get spec: %v", err)
	}
	if spec.Title != "test-sdk" {
		t.Fatalf("unexpected spec title: %s", spec.Title)
	}

	// GetSchema should return valid schema info
	schema, err := c.GetSchema(ctx)
	if err != nil {
		t.Fatalf("get schema: %v", err)
	}
	if schema.SchemaVersion == "" {
		t.Fatal("expected non-empty schema version")
	}
	if schema.ServerVersion == "" {
		t.Fatal("expected non-empty server version")
	}
	if schema.Changelog == "" {
		t.Fatal("expected non-empty changelog URL")
	}

	// Compatible should pass for current SDK
	if err := c.Compatible(ctx); err != nil {
		t.Fatalf("compatible: %v", err)
	}
}
