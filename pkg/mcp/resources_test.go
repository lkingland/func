package mcp

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	fn "knative.dev/func/pkg/functions"
	"knative.dev/func/pkg/mcp/mock"
)

func TestCurrentFunctionResource(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, func())
		validate func(t *testing.T, result *mcp.ReadResourceResult)
	}{
		{
			name: "valid function",
			setup: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				originalDir, _ := os.Getwd()

				// Create a function using the functions Client API
				client := fn.New()
				_, err := client.Init(fn.Function{
					Name:     "my-function",
					Runtime:  "go",
					Registry: "quay.io/user",
					Root:     dir,
				})
				if err != nil {
					t.Fatal(err)
				}

				// Change to temp directory so currentFunctionResource can find it
				if err := os.Chdir(dir); err != nil {
					t.Fatal(err)
				}

				cleanup := func() {
					_ = os.Chdir(originalDir)
				}

				return dir, cleanup
			},
			validate: func(t *testing.T, result *mcp.ReadResourceResult) {
				if len(result.Contents) != 1 {
					t.Fatalf("expected 1 content, got %d", len(result.Contents))
				}

				content := result.Contents[0]
				if content.URI != "func://current" {
					t.Errorf("expected URI 'func://current', got %q", content.URI)
				}
				if content.MIMEType != "application/json" {
					t.Errorf("expected MIMEType 'application/json', got %q", content.MIMEType)
				}

				// Parse the JSON to verify structure
				var f fn.Function
				if err := json.Unmarshal([]byte(content.Text), &f); err != nil {
					t.Fatalf("failed to parse JSON: %v", err)
				}

				if f.Name != "my-function" {
					t.Errorf("expected name 'my-function', got %q", f.Name)
				}
				if f.Runtime != "go" {
					t.Errorf("expected runtime 'go', got %q", f.Runtime)
				}
				if f.Registry != "quay.io/user" {
					t.Errorf("expected registry 'quay.io/user', got %q", f.Registry)
				}
			},
		},
		{
			name: "no function exists",
			setup: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				originalDir, _ := os.Getwd()

				// Change to empty temp directory (no function initialized)
				if err := os.Chdir(dir); err != nil {
					t.Fatal(err)
				}

				cleanup := func() {
					_ = os.Chdir(originalDir)
				}

				return dir, cleanup
			},
			validate: func(t *testing.T, result *mcp.ReadResourceResult) {
				if len(result.Contents) != 1 {
					t.Fatalf("expected 1 content, got %d", len(result.Contents))
				}

				content := result.Contents[0]
				if content.URI != "func://current" {
					t.Errorf("expected URI 'func://current', got %q", content.URI)
				}
				if content.MIMEType != "text/plain" {
					t.Errorf("expected MIMEType 'text/plain' for error, got %q", content.MIMEType)
				}

				// Should get friendly error message when no function exists
				text := strings.ToLower(content.Text)
				if !strings.Contains(text, "no function") || !strings.Contains(text, "current directory") {
					t.Errorf("expected error message about missing function, got: %s", content.Text)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, cleanup := tt.setup(t)
			defer cleanup()

			// Create resource and test handler
			resource := currentFunctionResource{}
			executor := &binaryExecutor{} // Use real executor (doesn't execute commands for this resource)
			handler := withResourcePrefix("", executor, resource.handle)

			result, err := handler(context.Background(), &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "func://current",
				},
			})

			if err != nil {
				t.Fatalf("handler returned unexpected error: %v", err)
			}

			tt.validate(t, result)

			// Verify we can still access the temp dir
			if _, err := os.Stat(dir); err != nil {
				t.Errorf("temp dir %q should still exist: %v", dir, err)
			}
		})
	}
}

func TestCurrentFunctionResourceDescriptor(t *testing.T) {
	resource := currentFunctionResource{}
	desc := resource.desc()

	if desc.URI != "func://current" {
		t.Errorf("expected URI 'func://current', got %q", desc.URI)
	}
	if desc.Name != "Current Function" {
		t.Errorf("expected Name 'Current Function', got %q", desc.Name)
	}
	if desc.Description != "Current function configuration from working directory" {
		t.Errorf("unexpected description: %s", desc.Description)
	}
	if desc.MIMEType != "application/json" {
		t.Errorf("expected MIMEType 'application/json', got %q", desc.MIMEType)
	}
}

func TestLanguagesResource(t *testing.T) {
	resource := languagesResource{}
	executor := mock.NewExecutor()
	executor.ExecuteFn = func(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
		return []byte("go\nnode\npython\nquarkus\nrust\nspringboot\ntypescript\n"), nil
	}
	handler := withResourcePrefix("func", executor, resource.handle)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "func://languages",
		},
	})

	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}

	if len(result.Contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(result.Contents))
	}

	content := result.Contents[0]
	if content.URI != "func://languages" {
		t.Errorf("expected URI 'func://languages', got %q", content.URI)
	}
	if content.MIMEType != "text/plain" {
		t.Errorf("expected MIMEType 'text/plain', got %q", content.MIMEType)
	}

	// Verify content includes expected languages
	expectedLanguages := []string{"go", "node", "python", "quarkus", "rust", "springboot", "typescript"}
	for _, lang := range expectedLanguages {
		if !strings.Contains(content.Text, lang) {
			t.Errorf("expected languages list to contain %q", lang)
		}
	}
}

func TestLanguagesResourceDescriptor(t *testing.T) {
	resource := languagesResource{}
	desc := resource.desc()

	if desc.URI != "func://languages" {
		t.Errorf("expected URI 'func://languages', got %q", desc.URI)
	}
	if desc.Name != "Available Languages" {
		t.Errorf("expected Name 'Available Languages', got %q", desc.Name)
	}
	if desc.Description != "List of available language runtimes" {
		t.Errorf("unexpected description: %s", desc.Description)
	}
	if desc.MIMEType != "text/plain" {
		t.Errorf("expected MIMEType 'text/plain', got %q", desc.MIMEType)
	}
}
