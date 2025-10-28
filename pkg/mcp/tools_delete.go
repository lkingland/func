package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// deleteTool implements the MCP delete tool as a thin wrapper around `func delete`.
type deleteTool struct{}

// DeleteInput defines the input parameters for the delete tool.
// All optional fields use pointers so we can distinguish "not provided" from "explicitly set to default".
// All operations occur in the current working directory.
type DeleteInput struct {
	Name      string  `json:"name" jsonschema:"required,description=Name of the function to delete"`
	Namespace *string `json:"namespace,omitempty" jsonschema:"description=Kubernetes namespace (default: current namespace)"`
	All       *string `json:"all,omitempty" jsonschema:"description=Delete all related resources like Pipelines, Secrets (true/false)"`
	Confirm   *bool   `json:"confirm,omitempty" jsonschema:"description=Prompt to confirm before deletion"`
	Verbose   *bool   `json:"verbose,omitempty" jsonschema:"description=Enable verbose logging output"`
}

// DeleteOutput defines the structured output returned by the delete tool.
type DeleteOutput struct {
	Name    string `json:"name" jsonschema:"description=Name of the deleted function"`
	Message string `json:"message" jsonschema:"description=Output message from func command"`
}

func (t deleteTool) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "delete",
		Title:       "Delete Function",
		Description: "Delete a deployed Function from the Kubernetes cluster.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "Name of the function to delete",
				},
				"namespace": map[string]any{
					"type":        "string",
					"description": "Kubernetes namespace to delete from (default: current or active namespace)",
				},
				"all": map[string]any{
					"type":        "string",
					"description": "Delete all related resources like Pipelines, Secrets (true/false)",
				},
				"confirm": map[string]any{
					"type":        "boolean",
					"description": "Prompt to confirm before deletion",
				},
				"verbose": map[string]any{
					"type":        "boolean",
					"description": "Enable verbose logging output",
				},
			},
			"required": []string{"name"},
		},
	}
}

func (t deleteTool) handle(ctx context.Context, request toolRequestInterface, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	// Unmarshal to typed struct
	input, err := unmarshalToolInput[DeleteInput](request)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid input: %v", err)), nil
	}

	// Build command with only provided flags (operates in current directory)
	args := []string{"delete", input.Name}

	// Optional flags - only add if non-nil
	args = appendStringFlag(args, "--namespace", input.Namespace)
	args = appendStringFlag(args, "--all", input.All)
	args = appendBoolFlag(args, "--confirm", input.Confirm)
	args = appendBoolFlag(args, "--verbose", input.Verbose)

	// Parse command prefix and execute from current directory
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	output, err := executor.Execute(ctx, ".", cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func delete failed: %s\nOutput: %s", err, string(output))), nil
	}

	// Build structured output
	result := DeleteOutput{
		Name:    input.Name,
		Message: string(output),
	}

	return jsonResult(result), nil
}
