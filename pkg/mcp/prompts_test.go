package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCreateFunctionPromptDescriptor(t *testing.T) {
	prompt := createFunctionPrompt{}
	desc := prompt.desc()

	if desc.Name != "create_function" {
		t.Errorf("expected name 'create_function', got %q", desc.Name)
	}

	if !strings.Contains(desc.Description, "Guide") {
		t.Errorf("expected description to mention 'Guide', got %q", desc.Description)
	}

	// Check all 5 arguments are defined
	expectedArgs := map[string]bool{
		"language":  false,
		"template":  false,
		"registry":  false,
		"namespace": false,
		"builder":   false,
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
	handler := withPromptPrefix("func", prompt.handle)

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

	// Verify conversation structure
	foundUserMessage := false
	foundAssistantMessage := false
	foundEmbeddedResource := false

	for _, msg := range result.Messages {
		if msg.Role == "user" {
			foundUserMessage = true
		}
		if msg.Role == "assistant" {
			foundAssistantMessage = true
			// Check for embedded resource
			if embedded, ok := msg.Content.(*mcp.EmbeddedResource); ok {
				if embedded.Resource.URI == "function://templates" {
					foundEmbeddedResource = true
				}
			}
		}
	}

	if !foundUserMessage {
		t.Error("expected at least one user message")
	}
	if !foundAssistantMessage {
		t.Error("expected at least one assistant message")
	}
	if !foundEmbeddedResource {
		t.Error("expected embedded templates resource in conversation")
	}
}

func TestCreateFunctionPromptWithFullArguments(t *testing.T) {
	tests := []struct {
		name          string
		args          map[string]string
		expectInText  []string
		builderChoice string
	}{
		{
			name: "go language with host builder",
			args: map[string]string{
				"language":  "go",
				"template":  "http",
				"registry":  "docker.io",
				"namespace": "alice",
			},
			expectInText: []string{
				"go",
				"http",
				"docker.io/alice",
				"host",
			},
			builderChoice: "host",
		},
		{
			name: "python language with host builder",
			args: map[string]string{
				"language":  "python",
				"template":  "http",
				"registry":  "ghcr.io",
				"namespace": "bob",
			},
			expectInText: []string{
				"python",
				"ghcr.io/bob",
				"host",
			},
			builderChoice: "host",
		},
		{
			name: "node language with pack builder",
			args: map[string]string{
				"language":  "node",
				"template":  "http",
				"registry":  "quay.io",
				"namespace": "charlie",
			},
			expectInText: []string{
				"node",
				"quay.io/charlie",
				"pack",
			},
			builderChoice: "pack",
		},
		{
			name: "explicit builder override",
			args: map[string]string{
				"language": "go",
				"builder":  "pack",
			},
			expectInText: []string{
				"pack",
			},
			builderChoice: "pack",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := createFunctionPrompt{}
			handler := withPromptPrefix("func", prompt.handle)

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

			// Verify builder choice is mentioned
			if !strings.Contains(fullText, tt.builderChoice) {
				t.Errorf("expected builder %q to be mentioned in conversation", tt.builderChoice)
			}
		})
	}
}

func TestCreateFunctionPromptBuilderDefaults(t *testing.T) {
	tests := []struct {
		language       string
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
			prompt := createFunctionPrompt{}
			handler := withPromptPrefix("func", prompt.handle)

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

func TestCreateFunctionPromptConversationFlow(t *testing.T) {
	prompt := createFunctionPrompt{}
	handler := withPromptPrefix("func", prompt.handle)

	result, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Arguments: map[string]string{
				"language":  "go",
				"template":  "http",
				"registry":  "docker.io",
				"namespace": "alice",
			},
		},
	})

	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}

	// Verify the conversation has a good flow with alternating roles
	if len(result.Messages) < 8 {
		t.Errorf("expected at least 8 messages for a complete workflow, got %d", len(result.Messages))
	}

	// Check that it starts with a user message
	if result.Messages[0].Role != "user" {
		t.Error("expected conversation to start with user message")
	}

	// Verify some key topics are covered in order
	topics := []string{
		"directory",
		"language",
		"registry",
		"builder",
		"deploy",
		"curl",
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
