package mcp

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"knative.dev/func/pkg/mcp/mock"
)

// TestCreate ensures the create tool executes correctly when handed
// valid minimal arguments
func TestCreate(t *testing.T) {
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
		if len(args) < 4 {
			t.Fatalf("expected at least 4 args, got %d: %v", len(args), args)
		}
		if args[0] != "create" {
			t.Fatalf("expected args[0]='create', got %q", args[0])
		}
		if args[1] != "-l" || args[2] != "go" {
			t.Fatalf("expected args[1:3]=['-l', 'go'], got %v", args[1:3])
		}
		if args[len(args)-1] != "my-function" {
			t.Fatalf("expected last arg='my-function', got %q", args[len(args)-1])
		}

		return []byte("OK\n"), nil
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
		Name: "create",
		Arguments: map[string]any{
			"path":     path,
			"name":     "my-function",
			"language": "go",
		},
	})
	if err != nil { // request failed to process
		t.Fatal(err)
	}
	if result.IsError { // process resulted in failure
		t.Fatalf("expected success, got error: %s", mcpError(result))
	}

	// Assert executor invoked
	if !executor.ExecuteInvoked { // executor never invoked
		t.Fatal("executor was not invoked - create tool should have been called")
	}
}

// TestCreate_Options ensures the create tool correctly passes all optional
// flags to the func binary
func TestCreate_Options(t *testing.T) {
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
		if len(args) != 10 {
			t.Fatalf("expected 10 args, got %d: %v", len(args), args)
		}
		if args[0] != "create" {
			t.Fatalf("expected args[0]='create', got %q", args[0])
		}
		if args[1] != "-l" || args[2] != "go" {
			t.Fatalf("expected '-l go', got %q %q", args[1], args[2])
		}
		if args[3] != "--template" || args[4] != "cloudevents" {
			t.Fatalf("expected '--template cloudevents', got %q %q", args[3], args[4])
		}
		if args[5] != "--repository" || args[6] != "https://example.com/repo" {
			t.Fatalf("expected '--repository https://example.com/repo', got %q %q", args[5], args[6])
		}
		if args[7] != "--confirm" {
			t.Fatalf("expected args[7]='--confirm', got %q", args[7])
		}
		if args[8] != "--verbose" {
			t.Fatalf("expected args[8]='--verbose', got %q", args[8])
		}
		if args[9] != "test-func" {
			t.Fatalf("expected args[9]='test-func', got %q", args[9])
		}

		return []byte("OK\n"), nil
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
		Name: "create",
		Arguments: map[string]any{
			"path":       path,
			"name":       "test-func",
			"language":   "go",
			"template":   "cloudevents",
			"repository": "https://example.com/repo",
			"confirm":    true,
			"verbose":    true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify result is not an error
	if result.IsError {
		t.Fatalf("expected success, got error: %s", mcpError(result))
	}

	// Assert executor invoked
	if !executor.ExecuteInvoked {
		t.Fatal("executor was not invoked")
	}
}

// TestCreate_PathValidation ensures the tool validates path existence before invoking binary
func TestCreate_PathValidation(t *testing.T) {
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
		Name: "create",
		Arguments: map[string]any{
			"path":     "/nonexistent/path/that/should/not/exist",
			"name":     "my-function",
			"language": "go",
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

// TestCreate_BinaryFailure ensures errors from the func binary are returned as MCP errors
func TestCreate_BinaryFailure(t *testing.T) {
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
		// Simulate func binary returning an error (e.g., name conflict, invalid template, etc.)
		return []byte("Error: function 'my-function' already exists\n"), fmt.Errorf("exit status 1")
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
		Name: "create",
		Arguments: map[string]any{
			"path":     path,
			"name":     "my-function",
			"language": "go",
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
	if !strings.Contains(errMsg, "already exists") {
		t.Errorf("expected error to include binary output, got: %s", errMsg)
	}
}
