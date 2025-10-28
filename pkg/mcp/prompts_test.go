package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"knative.dev/func/pkg/mcp/mock"
)

func TestCreateFunctionPromptDescriptor(t *testing.T) {
	prompt := createFunctionPrompt{}
	desc := prompt.desc()

	if desc.Name != "create_function" {
		t.Errorf("expected name 'create_function', got %q", desc.Name)
	}

	if !strings.Contains(desc.Description, "Interactive") || !strings.Contains(desc.Description, "step-by-step") {
		t.Errorf("expected description to mention interactive step-by-step guidance, got %q", desc.Description)
	}

	// Check only 2 arguments are defined
	expectedArgs := map[string]bool{
		"language": false,
		"template": false,
	}

	if len(desc.Arguments) != len(expectedArgs) {
		t.Fatalf("expected %d arguments, got %d", len(expectedArgs), len(desc.Arguments))
	}

	for _, arg := range desc.Arguments {
		if _, exists := expectedArgs[arg.Name]; !exists {
			t.Errorf("unexpected argument: %s", arg.Name)
		}
		expectedArgs[arg.Name] = true
		if arg.Required {
			t.Errorf("argument %s should be optional, but is marked required", arg.Name)
		}
	}

	for name, found := range expectedArgs {
		if !found {
			t.Errorf("missing expected argument: %s", name)
		}
	}
}

func TestCreateFunctionPromptWithDefaults(t *testing.T) {
	prompt := createFunctionPrompt{}
	executor := mock.NewExecutor()
	executor.ExecuteFn = func(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
		return []byte("go\nnode\npython\nquarkus\nrust\nspringboot\ntypescript\n"), nil
	}
	handler := withPromptPrefix("func", executor, prompt.handle)

	// Skip fetching templates by providing both language and template
	// This avoids slow HTTP calls to GitHub in tests
	result, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Arguments: map[string]string{
				"language": "go",
				"template": "http",
			},
		},
	})

	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}

	if result.Description == "" {
		t.Error("expected non-empty description")
	}

	if len(result.Messages) == 0 {
		t.Fatal("expected messages in result")
	}

	// Verify key workflow elements are present
	var allText strings.Builder
	for _, msg := range result.Messages {
		if textContent, ok := msg.Content.(*mcp.TextContent); ok {
			allText.WriteString(textContent.Text)
			allText.WriteString(" ")
		}
	}

	fullText := strings.ToLower(allText.String())
	if !strings.Contains(fullText, "action") {
		t.Error("expected conversation to contain ACTION directive")
	}
	if !strings.Contains(fullText, "create") {
		t.Error("expected conversation to mention creating function")
	}
	if !strings.Contains(fullText, "deploy") {
		t.Error("expected conversation to mention deployment as next step")
	}
}

func TestCreateFunctionPromptWithFullArguments(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]string
		expectInText []string
	}{
		{
			name: "go language with http template",
			args: map[string]string{
				"language": "go",
				"template": "http",
			},
			expectInText: []string{
				"ACTION",
				"create",
			},
		},
		{
			name: "python with http template",
			args: map[string]string{
				"language": "python",
				"template": "http",
			},
			expectInText: []string{
				"ACTION",
				"create",
			},
		},
		{
			name: "node with cloudevents template",
			args: map[string]string{
				"language": "node",
				"template": "cloudevents",
			},
			expectInText: []string{
				"ACTION",
				"create",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := createFunctionPrompt{}
			executor := mock.NewExecutor()
			executor.ExecuteFn = func(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
				return []byte("go\nnode\npython\nquarkus\nrust\nspringboot\ntypescript\n"), nil
			}
			handler := withPromptPrefix("func", executor, prompt.handle)

			result, err := handler(context.Background(), &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Arguments: tt.args,
				},
			})

			if err != nil {
				t.Fatalf("handler returned unexpected error: %v", err)
			}

			// Concatenate all text content to search
			var allText strings.Builder
			for _, msg := range result.Messages {
				if textContent, ok := msg.Content.(*mcp.TextContent); ok {
					allText.WriteString(textContent.Text)
					allText.WriteString(" ")
				}
			}

			fullText := allText.String()

			for _, expected := range tt.expectInText {
				if !strings.Contains(fullText, expected) {
					t.Errorf("expected conversation to contain %q, but it was not found", expected)
				}
			}
		})
	}
}

