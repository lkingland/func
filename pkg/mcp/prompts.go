package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// promptHandlerFunc decorates mcp prompt handlers with a command prefix and executor
type promptHandlerFunc func(context.Context, *mcp.GetPromptRequest, string, Executor) (*mcp.GetPromptResult, error)

// withPromptPrefix creates a mcp.PromptHandler that injects the prefix and executor
func withPromptPrefix(prefix string, executor Executor, impl promptHandlerFunc) mcp.PromptHandler {
	return func(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return impl(ctx, request, prefix, executor)
	}
}

// getArgOrDefault extracts an argument value or returns a default
func getArgOrDefault(args map[string]string, key, defaultValue string) string {
	if val, ok := args[key]; ok && val != "" {
		return val
	}
	return defaultValue
}
