package mcp

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"knative.dev/func/pkg/mcp/mock"
)

// TestDelete ensures the delete tool executes correctly with minimal arguments
func TestDelete(t *testing.T) {
	var (
		ctx, cancel          = context.WithCancel(context.Background())
		serverTpt, clientTpt = mcp.NewInMemoryTransports()
		client               = mcp.NewClient(&mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		}, nil)
	)
	defer cancel()

	executor := mock.NewExecutor()
	executor.ExecuteFn = func(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
		if dir != "" {
			t.Fatalf("expected empty dir, got %q", dir)
		}
		if name != "func" {
			t.Fatalf("expected command 'func', got %q", name)
		}

		// Validate args (order is deterministic)
		if len(args) != 2 {
			t.Fatalf("expected 2 args, got %d: %v", len(args), args)
		}
		if args[0] != "delete" {
			t.Fatalf("expected args[0]='delete', got %q", args[0])
		}
		if args[1] != "my-function" {
			t.Fatalf("expected args[1]='my-function', got %q", args[1])
		}

		return []byte("Function 'my-function' deleted successfully\n"), nil
	}

	// Connect Server
	server := New(WithExecutor(executor))
	serverSession, err := server.Connect(ctx, serverTpt)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	// Connect Client
	clientSession, err := client.Connect(ctx, clientTpt, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	// Send Request
	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "delete",
		Arguments: map[string]any{
			"name": "my-function",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", mcpError(result))
	}

	// Assert executor invoked
	if !executor.ExecuteInvoked {
		t.Fatal("executor was not invoked")
	}
}

// TestDelete_Options ensures the delete tool correctly passes all optional flags
func TestDelete_Options(t *testing.T) {
	var (
		path                 = t.TempDir()
		ctx, cancel          = context.WithCancel(context.Background())
		serverTpt, clientTpt = mcp.NewInMemoryTransports()
		client               = mcp.NewClient(&mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		}, nil)
	)
	defer cancel()

	executor := mock.NewExecutor()
	executor.ExecuteFn = func(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
		if dir != path {
			t.Fatalf("expected dir %q, got %q", path, dir)
		}
		if name != "func" {
			t.Fatalf("expected command 'func', got %q", name)
		}

		// Validate args (order is deterministic)
		// Expected: ["delete", "my-function", "--namespace", "prod", "--path", path, "--all", "true"]
		if len(args) != 8 {
			t.Fatalf("expected 8 args, got %d: %v", len(args), args)
		}
		if args[0] != "delete" {
			t.Fatalf("expected args[0]='delete', got %q", args[0])
		}
		if args[1] != "my-function" {
			t.Fatalf("expected args[1]='my-function', got %q", args[1])
		}
		if args[2] != "--namespace" || args[3] != "prod" {
			t.Fatalf("expected '--namespace prod', got %q %q", args[2], args[3])
		}
		if args[4] != "--path" || args[5] != path {
			t.Fatalf("expected '--path %s', got %q %q", path, args[4], args[5])
		}
		if args[6] != "--all" || args[7] != "true" {
			t.Fatalf("expected '--all true', got %q %q", args[6], args[7])
		}

		return []byte("Function and all related resources deleted\n"), nil
	}

	// Connect Server
	server := New(WithExecutor(executor))
	serverSession, err := server.Connect(ctx, serverTpt)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	// Connect Client
	clientSession, err := client.Connect(ctx, clientTpt, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	// Send Request
	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "delete",
		Arguments: map[string]any{
			"name":      "my-function",
			"namespace": "prod",
			"path":      path,
			"all":       "true",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", mcpError(result))
	}

	// Assert executor invoked
	if !executor.ExecuteInvoked {
		t.Fatal("executor was not invoked")
	}
}

// TestDelete_PathValidation ensures the tool validates path existence if provided
func TestDelete_PathValidation(t *testing.T) {
	var (
		ctx, cancel          = context.WithCancel(context.Background())
		serverTpt, clientTpt = mcp.NewInMemoryTransports()
		client               = mcp.NewClient(&mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		}, nil)
	)
	defer cancel()

	executor := mock.NewExecutor()
	executor.ExecuteFn = func(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
		t.Fatal("executor should not be called when path validation fails")
		return nil, nil
	}

	// Connect Server
	server := New(WithExecutor(executor))
	serverSession, err := server.Connect(ctx, serverTpt)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	// Connect Client
	clientSession, err := client.Connect(ctx, clientTpt, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	// Send Request with non-existent path
	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "delete",
		Arguments: map[string]any{
			"name": "my-function",
			"path": "/nonexistent/path/that/should/not/exist",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should return error without invoking executor
	if !result.IsError {
		t.Fatal("expected error result for non-existent path")
	}
	if executor.ExecuteInvoked {
		t.Fatal("executor should not be invoked when path validation fails")
	}

	// Error message should mention the path issue
	errMsg := mcpError(result)
	if !strings.Contains(errMsg, "path") && !strings.Contains(errMsg, "exist") {
		t.Errorf("expected error about path, got: %s", errMsg)
	}
}

// TestDelete_BinaryFailure ensures errors from the func binary are returned as MCP errors
func TestDelete_BinaryFailure(t *testing.T) {
	var (
		ctx, cancel          = context.WithCancel(context.Background())
		serverTpt, clientTpt = mcp.NewInMemoryTransports()
		client               = mcp.NewClient(&mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		}, nil)
	)
	defer cancel()

	executor := mock.NewExecutor()
	executor.ExecuteFn = func(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
		// Simulate func binary returning an error (e.g., function not found)
		return []byte("Error: function 'my-function' not found in namespace 'default'\n"), fmt.Errorf("exit status 1")
	}

	// Connect Server
	server := New(WithExecutor(executor))
	serverSession, err := server.Connect(ctx, serverTpt)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	// Connect Client
	clientSession, err := client.Connect(ctx, clientTpt, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	// Send Request
	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "delete",
		Arguments: map[string]any{
			"name": "my-function",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should return error from binary
	if !result.IsError {
		t.Fatal("expected error result when binary fails")
	}
	if !executor.ExecuteInvoked {
		t.Fatal("executor should have been invoked")
	}

	// Error should include binary output
	errMsg := mcpError(result)
	if !strings.Contains(errMsg, "not found") {
		t.Errorf("expected error to include binary output, got: %s", errMsg)
	}
}
