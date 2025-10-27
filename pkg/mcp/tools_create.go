package mcp

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// createTool implements the MCP create tool as a thin wrapper around `func create`.
type createTool struct{}

// CreateInput defines the input parameters for the create tool.
// All optional fields use pointers so we can distinguish "not provided" from "explicitly set to default".
type CreateInput struct {
	Path       string  `json:"path" jsonschema:"required,description=Parent directory where function will be created"`
	Name       string  `json:"name" jsonschema:"required,description=Name of the function to create"`
	Language   string  `json:"language" jsonschema:"required,description=Language runtime to use,enum=node,enum=python,enum=go,enum=quarkus,enum=rust,enum=typescript,enum=springboot"`
	Template   *string `json:"template,omitempty" jsonschema:"description=Function template (http or cloudevents)"`
	Repository *string `json:"repository,omitempty" jsonschema:"description=Git repository URI containing custom templates"`
	Confirm    *bool   `json:"confirm,omitempty" jsonschema:"description=Prompt to confirm before creating"`
	Verbose    *bool   `json:"verbose,omitempty" jsonschema:"description=Enable verbose logging output"`
}

// CreateOutput defines the structured output returned by the create tool.
type CreateOutput struct {
	FunctionPath string `json:"functionPath" jsonschema:"description=Full path to the created function"`
	Name         string `json:"name" jsonschema:"description=Name of the created function"`
	Runtime      string `json:"runtime" jsonschema:"description=Language runtime used"`
	Message      string `json:"message,omitempty" jsonschema:"description=Output message from func command"`
}

func (t createTool) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "create",
		Description: "Creates a new Knative function project in the specified directory. The function will be created as a subdirectory with the given name.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Parent directory where function will be created (must exist)",
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
					"description": "Prompt to confirm before creating",
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

	// Validate parent path exists (simple filesystem check)
	if err := validatePathExists(input.Path); err != nil {
		return errorResult(err.Error()), nil
	}

	// Build command with only provided flags
	args := []string{"create", "-l", input.Language}

	// Optional flags - only add if non-nil
	args = appendStringFlag(args, "--template", input.Template)
	args = appendStringFlag(args, "--repository", input.Repository)
	args = appendBoolFlag(args, "--confirm", input.Confirm)
	args = appendBoolFlag(args, "--verbose", input.Verbose)

	// Name is a positional argument
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
		FunctionPath: filepath.Join(input.Path, input.Name),
		Name:         input.Name,
		Runtime:      input.Language,
		Message:      string(output),
	}

	return jsonResult(result), nil
}