func TestCreateFunctionPromptConversationFlow(t *testing.T) {
	prompt := createFunctionPrompt{}
	executor := mock.NewExecutor()
	executor.ExecuteFn = func(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
		return []byte("go\nnode\npython\nquarkus\nrust\nspringboot\ntypescript\n"), nil
	}
	handler := withPromptPrefix("func", executor, prompt.handle)

	// Provide both args to avoid slow HTTP calls to GitHub in tests
	result, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Arguments: map[string]string{
				"language": "go",
				"template": "http",
			},
		},
	})

	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}

	// Verify the conversation has multiple messages
	if len(result.Messages) < 3 {
		t.Errorf("expected at least 3 messages for workflow, got %d", len(result.Messages))
	}

	// Verify key topics for create workflow are covered
	topics := []string{
		"directory",
		"language",
		"create",
	}

	var allText strings.Builder
	for _, msg := range result.Messages {
		if textContent, ok := msg.Content.(*mcp.TextContent); ok {
			allText.WriteString(textContent.Text)
			allText.WriteString(" ")
		}
	}

	fullText := strings.ToLower(allText.String())
	for _, topic := range topics {
		if !strings.Contains(fullText, topic) {
			t.Errorf("expected conversation to cover topic %q", topic)
		}
	}
}

func TestGetArgOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]string
		key          string
		defaultValue string
		expected     string
	}{
		{
			name:         "existing key",
			args:         map[string]string{"foo": "bar"},
			key:          "foo",
			defaultValue: "default",
			expected:     "bar",
		},
		{
			name:         "missing key",
			args:         map[string]string{"foo": "bar"},
			key:          "baz",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "empty value",
			args:         map[string]string{"foo": ""},
			key:          "foo",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "nil map",
			args:         nil,
			key:          "foo",
			defaultValue: "default",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getArgOrDefault(tt.args, tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDeployFunctionPromptDescriptor(t *testing.T) {
	prompt := deployFunctionPrompt{}
	desc := prompt.desc()

	if desc.Name != "deploy_function" {
		t.Errorf("expected name 'deploy_function', got %q", desc.Name)
	}

	if !strings.Contains(desc.Description, "Kubernetes") {
		t.Errorf("expected description to mention 'Kubernetes', got %q", desc.Description)
	}

	// Check all 4 arguments are defined (image was removed in simplification)
	expectedArgs := map[string]bool{
		"language":      false,
		"registry":      false,
		"image":         false,
		"builder":       false,
		"k8s_namespace": false,
	}

	if len(desc.Arguments) != len(expectedArgs) {
		t.Fatalf("expected %d arguments, got %d", len(expectedArgs), len(desc.Arguments))
	}

	for _, arg := range desc.Arguments {
		if _, exists := expectedArgs[arg.Name]; !exists {
			t.Errorf("unexpected argument: %s", arg.Name)
		}
		expectedArgs[arg.Name] = true
		if arg.Required {
			t.Errorf("argument %s should be optional, but is marked required", arg.Name)
		}
	}

	for name, found := range expectedArgs {
		if !found {
			t.Errorf("missing expected argument: %s", name)
		}
	}
}

func TestDeployFunctionPromptWithDefaults(t *testing.T) {
	prompt := deployFunctionPrompt{}
	executor := mock.NewExecutor()
	handler := withPromptPrefix("func", executor, prompt.handle)

	result, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Arguments: map[string]string{},
		},
	})

	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}

	if result.Description == "" {
		t.Error("expected non-empty description")
	}

	if len(result.Messages) == 0 {
		t.Fatal("expected messages in result")
	}

	// Verify conversation structure includes cluster verification
	var allText strings.Builder
	for _, msg := range result.Messages {
		if textContent, ok := msg.Content.(*mcp.TextContent); ok {
			allText.WriteString(textContent.Text)
			allText.WriteString(" ")
		}
	}

	fullText := strings.ToLower(allText.String())
	if !strings.Contains(fullText, "kubectl") {
		t.Error("expected conversation to mention kubectl for cluster verification")
	}
	if !strings.Contains(fullText, "registry") {
		t.Error("expected conversation to ask about registry")
	}
	if !strings.Contains(fullText, "action") {
		t.Error("expected conversation to contain ACTION directive")
	}
}

