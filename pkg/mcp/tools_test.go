package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"knative.dev/func/pkg/mcp/mock"
)

// TestMain sets up test environment - enables MCP feature for tests
func TestMain(m *testing.M) {
	// Enable MCP feature for all tests
	os.Setenv("FUNC_ENABLE_MCP", "true")
	code := m.Run()
	os.Exit(code)
}

// mcpError extracts error message from MCP result for cleaner test assertions.
// Used across all tool test files.
func mcpError(result *mcp.CallToolResult) string {
	if len(result.Content) > 0 {
		if tc, ok := result.Content[0].(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	return fmt.Sprintf("%v", result.Content)
}

// TestHelperFunctions validates the shared utility functions used by all tools
func TestHelperFunctions(t *testing.T) {
	// Test validatePathExists with non-existent path
	if err := validatePathExists("/nonexistent/path/that/should/not/exist"); err == nil {
		t.Error("validatePathExists should fail for non-existent path")
	}

	// Test validatePathExists with root (should exist on all systems)
	if err := validatePathExists("/"); err != nil {
		t.Errorf("validatePathExists failed for root: %v", err)
	}

	// Test jsonResult helper
	testData := map[string]string{"key": "value"}
	result := jsonResult(testData)
	if len(result.Content) == 0 {
		t.Fatal("jsonResult produced no content")
	}

	// Verify the JSON is valid
	var decoded map[string]string
	textContent := result.Content[0].(*mcp.TextContent)
	if err := json.Unmarshal([]byte(textContent.Text), &decoded); err != nil {
		t.Errorf("jsonResult produced invalid JSON: %v", err)
	}

	if decoded["key"] != "value" {
		t.Errorf("jsonResult data mismatch: got %v", decoded)
	}

	// Test appendStringFlag helper
	args := []string{"create"}
	value := "test"
	args = appendStringFlag(args, "--flag", &value)
	if len(args) != 3 || args[1] != "--flag" || args[2] != "test" {
		t.Errorf("appendStringFlag failed: got %v", args)
	}

	// Test appendStringFlag with nil value (should not append)
	args = []string{"create"}
	args = appendStringFlag(args, "--flag", nil)
	if len(args) != 1 {
		t.Errorf("appendStringFlag should not append nil value: got %v", args)
	}

	// Test appendBoolFlag helper
	args = []string{"create"}
	boolValue := true
	args = appendBoolFlag(args, "--verbose", &boolValue)
	if len(args) != 2 || args[1] != "--verbose" {
		t.Errorf("appendBoolFlag failed: got %v", args)
	}

	// Test appendBoolFlag with false (should not append)
	args = []string{"create"}
	falseValue := false
	args = appendBoolFlag(args, "--verbose", &falseValue)
	if len(args) != 1 {
		t.Errorf("appendBoolFlag should not append false value: got %v", args)
	}

	// Test parseCommand helper
	parts := parseCommand("kn func")
	if len(parts) != 2 || parts[0] != "kn" || parts[1] != "func" {
		t.Errorf("parseCommand failed: got %v", parts)
	}

	// Test errorResult helper
	errResult := errorResult("test error")
	if !errResult.IsError {
		t.Error("errorResult should set IsError to true")
	}
	if len(errResult.Content) == 0 {
		t.Fatal("errorResult produced no content")
	}
	errText := errResult.Content[0].(*mcp.TextContent)
	if !strings.Contains(errText.Text, "test error") {
		t.Errorf("errorResult text mismatch: got %s", errText.Text)
	}
}

// TestToolGating verifies that tools are properly gated when FUNC_ENABLE_MCP is not set
func TestToolGating(t *testing.T) {
	// Save current env var state
	originalValue := os.Getenv("FUNC_ENABLE_MCP")
	defer os.Setenv("FUNC_ENABLE_MCP", originalValue)

	// Unset the env var to test gating
	os.Unsetenv("FUNC_ENABLE_MCP")

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
		t.Fatal("executor should not be called when env var is not set")
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

	// Attempt to call a tool - should be gated
	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "create",
		Arguments: map[string]any{
			"path":     path,
			"name":     "my-function",
			"language": "go",
		},
	})

	if err != nil {
		t.Fatalf("CallTool returned unexpected error: %v", err)
	}

	// Tool should return an error result
	if !result.IsError {
		t.Fatal("expected tool to be gated with IsError=true")
	}

	// Error message should be directive and concise
	errorMsg := strings.ToLower(mcpError(result))
	if !strings.Contains(errorMsg, "func_enable_mcp") {
		t.Errorf("expected error message to mention FUNC_ENABLE_MCP, got: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "disabled") {
		t.Errorf("expected error message to mention tools are disabled, got: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "close") || !strings.Contains(errorMsg, "exit") {
		t.Errorf("expected error message to tell user to close/exit client, got: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "do not") {
		t.Errorf("expected error message to explicitly forbid workarounds, got: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "setup instructions") || !strings.Contains(errorMsg, "github.com") {
		t.Errorf("expected error message to reference setup instructions URL, got: %s", errorMsg)
	}
}
