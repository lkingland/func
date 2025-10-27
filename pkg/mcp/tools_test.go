package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
