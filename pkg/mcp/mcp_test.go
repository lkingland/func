package mcp

import (
	"context"
	"testing"

	mcpmock "knative.dev/func/pkg/mcp/mock"
)

// TestServerCreation verifies that the MCP server can be instantiated with custom options
func TestServerCreation(t *testing.T) {
	// Define a mock executor
	mock := mcpmock.NewExecutor()
	mock.ExecuteFn = func(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
		return []byte("success"), nil
	}

	// Instantiate the MCP Server with mock executor and custom prefix
	server := New(WithPrefix("kn func"), WithExecutor(mock))

	// Verify server was created
	if server == nil {
		t.Fatal("expected server to be non-nil")
	}

	// Verify prefix was set correctly
	if server.prefix != "kn func" {
		t.Errorf("expected prefix 'kn func', got '%s'", server.prefix)
	}

	// Verify executor was set correctly
	if server.executor != mock {
		t.Error("expected executor to be the mock")
	}

	// Verify tools were registered
	if len(server.tools) != 9 {
		t.Errorf("expected 9 tools, got %d", len(server.tools))
	}

	// Verify resources were registered
	if len(server.resources) != 13 {
		t.Errorf("expected 13 resources, got %d", len(server.resources))
	}

	// Verify prompts were registered
	if len(server.prompts) != 3 {
		t.Errorf("expected 3 prompts, got %d", len(server.prompts))
	}
}
