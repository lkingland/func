package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolHandlerFunc decorates mcp tool handlers with a command prefix and executor
type toolHandlerFunc func(context.Context, toolRequestInterface, string, Executor) (*mcp.CallToolResult, error)

// with creates a mcp.ToolHandler that injects the prefix and executor
func with(prefix string, executor Executor, impl toolHandlerFunc) mcp.ToolHandler {
	return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Gate all tool interactions behind environment variable check
		if err := checkMCPEnabled(); err != nil {
			return errorResult(err.Error()), nil
		}
		// Convert *mcp.CallToolRequest to toolRequestInterface
		// Both types implement GetParams(), so this works transparently
		return impl(ctx, request, prefix, executor)
	}
}

// errorResult creates an error CallToolResult with the given message.
// Used by tools to return errors in MCP protocol format.
func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}

// parseCommand splits a command string like "kn func" into its parts
func parseCommand(cmdPrefix string) []string {
	return strings.Fields(cmdPrefix)
}

// validatePathExists performs a simple filesystem check to verify a path exists.
// This is the ONLY validation we do - we do NOT read func.yaml or validate
// function structure. The func binary handles all function-specific validation.
func validatePathExists(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", path)
		}
		return fmt.Errorf("cannot access path: %s", path)
	}
	return nil
}

// toolRequestInterface allows for testing with mock requests
type toolRequestInterface interface {
	GetParams() mcp.Params
}

// unmarshalToolInput unmarshals the MCP request arguments into a typed struct.
// Uses Go generics to provide type-safe unmarshaling.
// Accepts an interface to allow for easier testing with mocks.
func unmarshalToolInput[T any](request toolRequestInterface) (T, error) {
	var result T
	params := request.GetParams().(*mcp.CallToolParamsRaw)
	if err := json.Unmarshal(params.Arguments, &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal arguments: %w", err)
	}
	return result, nil
}

// jsonResult creates a successful CallToolResult with JSON-serialized content.
// Used for returning structured output from tools.
func jsonResult(v any) *mcp.CallToolResult {
	data, err := json.Marshal(v)
	if err != nil {
		// This should never happen if v is properly structured
		return errorResult(fmt.Sprintf("failed to marshal result: %v", err))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}
}

// appendStringFlag adds a string flag to args only if the value is non-nil and non-empty.
// This ensures we only pass flags that were explicitly provided by the user.
func appendStringFlag(args []string, flag string, value *string) []string {
	if value != nil && *value != "" {
		return append(args, flag, *value)
	}
	return args
}

// appendBoolFlag adds a boolean flag to args only if the value is non-nil and true.
// This ensures we only pass flags that were explicitly provided by the user.
func appendBoolFlag(args []string, flag string, value *bool) []string {
	if value != nil && *value {
		return append(args, flag)
	}
	return args
}

// healthCheck is a simple tool for testing MCP server connectivity
type healthCheck struct{}

func (t healthCheck) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "healthcheck",
		Description: "Checks if the MCP server is running and responsive",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func (t healthCheck) handle(ctx context.Context, request toolRequestInterface, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	return jsonResult(map[string]string{
		"status":  "ok",
		"message": "The MCP server is running!",
	}), nil
}
