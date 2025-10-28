package mcp

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"knative.dev/func/pkg/mcp/mock"
)

// TestNew ensures that the MCP server can be instantiated with custom options
func TestNew(t *testing.T) {
	// Define a mock executor
	executor := mock.NewExecutor()
	executor.ExecuteFn = func(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
		return []byte("success"), nil
	}

	// Instantiate the MCP Server with mock executor and custom prefix
	server := New(WithPrefix("kn func"), WithExecutor(executor))

	// Verify server was created
	if server == nil {
		t.Fatal("expected server to be non-nil")
	}

	// Verify prefix was set correctly
	if server.prefix != "kn func" {
		t.Errorf("expected prefix 'kn func', got '%s'", server.prefix)
	}

	// Verify executor was set
	if server.executor != executor {
		t.Error("expected executor to be the mock")
	}
}

// TestStart ensures the server starts and a client can retrieve server metadata
func TestStart(t *testing.T) {
	var (
		ctx, cancel          = context.WithCancel(context.Background())
		serverTpt, clientTpt = mcp.NewInMemoryTransports()
		client               = mcp.NewClient(&mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		}, nil)
	)
	defer cancel()

	// Create server with mock executor
	executor := mock.NewExecutor()
	server := New(WithExecutor(executor))

	// Connect Server
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

	// Verify initialization result contains server info
	initResult := clientSession.InitializeResult()
	if initResult == nil {
		t.Fatal("expected non-nil initialization result")
	}
	if initResult.ServerInfo.Name != "func-mcp" {
		t.Errorf("expected server name 'func-mcp', got %q", initResult.ServerInfo.Name)
	}
	if initResult.ServerInfo.Version != "1.0.0" {
		t.Errorf("expected server version '1.0.0', got %q", initResult.ServerInfo.Version)
	}

	// List tools - should have all 9 registered tools
	toolsResult, err := clientSession.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("failed to list tools: %v", err)
	}
	if len(toolsResult.Tools) != 9 {
		t.Errorf("expected 9 tools, got %d", len(toolsResult.Tools))
	}

	// Verify expected tool names are present
	expectedTools := map[string]bool{
		"healthcheck":    false,
		"create":         false,
		"build":          false,
		"deploy":         false,
		"delete":         false,
		"list":           false,
		"config_volumes": false,
		"config_labels":  false,
		"config_envs":    false,
	}
	for _, tool := range toolsResult.Tools {
		if _, ok := expectedTools[tool.Name]; ok {
			expectedTools[tool.Name] = true
		}
	}
	for name, found := range expectedTools {
		if !found {
			t.Errorf("expected tool %q not found in tools list", name)
		}
	}

	// List resources - should have all 14 registered resources
	resourcesResult, err := clientSession.ListResources(ctx, &mcp.ListResourcesParams{})
	if err != nil {
		t.Fatalf("failed to list resources: %v", err)
	}
	if len(resourcesResult.Resources) != 14 {
		t.Errorf("expected 14 resources, got %d", len(resourcesResult.Resources))
	}

	// Verify new resource URIs are present
	expectedURIs := map[string]bool{
		"function://current":                      false,
		"function://help/root":                    false,
		"function://help/create":                  false,
		"function://help/build":                   false,
		"function://help/deploy":                  false,
		"function://help/list":                    false,
		"function://help/delete":                  false,
		"function://help/config/volumes/add":      false,
		"function://help/config/volumes/remove":   false,
		"function://help/config/labels/add":       false,
		"function://help/config/labels/remove":    false,
		"function://help/config/envs/add":         false,
		"function://help/config/envs/remove":      false,
		"function://templates":                    false,
	}
	for _, resource := range resourcesResult.Resources {
		if _, ok := expectedURIs[resource.URI]; ok {
			expectedURIs[resource.URI] = true
		}
	}
	for uri, found := range expectedURIs {
		if !found {
			t.Errorf("expected resource URI %q not found in resources list", uri)
		}
	}

	// List prompts - should have the workflow prompt
	promptsResult, err := clientSession.ListPrompts(ctx, &mcp.ListPromptsParams{})
	if err != nil {
		t.Fatalf("failed to list prompts: %v", err)
	}
	if len(promptsResult.Prompts) != 1 {
		t.Errorf("expected 1 prompt, got %d", len(promptsResult.Prompts))
	}

	// Verify the workflow prompt is registered
	if len(promptsResult.Prompts) > 0 && promptsResult.Prompts[0].Name != "create_function" {
		t.Errorf("expected prompt name 'create_function', got %q", promptsResult.Prompts[0].Name)
	}
}
