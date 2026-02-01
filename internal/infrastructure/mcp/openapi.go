package mcp

import (
	"encoding/json"
	"fmt"

	mcplib "github.com/felixgeelhaar/mcp-go"
)

// OpenAPISpec represents a minimal OpenAPI 3.0 document.
type OpenAPISpec struct {
	OpenAPI string                `json:"openapi"`
	Info    OpenAPIInfo           `json:"info"`
	Paths   map[string]PathItem  `json:"paths"`
}

// OpenAPIInfo is the info section of an OpenAPI spec.
type OpenAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

// PathItem represents a single path with operations.
type PathItem struct {
	Post *Operation `json:"post,omitempty"`
}

// Operation is an OpenAPI operation.
type Operation struct {
	OperationID string              `json:"operationId"`
	Summary     string              `json:"summary,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
	Tags        []string            `json:"tags,omitempty"`
}

// RequestBody is the request body definition.
type RequestBody struct {
	Required bool                  `json:"required"`
	Content  map[string]MediaType  `json:"content"`
}

// MediaType describes a media type with schema.
type MediaType struct {
	Schema any `json:"schema"`
}

// Response is an OpenAPI response.
type Response struct {
	Description string `json:"description"`
}

// OpenAPI returns the OpenAPI 3.0 JSON document for this server.
func (s *Server) OpenAPI() ([]byte, error) {
	return GenerateOpenAPI(s.mcpServer)
}

// GenerateOpenAPI produces an OpenAPI 3.0 JSON document from the server's registered tools.
// Each MCP tool is mapped to a POST endpoint at /tools/{tool_name}.
func GenerateOpenAPI(srv *mcplib.Server) ([]byte, error) {
	tools := srv.Tools()

	paths := make(map[string]PathItem, len(tools))
	for _, t := range tools {
		op := Operation{
			OperationID: t.Name,
			Summary:     t.Description,
			Responses: map[string]Response{
				"200": {Description: "Successful response"},
				"400": {Description: "Invalid request parameters"},
				"500": {Description: "Internal server error"},
			},
			Tags: []string{"roady"},
		}

		if t.InputSchema != nil {
			if hasProperties(t.InputSchema) {
				op.RequestBody = &RequestBody{
					Required: true,
					Content: map[string]MediaType{
						"application/json": {Schema: t.InputSchema},
					},
				}
			}
		}

		path := fmt.Sprintf("/tools/%s", t.Name)
		paths[path] = PathItem{Post: &op}
	}

	spec := OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: OpenAPIInfo{
			Title:       "Roady MCP API",
			Description: "Auto-generated OpenAPI spec from Roady MCP tool registrations.",
			Version:     SchemaVersion,
		},
		Paths: paths,
	}

	return json.MarshalIndent(spec, "", "  ")
}

// hasProperties checks whether a JSON Schema has any properties defined.
func hasProperties(schema any) bool {
	m, ok := schema.(map[string]any)
	if !ok {
		return false
	}
	props, ok := m["properties"]
	if !ok {
		return false
	}
	pm, ok := props.(map[string]any)
	if !ok {
		return false
	}
	return len(pm) > 0
}
