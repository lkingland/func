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
type DeleteInput struct {
	Name      string  `json:"name" jsonschema:"required,description=Name of the function to delete"`
	Namespace *string `json:"namespace,omitempty" jsonschema:"description=Kubernetes namespace (default: current namespace)"`
	Path      *string `json:"path,omitempty" jsonschema:"description=Path to the function project directory"`
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
		Description: "Deletes a deployed function from the Kubernetes cluster. Can optionally delete related resources like Pipelines and Secrets.",
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
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the function project directory",
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

	// Validate path if provided
	if input.Path != nil && *input.Path != "" {
		if err := validatePathExists(*input.Path); err != nil {
			return errorResult(err.Error()), nil
		}
	}

	// Build command with only provided flags
	args := []string{"delete", input.Name}

	// Optional flags - only add if non-nil
	args = appendStringFlag(args, "--namespace", input.Namespace)
	args = appendStringFlag(args, "--path", input.Path)
	args = appendStringFlag(args, "--all", input.All)
	args = appendBoolFlag(args, "--confirm", input.Confirm)
	args = appendBoolFlag(args, "--verbose", input.Verbose)

	// Parse command prefix and execute
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	// Execute from working directory (not from path, since path is optional for delete)
	workDir := ""
	if input.Path != nil {
		workDir = *input.Path
	}

	output, err := executor.Execute(ctx, workDir, cmdParts[0], cmdParts[1:]...)
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
