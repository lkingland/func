package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// configVolumesTool implements the MCP config_volumes tool as a thin wrapper around `func config volumes`.
type configVolumesTool struct{}

// ConfigVolumesInput defines the input parameters for the config_volumes tool.
type ConfigVolumesInput struct {
	Action    string  `json:"action" jsonschema:"required,description=Action to perform,enum=add,enum=remove,enum=list"`
	Path      string  `json:"path" jsonschema:"required,description=Path to the function project directory"`
	Type      *string `json:"type,omitempty" jsonschema:"description=Volume type for add action,enum=configmap,enum=secret,enum=pvc,enum=emptydir"`
	MountPath *string `json:"mountPath,omitempty" jsonschema:"description=Mount path in the container"`
	Source    *string `json:"source,omitempty" jsonschema:"description=Name of ConfigMap, Secret, or PVC"`
	Medium    *string `json:"medium,omitempty" jsonschema:"description=Storage medium for EmptyDir (Memory or empty)"`
	Size      *string `json:"size,omitempty" jsonschema:"description=Size limit for EmptyDir (e.g., 1Gi)"`
	ReadOnly  *bool   `json:"readOnly,omitempty" jsonschema:"description=Mount as read-only (PVC only)"`
	Verbose   *bool   `json:"verbose,omitempty" jsonschema:"description=Enable verbose logging output"`
}

// ConfigVolumesOutput defines the structured output returned by the config_volumes tool.
type ConfigVolumesOutput struct {
	Message string `json:"message" jsonschema:"description=Output message from func command"`
}

func (t configVolumesTool) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "config_volumes",
		Description: "Manages volume configurations for a function. Can add, remove, or list volumes in func.yaml.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"description": "Action to perform: add, remove, or list",
					"enum":        []string{"add", "remove", "list"},
				},
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the function project directory (must exist and contain func.yaml)",
				},
				"type": map[string]any{
					"type":        "string",
					"description": "Volume type for add action: configmap, secret, pvc, or emptydir",
					"enum":        []string{"configmap", "secret", "pvc", "emptydir"},
				},
				"mountPath": map[string]any{
					"type":        "string",
					"description": "Mount path for the volume in the container",
				},
				"source": map[string]any{
					"type":        "string",
					"description": "Name of the ConfigMap, Secret, or PVC to mount",
				},
				"medium": map[string]any{
					"type":        "string",
					"description": "Storage medium for EmptyDir volume: Memory or empty string",
				},
				"size": map[string]any{
					"type":        "string",
					"description": "Maximum size limit for EmptyDir volume (e.g., 1Gi)",
				},
				"readOnly": map[string]any{
					"type":        "boolean",
					"description": "Mount volume as read-only (only for PVC)",
				},
				"verbose": map[string]any{
					"type":        "boolean",
					"description": "Enable verbose logging output",
				},
			},
			"required": []string{"action", "path"},
		},
	}
}

func (t configVolumesTool) handle(ctx context.Context, request toolRequestInterface, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	// Unmarshal to typed struct
	input, err := unmarshalToolInput[ConfigVolumesInput](request)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid input: %v", err)), nil
	}

	// Validate path exists
	if err := validatePathExists(input.Path); err != nil {
		return errorResult(err.Error()), nil
	}

	// Build command based on action
	var args []string
	if input.Action == "list" {
		args = []string{"config", "volumes"}
	} else {
		args = []string{"config", "volumes", input.Action}
	}

	// Optional flags - only add if non-nil
	args = appendStringFlag(args, "--type", input.Type)
	args = appendStringFlag(args, "--mount-path", input.MountPath)
	args = appendStringFlag(args, "--source", input.Source)
	args = appendStringFlag(args, "--medium", input.Medium)
	args = appendStringFlag(args, "--size", input.Size)
	args = appendBoolFlag(args, "--read-only", input.ReadOnly)
	args = appendBoolFlag(args, "--verbose", input.Verbose)

	// Parse command prefix and execute
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	output, err := executor.Execute(ctx, input.Path, cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func config volumes failed: %s\nOutput: %s", err, string(output))), nil
	}

	// Build structured output
	result := ConfigVolumesOutput{
		Message: string(output),
	}

	return jsonResult(result), nil
}

// configLabelsTool implements the MCP config_labels tool as a thin wrapper around `func config labels`.
type configLabelsTool struct{}

// ConfigLabelsInput defines the input parameters for the config_labels tool.
type ConfigLabelsInput struct {
	Action  string  `json:"action" jsonschema:"required,description=Action to perform,enum=add,enum=remove,enum=list"`
	Path    string  `json:"path" jsonschema:"required,description=Path to the function project directory"`
	Name    *string `json:"name,omitempty" jsonschema:"description=Label name"`
	Value   *string `json:"value,omitempty" jsonschema:"description=Label value (for add action)"`
	Verbose *bool   `json:"verbose,omitempty" jsonschema:"description=Enable verbose logging output"`
}

