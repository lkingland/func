package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// buildTool implements the MCP build tool as a thin wrapper around `func build`.
type buildTool struct{}

// BuildInput defines the input parameters for the build tool.
// All optional fields use pointers so we can distinguish "not provided" from "explicitly set to default".
type BuildInput struct {
	Path             string  `json:"path" jsonschema:"required,description=Path to the function project directory"`
	Builder          *string `json:"builder,omitempty" jsonschema:"description=Builder to use (pack|s2i|host),enum=pack,enum=s2i,enum=host"`
	Registry         *string `json:"registry,omitempty" jsonschema:"description=Container registry for function image (e.g. ghcr.io/user)"`
	BuilderImage     *string `json:"builderImage,omitempty" jsonschema:"description=Custom builder image to use with buildpacks"`
	Image            *string `json:"image,omitempty" jsonschema:"description=Full image name (overrides registry + function name)"`
	Platform         *string `json:"platform,omitempty" jsonschema:"description=Target platform (e.g. linux/amd64)"`
	Push             *bool   `json:"push,omitempty" jsonschema:"description=Push image to registry after building"`
	RegistryInsecure *bool   `json:"registryInsecure,omitempty" jsonschema:"description=Skip TLS verification for insecure registries"`
	BuildTimestamp   *bool   `json:"buildTimestamp,omitempty" jsonschema:"description=Use actual time for image timestamp (buildpacks only)"`
	Confirm          *bool   `json:"confirm,omitempty" jsonschema:"description=Prompt for confirmation before proceeding"`
	Verbose          *bool   `json:"verbose,omitempty" jsonschema:"description=Enable verbose logging output"`
}

// BuildOutput defines the structured output returned by the build tool.
type BuildOutput struct {
	Image   string `json:"image,omitempty" jsonschema:"description=The built image name"`
	Message string `json:"message" jsonschema:"description=Output message from func command"`
}

func (t buildTool) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "build",
		Description: "Builds a function container image.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the function project directory (must exist and contain func.yaml)",
				},
				"builder": map[string]any{
					"type":        "string",
					"description": "Builder to use (pack, s2i, or host)",
					"enum":        []string{"pack", "s2i", "host"},
				},
				"registry": map[string]any{
					"type":        "string",
					"description": "Container registry for function image (e.g., ghcr.io/user)",
				},
				"builderImage": map[string]any{
					"type":        "string",
					"description": "Custom builder image to use with buildpacks",
				},
				"image": map[string]any{
					"type":        "string",
					"description": "Full image name (overrides registry + function name)",
				},
				"platform": map[string]any{
					"type":        "string",
					"description": "Target platform (e.g., linux/amd64)",
				},
				"push": map[string]any{
					"type":        "boolean",
					"description": "Push image to registry after building",
				},
				"registryInsecure": map[string]any{
					"type":        "boolean",
					"description": "Skip TLS verification for insecure registries",
				},
				"buildTimestamp": map[string]any{
					"type":        "boolean",
					"description": "Use actual time for image timestamp (buildpacks only)",
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
			"required": []string{"path"},
		},
	}
}

func (t buildTool) handle(ctx context.Context, request toolRequestInterface, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	// Unmarshal to typed struct
	input, err := unmarshalToolInput[BuildInput](request)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid input: %v", err)), nil
	}

	// Validate path exists (simple filesystem check)
	if err := validatePathExists(input.Path); err != nil {
		return errorResult(err.Error()), nil
	}

	// Build command with only provided flags
	args := []string{"build"}

	// Optional flags - only add if non-nil
	args = appendStringFlag(args, "--builder", input.Builder)
	args = appendStringFlag(args, "--registry", input.Registry)
	args = appendStringFlag(args, "--builder-image", input.BuilderImage)
	args = appendStringFlag(args, "--image", input.Image)
	args = appendStringFlag(args, "--platform", input.Platform)
	args = appendBoolFlag(args, "--push", input.Push)
	args = appendBoolFlag(args, "--registry-insecure", input.RegistryInsecure)
	args = appendBoolFlag(args, "--build-timestamp", input.BuildTimestamp)
	args = appendBoolFlag(args, "--confirm", input.Confirm)
	args = appendBoolFlag(args, "--verbose", input.Verbose)

	// Parse command prefix and execute
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	output, err := executor.Execute(ctx, input.Path, cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func build failed: %s\nOutput: %s", err, string(output))), nil
	}

	// Build structured output
	result := BuildOutput{
		Message: string(output),
	}

	return jsonResult(result), nil
}
