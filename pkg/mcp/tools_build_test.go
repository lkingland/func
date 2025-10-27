package mcp

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"knative.dev/func/pkg/mcp/mock"
)

// TestBuild ensures the build tool executes correctly with minimal arguments
func TestBuild(t *testing.T) {
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
		if len(args) != 1 {
			t.Fatalf("expected 1 arg, got %d: %v", len(args), args)
		}
		if args[0] != "build" {
			t.Fatalf("expected args[0]='build', got %q", args[0])
		}

		return []byte("Function image built: example.com/my-function:latest\n"), nil
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
		Name: "build",
		Arguments: map[string]any{
			"path": path,
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

// TestBuild_Options ensures the build tool correctly passes all optional flags
func TestBuild_Options(t *testing.T) {
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
		// Expected: ["build", "--builder", "pack", "--registry", "ghcr.io/user", "--platform", "linux/amd64", "--push", "--verbose"]
		if len(args) != 8 {
			t.Fatalf("expected 8 args, got %d: %v", len(args), args)
		}
		if args[0] != "build" {
			t.Fatalf("expected args[0]='build', got %q", args[0])
		}
		if args[1] != "--builder" || args[2] != "pack" {
			t.Fatalf("expected '--builder pack', got %q %q", args[1], args[2])
		}
		if args[3] != "--registry" || args[4] != "ghcr.io/user" {
			t.Fatalf("expected '--registry ghcr.io/user', got %q %q", args[3], args[4])
		}
		if args[5] != "--platform" || args[6] != "linux/amd64" {
			t.Fatalf("expected '--platform linux/amd64', got %q %q", args[5], args[6])
		}
		if args[7] != "--push" {
			t.Fatalf("expected args[7]='--push', got %q", args[7])
		}

		return []byte("Function image built and pushed\n"), nil
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
		Name: "build",
		Arguments: map[string]any{
			"path":     path,
			"builder":  "pack",
			"registry": "ghcr.io/user",
			"platform": "linux/amd64",
			"push":     true,
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

// TestBuild_PathValidation ensures the tool validates path existence before invoking binary
func TestBuild_PathValidation(t *testing.T) {
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
		Name: "build",
		Arguments: map[string]any{
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

// TestBuild_BinaryFailure ensures errors from the func binary are returned as MCP errors
func TestBuild_BinaryFailure(t *testing.T) {
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
		// Simulate func binary returning an error (e.g., missing func.yaml, build failure, etc.)
		return []byte("Error: no func.yaml found in directory\n"), fmt.Errorf("exit status 1")
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
		Name: "build",
		Arguments: map[string]any{
			"path": path,
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
	if !strings.Contains(errMsg, "func.yaml") {
		t.Errorf("expected error to include binary output, got: %s", errMsg)
	}
}
