package mcp

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"knative.dev/func/pkg/mcp/mock"
)

// TestConfigVolumes_List ensures the config_volumes tool can list volumes
func TestConfigVolumes_List(t *testing.T) {
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

		// Validate args for list action
		if len(args) != 2 {
			t.Fatalf("expected 2 args, got %d: %v", len(args), args)
		}
		if args[0] != "config" || args[1] != "volumes" {
			t.Fatalf("expected 'config volumes', got %q %q", args[0], args[1])
		}

		return []byte("secret:my-secret:/workspace/secret\n"), nil
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
		Name: "config_volumes",
		Arguments: map[string]any{
			"action": "list",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", mcpError(result))
	}

	if !executor.ExecuteInvoked {
		t.Fatal("executor was not invoked")
	}
}

// TestConfigVolumes_Add ensures the config_volumes tool can add volumes
func TestConfigVolumes_Add(t *testing.T) {
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

		// Expected: ["config", "volumes", "add", "--type", "secret", "--mount-path", "/workspace/secret", "--source", "my-secret"]
		if len(args) != 9 {
			t.Fatalf("expected 9 args, got %d: %v", len(args), args)
		}
		if args[0] != "config" || args[1] != "volumes" || args[2] != "add" {
			t.Fatalf("expected 'config volumes add', got %q %q %q", args[0], args[1], args[2])
		}
		if args[3] != "--type" || args[4] != "secret" {
			t.Fatalf("expected '--type secret', got %q %q", args[3], args[4])
		}
		if args[5] != "--mount-path" || args[6] != "/workspace/secret" {
			t.Fatalf("expected '--mount-path /workspace/secret', got %q %q", args[5], args[6])
		}
		if args[7] != "--source" || args[8] != "my-secret" {
			t.Fatalf("expected '--source my-secret', got %q %q", args[7], args[8])
		}

		return []byte("Volume added successfully\n"), nil
	}

	server := New(WithExecutor(executor))
	serverSession, err := server.Connect(ctx, serverTpt)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	clientSession, err := client.Connect(ctx, clientTpt, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "config_volumes",
		Arguments: map[string]any{
			"action":    "add",
			"type":      "secret",
			"mountPath": "/workspace/secret",
			"source":    "my-secret",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", mcpError(result))
	}
}

// TestConfigVolumes_BinaryFailure ensures binary errors are propagated
func TestConfigVolumes_BinaryFailure(t *testing.T) {
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
		return []byte("Error: invalid volume configuration\n"), fmt.Errorf("exit status 1")
	}

	server := New(WithExecutor(executor))
	serverSession, err := server.Connect(ctx, serverTpt)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	clientSession, err := client.Connect(ctx, clientTpt, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "config_volumes",
		Arguments: map[string]any{
			"action": "list",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Fatal("expected error result when binary fails")
	}
	if !executor.ExecuteInvoked {
		t.Fatal("executor should have been invoked")
	}
}

// TestConfigLabels_List ensures the config_labels tool can list labels
func TestConfigLabels_List(t *testing.T) {
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

		if len(args) != 2 {
			t.Fatalf("expected 2 args, got %d: %v", len(args), args)
		}
		if args[0] != "config" || args[1] != "labels" {
			t.Fatalf("expected 'config labels', got %q %q", args[0], args[1])
		}

		return []byte("app=my-function\nenvironment=prod\n"), nil
	}

	server := New(WithExecutor(executor))
	serverSession, err := server.Connect(ctx, serverTpt)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	clientSession, err := client.Connect(ctx, clientTpt, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "config_labels",
		Arguments: map[string]any{
			"action": "list",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", mcpError(result))
	}
}

// TestConfigLabels_Add ensures the config_labels tool can add labels
func TestConfigLabels_Add(t *testing.T) {
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
		// Expected: ["config", "labels", "add", "--name", "environment", "--value", "prod"]
		if len(args) != 7 {
			t.Fatalf("expected 7 args, got %d: %v", len(args), args)
		}
		if args[0] != "config" || args[1] != "labels" || args[2] != "add" {
			t.Fatalf("expected 'config labels add', got %q %q %q", args[0], args[1], args[2])
		}
		if args[3] != "--name" || args[4] != "environment" {
			t.Fatalf("expected '--name environment', got %q %q", args[3], args[4])
		}
		if args[5] != "--value" || args[6] != "prod" {
			t.Fatalf("expected '--value prod', got %q %q", args[5], args[6])
		}

		return []byte("Label added successfully\n"), nil
	}

	server := New(WithExecutor(executor))
	serverSession, err := server.Connect(ctx, serverTpt)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	clientSession, err := client.Connect(ctx, clientTpt, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "config_labels",
		Arguments: map[string]any{
			"action": "add",
			"name":   "environment",
			"value":  "prod",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", mcpError(result))
	}
}

// TestConfigEnvs_List ensures the config_envs tool can list environment variables
func TestConfigEnvs_List(t *testing.T) {
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

		if len(args) != 2 {
			t.Fatalf("expected 2 args, got %d: %v", len(args), args)
		}
		if args[0] != "config" || args[1] != "envs" {
			t.Fatalf("expected 'config envs', got %q %q", args[0], args[1])
		}

		return []byte("DATABASE_URL=postgres://localhost\nAPI_KEY=secret\n"), nil
	}

	server := New(WithExecutor(executor))
	serverSession, err := server.Connect(ctx, serverTpt)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	clientSession, err := client.Connect(ctx, clientTpt, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "config_envs",
		Arguments: map[string]any{
			"action": "list",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", mcpError(result))
	}
}

// TestConfigEnvs_Add ensures the config_envs tool can add environment variables
func TestConfigEnvs_Add(t *testing.T) {
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
		// Expected: ["config", "envs", "add", "--name", "API_KEY", "--value", "secret123"]
		if len(args) != 7 {
			t.Fatalf("expected 7 args, got %d: %v", len(args), args)
		}
		if args[0] != "config" || args[1] != "envs" || args[2] != "add" {
			t.Fatalf("expected 'config envs add', got %q %q %q", args[0], args[1], args[2])
		}
		if args[3] != "--name" || args[4] != "API_KEY" {
			t.Fatalf("expected '--name API_KEY', got %q %q", args[3], args[4])
		}
		if args[5] != "--value" || args[6] != "secret123" {
			t.Fatalf("expected '--value secret123', got %q %q", args[5], args[6])
		}

		return []byte("Environment variable added successfully\n"), nil
	}

	server := New(WithExecutor(executor))
	serverSession, err := server.Connect(ctx, serverTpt)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	clientSession, err := client.Connect(ctx, clientTpt, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "config_envs",
		Arguments: map[string]any{
			"action": "add",
			"name":   "API_KEY",
			"value":  "secret123",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", mcpError(result))
	}
}

// TestConfigEnvs_BinaryFailure ensures binary errors are propagated
func TestConfigEnvs_BinaryFailure(t *testing.T) {
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
		return []byte("Error: no func.yaml found\n"), fmt.Errorf("exit status 1")
	}

	server := New(WithExecutor(executor))
	serverSession, err := server.Connect(ctx, serverTpt)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	clientSession, err := client.Connect(ctx, clientTpt, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "config_envs",
		Arguments: map[string]any{
			"action": "list",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Fatal("expected error result when binary fails")
	}
	if !executor.ExecuteInvoked {
		t.Fatal("executor should have been invoked")
	}

	errMsg := mcpError(result)
	if !strings.Contains(errMsg, "func.yaml") {
		t.Errorf("expected error to include binary output, got: %s", errMsg)
	}
}