// ConfigLabelsOutput defines the structured output returned by the config_labels tool.
type ConfigLabelsOutput struct {
	Message string `json:"message" jsonschema:"description=Output message from func command"`
}

func (t configLabelsTool) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "config_labels",
		Description: "Manages label configurations for a function. Can add, remove, or list labels in func.yaml.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"description": "Action to perform: add, remove, or list",
					"enum":        []string{"add", "remove", "list"},
				},
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the function project directory (must exist and contain func.yaml)",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "Name of the label",
				},
				"value": map[string]any{
					"type":        "string",
					"description": "Value of the label (for add action)",
				},
				"verbose": map[string]any{
					"type":        "boolean",
					"description": "Enable verbose logging output",
				},
			},
			"required": []string{"action", "path"},
		},
	}
}

func (t configLabelsTool) handle(ctx context.Context, request toolRequestInterface, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	// Unmarshal to typed struct
	input, err := unmarshalToolInput[ConfigLabelsInput](request)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid input: %v", err)), nil
	}

	// Validate path exists
	if err := validatePathExists(input.Path); err != nil {
		return errorResult(err.Error()), nil
	}

	// Build command based on action
	var args []string
	if input.Action == "list" {
		args = []string{"config", "labels"}
	} else {
		args = []string{"config", "labels", input.Action}
	}

	// Optional flags - only add if non-nil
	args = appendStringFlag(args, "--name", input.Name)
	args = appendStringFlag(args, "--value", input.Value)
	args = appendBoolFlag(args, "--verbose", input.Verbose)

	// Parse command prefix and execute
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	output, err := executor.Execute(ctx, input.Path, cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func config labels failed: %s\nOutput: %s", err, string(output))), nil
	}

	// Build structured output
	result := ConfigLabelsOutput{
		Message: string(output),
	}

	return jsonResult(result), nil
}

// configEnvsTool implements the MCP config_envs tool as a thin wrapper around `func config envs`.
type configEnvsTool struct{}

// ConfigEnvsInput defines the input parameters for the config_envs tool.
type ConfigEnvsInput struct {
	Action  string  `json:"action" jsonschema:"required,description=Action to perform,enum=add,enum=remove,enum=list"`
	Path    string  `json:"path" jsonschema:"required,description=Path to the function project directory"`
	Name    *string `json:"name,omitempty" jsonschema:"description=Environment variable name"`
	Value   *string `json:"value,omitempty" jsonschema:"description=Environment variable value (for add action)"`
	Verbose *bool   `json:"verbose,omitempty" jsonschema:"description=Enable verbose logging output"`
}

// ConfigEnvsOutput defines the structured output returned by the config_envs tool.
type ConfigEnvsOutput struct {
	Message string `json:"message" jsonschema:"description=Output message from func command"`
}

func (t configEnvsTool) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "config_envs",
		Description: "Manages environment variable configurations for a function. Can add, remove, or list environment variables in func.yaml.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"description": "Action to perform: add, remove, or list",
					"enum":        []string{"add", "remove", "list"},
				},
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the function project directory (must exist and contain func.yaml)",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "Name of the environment variable",
				},
				"value": map[string]any{
					"type":        "string",
					"description": "Value of the environment variable (for add action)",
				},
				"verbose": map[string]any{
					"type":        "boolean",
					"description": "Enable verbose logging output",
				},
			},
			"required": []string{"action", "path"},
		},
	}
}

func (t configEnvsTool) handle(ctx context.Context, request toolRequestInterface, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	// Unmarshal to typed struct
	input, err := unmarshalToolInput[ConfigEnvsInput](request)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid input: %v", err)), nil
	}

	// Validate path exists
	if err := validatePathExists(input.Path); err != nil {
		return errorResult(err.Error()), nil
	}

	// Build command based on action
	var args []string
	if input.Action == "list" {
		args = []string{"config", "envs"}
	} else {
		args = []string{"config", "envs", input.Action}
	}

	// Optional flags - only add if non-nil
	args = appendStringFlag(args, "--name", input.Name)
	args = appendStringFlag(args, "--value", input.Value)
	args = appendBoolFlag(args, "--verbose", input.Verbose)

	// Parse command prefix and execute
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	output, err := executor.Execute(ctx, input.Path, cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func config envs failed: %s\nOutput: %s", err, string(output))), nil
	}

	// Build structured output
	result := ConfigEnvsOutput{
		Message: string(output),
	}

	return jsonResult(result), nil
}