func TestDeployFunctionPromptWithFullArguments(t *testing.T) {
	tests := []struct {
		name              string
		args              map[string]string
		expectInText      []string
		builderChoice     string
		expectImageFormat string
	}{
		{
			name: "go with host builder and registry",
			args: map[string]string{
				"language": "go",
				"registry": "docker.io/alice",
			},
			expectInText: []string{
				"docker.io/alice",
				"host",
			},
			builderChoice:     "host",
			expectImageFormat: "docker.io/alice",
		},
		{
			name: "python with host builder and ghcr",
			args: map[string]string{
				"language": "python",
				"registry": "ghcr.io/bob",
			},
			expectInText: []string{
				"ghcr.io/bob",
				"host",
			},
			builderChoice:     "host",
			expectImageFormat: "ghcr.io/bob",
		},
		{
			name: "node with pack builder and quay",
			args: map[string]string{
				"language": "node",
				"registry": "quay.io/charlie",
			},
			expectInText: []string{
				"quay.io/charlie",
				"pack",
			},
			builderChoice:     "pack",
			expectImageFormat: "quay.io/charlie",
		},
		{
			name: "complete image override",
			args: map[string]string{
				"image": "registry.example.com/org/custom-func:v1.2.3",
			},
			expectInText: []string{
				"registry.example.com/org/custom-func:v1.2.3",
			},
			expectImageFormat: "registry.example.com/org/custom-func:v1.2.3",
		},
		{
			name: "with k8s namespace",
			args: map[string]string{
				"language":      "go",
				"registry":      "docker.io/alice",
				"k8s_namespace": "production",
			},
			expectInText: []string{
				"production",
				"namespace: production",
			},
			builderChoice: "host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := deployFunctionPrompt{}
			executor := mock.NewExecutor()
			handler := withPromptPrefix("func", executor, prompt.handle)

			result, err := handler(context.Background(), &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Arguments: tt.args,
				},
			})

			if err != nil {
				t.Fatalf("handler returned unexpected error: %v", err)
			}

			// Concatenate all text content to search
			var allText strings.Builder
			for _, msg := range result.Messages {
				if textContent, ok := msg.Content.(*mcp.TextContent); ok {
					allText.WriteString(textContent.Text)
					allText.WriteString(" ")
				}
			}

			fullText := allText.String()

			for _, expected := range tt.expectInText {
				if !strings.Contains(fullText, expected) {
					t.Errorf("expected conversation to contain %q, but it was not found", expected)
				}
			}

			// Verify builder choice is mentioned if specified
			if tt.builderChoice != "" {
				if !strings.Contains(fullText, tt.builderChoice) {
					t.Errorf("expected builder %q to be mentioned in conversation", tt.builderChoice)
				}
			}
		})
	}
}

func TestDeployFunctionPromptBuilderDefaults(t *testing.T) {
	tests := []struct {
		language        string
		expectedBuilder string
	}{
		{"go", "host"},
		{"python", "host"},
		{"node", "pack"},
		{"typescript", "pack"},
		{"rust", "pack"},
		{"java", "pack"},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			prompt := deployFunctionPrompt{}
			executor := mock.NewExecutor()
			handler := withPromptPrefix("func", executor, prompt.handle)

			result, err := handler(context.Background(), &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Arguments: map[string]string{
						"language": tt.language,
					},
				},
			})

			if err != nil {
				t.Fatalf("handler returned unexpected error: %v", err)
			}

			// Check that the expected builder is mentioned
			var allText strings.Builder
			for _, msg := range result.Messages {
				if textContent, ok := msg.Content.(*mcp.TextContent); ok {
					allText.WriteString(textContent.Text)
					allText.WriteString(" ")
				}
			}

			fullText := strings.ToLower(allText.String())
			if !strings.Contains(fullText, tt.expectedBuilder) {
				t.Errorf("expected builder %q to be mentioned for language %q", tt.expectedBuilder, tt.language)
			}
		})
	}
}

func TestDeployFunctionPromptConversationFlow(t *testing.T) {
	prompt := deployFunctionPrompt{}
	executor := mock.NewExecutor()
	handler := withPromptPrefix("func", executor, prompt.handle)

	result, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Arguments: map[string]string{
				"language": "go",
				"registry": "docker.io/alice",
			},
		},
	})

	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}

	// Verify the conversation has a good flow
	if len(result.Messages) < 4 {
		t.Errorf("expected at least 4 messages for deploy workflow, got %d", len(result.Messages))
	}

	// Verify key topics for deploy workflow are covered
	topics := []string{
		"kubernetes",
		"registry",
		"builder",
		"deploy",
	}

	var allText strings.Builder
	for _, msg := range result.Messages {
		if textContent, ok := msg.Content.(*mcp.TextContent); ok {
			allText.WriteString(textContent.Text)
			allText.WriteString(" ")
		}
	}

	fullText := strings.ToLower(allText.String())
	for _, topic := range topics {
		if !strings.Contains(fullText, topic) {
			t.Errorf("expected conversation to cover topic %q", topic)
		}
	}
}
