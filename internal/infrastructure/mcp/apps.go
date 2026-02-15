package mcp

import (
	"context"
	"embed"

	mcplib "github.com/felixgeelhaar/mcp-go"
)

//go:embed all:dist
var distFS embed.FS

// appEntry maps a UI resource URI to its dist file path.
type appEntry struct {
	uri      string
	filePath string
}

var appEntries = []appEntry{
	{uri: "ui://roady/status", filePath: "dist/status.html"},
	{uri: "ui://roady/spec", filePath: "dist/spec.html"},
	{uri: "ui://roady/plan", filePath: "dist/plan.html"},
	{uri: "ui://roady/state", filePath: "dist/state.html"},
	{uri: "ui://roady/drift", filePath: "dist/drift.html"},
	{uri: "ui://roady/policy", filePath: "dist/policy.html"},
	{uri: "ui://roady/forecast", filePath: "dist/forecast.html"},
	{uri: "ui://roady/usage", filePath: "dist/usage.html"},
	{uri: "ui://roady/org", filePath: "dist/org.html"},
	{uri: "ui://roady/git-sync", filePath: "dist/git-sync.html"},
	{uri: "ui://roady/sync", filePath: "dist/sync.html"},
	{uri: "ui://roady/deps", filePath: "dist/deps.html"},
	{uri: "ui://roady/debt", filePath: "dist/debt.html"},
	{uri: "ui://roady/init", filePath: "dist/init.html"},
	{uri: "ui://roady/billing", filePath: "dist/billing.html"},
}

func (s *Server) registerApps() {
	for _, entry := range appEntries {
		fp := entry.filePath
		uri := entry.uri
		s.mcpServer.Resource(uri).
			Name(uri).
			Description("MCP App UI for " + uri).
			MimeType("text/html;profile=mcp-app").
			Handler(func(_ context.Context, _ string, _ map[string]string) (*mcplib.ResourceContent, error) {
				data, err := distFS.ReadFile(fp)
				if err != nil {
					return nil, err
				}
				return &mcplib.ResourceContent{
					URI:      uri,
					MimeType: "text/html;profile=mcp-app",
					Text:     string(data),
				}, nil
			})
	}
}
