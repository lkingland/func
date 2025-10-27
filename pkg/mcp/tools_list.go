package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// listTool implements the MCP list tool as a thin wrapper around `func list`.
type listTool struct{}

// ListInput defines the input parameters for the list tool.
// All fields are optional since list can work without any parameters.
type ListInput struct {
	AllNamespaces *bool   `json:"allNamespaces,omitempty" jsonschema:"description=List functions in all namespaces"`
	Namespace     *string `json:"namespace,omitempty" jsonschema:"description=Kubernetes namespace to list functions in"`
	Output        *string `json:"output,omitempty" jsonschema:"description=Output format,enum=human,enum=plain,enum=json,enum=xml,enum=yaml"`
	Verbose       *bool   `json:"verbose,omitempty" jsonschema:"description=Enable verbose logging output"`
}

// ListOutput defines the structured output returned by the list tool.
type ListOutput struct {
	Message string `json:"message" jsonschema:"description=Output message from func command"`
}

func (t listTool) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list",
		Description: "Lists all deployed functions in the current namespace, specified namespace, or all namespaces.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"allNamespaces": map[string]any{
					"type":        "boolean",
					"description": "List functions in all namespaces (overrides namespace parameter)",
				},
				"namespace": map[string]any{
					"type":        "string",
					"description": "Kubernetes namespace to list functions in (default: current namespace)",
				},
				"output": map[string]any{
					"type":        "string",
					"description": "Output format: human, plain, json, xml, or yaml",
					"enum":        []string{"human", "plain", "json", "xml", "yaml"},
				},
				"verbose": map[string]any{
					"type":        "boolean",
					"description": "Enable verbose logging output",
				},
			},
		},
	}
}

func (t listTool) handle(ctx context.Context, request toolRequestInterface, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	// Unmarshal to typed struct
	input, err := unmarshalToolInput[ListInput](request)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid input: %v", err)), nil
	}

	// Build command with only provided flags
	args := []string{"list"}

	// Optional flags - only add if non-nil
	args = appendBoolFlag(args, "--all-namespaces", input.AllNamespaces)
	args = appendStringFlag(args, "--namespace", input.Namespace)
	args = appendStringFlag(args, "--output", input.Output)
	args = appendBoolFlag(args, "--verbose", input.Verbose)

	// Parse command prefix and execute
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	// Execute from working directory (no specific path needed for list)
	output, err := executor.Execute(ctx, "", cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func list failed: %s\nOutput: %s", err, string(output))), nil
	}

	// Build structured output
	result := ListOutput{
		Message: string(output),
	}

	return jsonResult(result), nil
}
