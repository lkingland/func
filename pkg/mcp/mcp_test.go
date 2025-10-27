package mcp

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	mcpmock "knative.dev/func/pkg/mcp/mock"
)

// TestCreate demonstrates the executor pattern for testing tools without requiring
// the actual func binary to be available.
func TestCreate(t *testing.T) {
	// Define a mock executor inline for this specific test
	mock := mcpmock.NewExecutor()
	mock.ExecuteFn = func(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
		// Assertions specific to the "create" command
		if dir != "/test/dir" {
			t.Errorf("expected dir '/test/dir', got '%s'", dir)
		}
		if name != "func" {
			t.Errorf("expected command 'func', got '%s'", name)
		}

		// Verify the command arguments match what we expect
		expectedArgs := []string{"create", "-l", "go", "--template", "http", "my-function"}
		if len(args) != len(expectedArgs) {
			t.Errorf("expected %d args, got %d: %v", len(expectedArgs), len(args), args)
			return nil, nil
		}
		for i, expected := range expectedArgs {
			if args[i] != expected {
				t.Errorf("arg[%d]: expected '%s', got '%s'", i, expected, args[i])
			}
		}

		// Return a mock success response
		return []byte("Function created successfully"), nil
	}

	// Instantiate the MCP Server with mock executor and default prefix
	_ = New(WithPrefix("func"), WithExecutor(mock))

	// Create a mock MCP request
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"cwd":      "/test/dir",
		"name":     "my-function",
		"language": "go",
		"template": "http",
	}

	// Create the tool and invoke its handler
	tool := create{}
	ctx := context.Background()
	result, err := tool.handle(ctx, request, "func", mock)

	// Verify the tool executed without error
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify the mock was called
	if !mock.ExecuteInvoked {
		t.Error("expected Execute to be called")
	}

	// Verify the result contains the expected output
	if result == nil {
		t.Fatal("expected result to be non-nil")
	}

	// Additional result validation could go here
	// For example, checking the result text contains expected content
}
