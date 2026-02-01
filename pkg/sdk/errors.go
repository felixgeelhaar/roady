package sdk

import (
	"errors"
	"fmt"
)

// ErrNoContent is returned when a tool result contains no content items.
var ErrNoContent = errors.New("roady: empty tool result")

// ToolError is returned when a tool call returns an error result.
type ToolError struct {
	Tool    string
	Message string
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("roady: tool %s: %s", e.Tool, e.Message)
}
