package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// createTool implements the MCP create tool as a thin wrapper around `func create`.
type createTool struct{}

// CreateInput defines the input parameters for the create tool.
// All optional fields use pointers so we can distinguish "not provided" from "explicitly set to default".
type CreateInput struct {
	Path       string  `json:"path" jsonschema:"required,description=Path to the function project directory (must exist and contain func.yaml)"`
	Name       string  `json:"name" jsonschema:"required,description=Name of the function to create"`
	Language   string  `json:"language" jsonschema:"required,description=Language runtime to use,enum=node,enum=python,enum=go,enum=quarkus,enum=rust,enum=typescript,enum=springboot"`
	Template   *string `json:"template,omitempty" jsonschema:"description=Function template (default http),enum=http,enum=cloudevent"`
	Repository *string `json:"repository,omitempty" jsonschema:"description=Git repository URI containing custom templates"`
	Confirm    *bool   `json:"confirm,omitempty" jsonschema:"description=Prompt for confirmation before proceeding"`
	Verbose    *bool   `json:"verbose,omitempty" jsonschema:"description=Enable verbose logging output"`
}

// CreateOutput defines the structured output returned by the create tool.
type CreateOutput struct {
	Runtime  string  `json:"runtime" jsonschema:"description=Language runtime used"`
	Template *string `json:"template" jsonschema:"description=Template used"`
	Message  string  `json:"message,omitempty" jsonschema:"description=Output message from func command"`
}

func (t createTool) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "create",
		Title:       "Create Function",
		Description: "Initialize a new Function project in the current directory (like 'git init'). Function name defaults to directory name.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Create Function",
			ReadOnlyHint:    false,
			DestructiveHint: ptr(false),
			IdempotentHint:  false, // Running create twice on the same path fails because function files already exist.
		},
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the function project directory (must exist and contain func.yaml)",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "Name of the function to create",
				},
				"language": map[string]any{
					"type":        "string",
					"description": "Language runtime to use",
					"enum":        []string{"node", "python", "go", "quarkus", "rust", "typescript", "springboot"},
				},
				"template": map[string]any{
					"type":        "string",
					"description": "Function template (e.g., http, cloudevents)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Git repository URI containing custom templates",
				},
				"confirm": map[string]any{
					"type":        "boolean",
					"description": "Prompt for confirmation before proceeding",
				},
				"verbose": map[string]any{
					"type":        "boolean",
					"description": "Enable verbose logging output",
				},
			},
			"required": []string{"path", "name", "language"},
		},
	}
}

func (t createTool) handle(ctx context.Context, request toolRequestInterface, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	// Unmarshal to typed struct
	input, err := unmarshalToolInput[CreateInput](request)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid input: %v", err)), nil
	}

	// Validate path exists
	if err := validatePathExists(input.Path); err != nil {
		return errorResult(fmt.Sprintf("Invalid path: %v", err)), nil
	}

	// Build command with only provided flags
	args := []string{"create", "-l", input.Language}

	// Optional flags - only add if non-nil
	args = appendStringFlag(args, "--template", input.Template)
	args = appendStringFlag(args, "--repository", input.Repository)
	args = appendBoolFlag(args, "--confirm", input.Confirm)
	args = appendBoolFlag(args, "--verbose", input.Verbose)

	// Add function name as positional argument
	args = append(args, input.Name)

	// Parse command prefix and execute
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	output, err := executor.Execute(ctx, input.Path, cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func create failed: %s\nOutput: %s", err, string(output))), nil
	}

	// Build structured output
	result := CreateOutput{
		Runtime:  input.Language,
		Template: input.Template,
		Message:  string(output),
	}

	return jsonResult(result), nil
}
