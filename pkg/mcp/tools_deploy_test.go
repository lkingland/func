package mcp

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"knative.dev/func/pkg/mcp/mock"
)

// TestDeploy ensures the deploy tool executes correctly with minimal arguments
func TestDeploy(t *testing.T) {
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
		if dir != "." {
			t.Fatalf("expected dir %q, got %q", ".", dir)
		}
		if name != "func" {
			t.Fatalf("expected command 'func', got %q", name)
		}

		// Validate args (order is deterministic)
		if len(args) != 1 {
			t.Fatalf("expected 1 arg, got %d: %v", len(args), args)
		}
		if args[0] != "deploy" {
			t.Fatalf("expected args[0]='deploy', got %q", args[0])
		}

		return []byte("Function deployed: https://my-function.example.com\n"), nil
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
		Name:      "deploy",
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

// TestDeploy_Options ensures the deploy tool correctly passes all optional flags
func TestDeploy_Options(t *testing.T) {
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
		if dir != "." {
			t.Fatalf("expected dir %q, got %q", ".", dir)
		}
		if name != "func" {
			t.Fatalf("expected command 'func', got %q", name)
		}

		// Validate args (order is deterministic)
		// Expected: ["deploy", "--namespace", "prod", "--registry", "ghcr.io/user", "--platform", "linux/amd64", "--push", "--remote"]
		if len(args) != 9 {
			t.Fatalf("expected 9 args, got %d: %v", len(args), args)
		}
		if args[0] != "deploy" {
			t.Fatalf("expected args[0]='deploy', got %q", args[0])
		}
		if args[1] != "--registry" || args[2] != "ghcr.io/user" {
			t.Fatalf("expected '--registry ghcr.io/user', got %q %q", args[1], args[2])
		}
		if args[3] != "--namespace" || args[4] != "prod" {
			t.Fatalf("expected '--namespace prod', got %q %q", args[3], args[4])
		}
		if args[5] != "--platform" || args[6] != "linux/amd64" {
			t.Fatalf("expected '--platform linux/amd64', got %q %q", args[5], args[6])
		}
		if args[7] != "--push" {
			t.Fatalf("expected args[7]='--push', got %q", args[7])
		}
		if args[8] != "--remote" {
			t.Fatalf("expected args[8]='--remote', got %q", args[8])
		}

		return []byte("Function deployed to prod namespace\n"), nil
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
		Name: "deploy",
		Arguments: map[string]any{
			"namespace": "prod",
			"registry":  "ghcr.io/user",
			"platform":  "linux/amd64",
			"push":      true,
			"remote":    true,
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

// TestDeploy_BinaryFailure ensures errors from the func binary are returned as MCP errors
func TestDeploy_BinaryFailure(t *testing.T) {
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
		// Simulate func binary returning an error (e.g., not connected to cluster, permission denied, etc.)
		return []byte("Error: no Kubernetes cluster found\n"), fmt.Errorf("exit status 1")
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
		Name:      "deploy",
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

	// Error should include binary output
	errMsg := mcpError(result)
	if !strings.Contains(errMsg, "cluster") {
		t.Errorf("expected error to include binary output, got: %s", errMsg)
	}
}
