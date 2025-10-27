package mcp

import (
	"context"
	"fmt"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"knative.dev/func/pkg/mcp/mock"
)

// TestList ensures the list tool executes correctly with no arguments
func TestList(t *testing.T) {
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
		if len(args) != 1 {
			t.Fatalf("expected 1 arg, got %d: %v", len(args), args)
		}
		if args[0] != "list" {
			t.Fatalf("expected args[0]='list', got %q", args[0])
		}

		return []byte("NAME          NAMESPACE  RUNTIME  URL\nmy-function   default    go       https://my-function.example.com\n"), nil
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

	// Send Request (no arguments)
	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list",
		Arguments: map[string]any{},
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

// TestList_Options ensures the list tool correctly passes all optional flags
func TestList_Options(t *testing.T) {
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
		// Expected: ["list", "--namespace", "prod", "--output", "json"]
		if len(args) != 5 {
			t.Fatalf("expected 5 args, got %d: %v", len(args), args)
		}
		if args[0] != "list" {
			t.Fatalf("expected args[0]='list', got %q", args[0])
		}
		if args[1] != "--namespace" || args[2] != "prod" {
			t.Fatalf("expected '--namespace prod', got %q %q", args[1], args[2])
		}
		if args[3] != "--output" || args[4] != "json" {
			t.Fatalf("expected '--output json', got %q %q", args[3], args[4])
		}

		return []byte(`[{"name":"my-function","namespace":"prod"}]`), nil
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
		Name: "list",
		Arguments: map[string]any{
			"namespace": "prod",
			"output":    "json",
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

// TestList_BinaryFailure ensures errors from the func binary are returned as MCP errors
func TestList_BinaryFailure(t *testing.T) {
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
		// Simulate func binary returning an error (e.g., not connected to cluster)
		return []byte("Error: cannot connect to Kubernetes cluster\n"), fmt.Errorf("exit status 1")
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
		Name:      "list",
		Arguments: map[string]any{},
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
}
