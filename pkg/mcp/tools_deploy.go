package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// deployTool implements the MCP deploy tool as a thin wrapper around `func deploy`.
type deployTool struct{}

// DeployInput defines the input parameters for the deploy tool.
// All optional fields use pointers so we can distinguish "not provided" from "explicitly set to default".
// All operations occur in the current working directory.
type DeployInput struct {
	Builder            *string `json:"builder,omitempty" jsonschema:"description=Builder to use (pack|s2i|host),enum=pack,enum=s2i,enum=host"`
	Registry           *string `json:"registry,omitempty" jsonschema:"description=Container registry for function image"`
	Image              *string `json:"image,omitempty" jsonschema:"description=Full image name (overrides registry)"`
	Namespace          *string `json:"namespace,omitempty" jsonschema:"description=Kubernetes namespace to deploy into"`
	GitURL             *string `json:"gitUrl,omitempty" jsonschema:"description=Git URL containing the function source"`
	GitBranch          *string `json:"gitBranch,omitempty" jsonschema:"description=Git branch for remote deployment"`
	GitDir             *string `json:"gitDir,omitempty" jsonschema:"description=Directory inside the Git repository"`
	BuilderImage       *string `json:"builderImage,omitempty" jsonschema:"description=Custom builder image"`
	Domain             *string `json:"domain,omitempty" jsonschema:"description=Domain for the function route"`
	Platform           *string `json:"platform,omitempty" jsonschema:"description=Target platform (e.g. linux/amd64)"`
	Build              *string `json:"build,omitempty" jsonschema:"description=Build control: true, false, or auto"`
	PVCSize            *string `json:"pvcSize,omitempty" jsonschema:"description=Custom volume size for remote builds"`
	ServiceAccount     *string `json:"serviceAccount,omitempty" jsonschema:"description=Kubernetes ServiceAccount to use"`
	RemoteStorageClass *string `json:"remoteStorageClass,omitempty" jsonschema:"description=Storage class for remote volume"`
	Push               *bool   `json:"push,omitempty" jsonschema:"description=Push image to registry before deployment"`
	RegistryInsecure   *bool   `json:"registryInsecure,omitempty" jsonschema:"description=Skip TLS verification for registry"`
	BuildTimestamp     *bool   `json:"buildTimestamp,omitempty" jsonschema:"description=Use actual time in image metadata"`
	Remote             *bool   `json:"remote,omitempty" jsonschema:"description=Trigger remote deployment"`
	Verbose            *bool   `json:"verbose,omitempty" jsonschema:"description=Enable verbose logging output"`
}

// DeployOutput defines the structured output returned by the deploy tool.
type DeployOutput struct {
	URL     string `json:"url,omitempty" jsonschema:"description=The deployed function URL"`
	Image   string `json:"image,omitempty" jsonschema:"description=The function image name"`
	Message string `json:"message" jsonschema:"description=Output message from func command"`
}

func (t deployTool) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "deploy",
		Title:       "Deploy Function",
		Description: "Deploy the Function to Kubernetes from the current directory. Builds the container image if needed.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"builder": map[string]any{
					"type":        "string",
					"description": "Builder to use (pack, s2i, or host)",
					"enum":        []string{"host", "pack", "s2i"},
				},
				"registry": map[string]any{
					"type":        "string",
					"description": "Container registry for the Function's image",
				},
				"image": map[string]any{
					"type":        "string",
					"description": "Full image name (overrides registry)",
				},
				"namespace": map[string]any{
					"type":        "string",
					"description": "Namespace into which the Function will be deployed",
				},
				"gitUrl": map[string]any{
					"type":        "string",
					"description": "Git URL containing the Function",
				},
				"gitBranch": map[string]any{
					"type":        "string",
					"description": "Git branch to use with remote deployment",
				},
				"gitDir": map[string]any{
					"type":        "string",
					"description": "Directory inside the Git repository when using remote Git deployment",
				},
				"builderImage": map[string]any{
					"type":        "string",
					"description": "Use a custom builder image (pack builder only)",
				},
				"domain": map[string]any{
					"type":        "string",
					"description": "Allocate a domain (route) for the Function",
				},
				"platform": map[string]any{
					"type":        "string",
					"description": "Specify a target platform (e.g., linux/amd64)",
				},
				"build": map[string]any{
					"type":        "string",
					"description": "Build the Function image: true, false, or auto (default)",
				},
				"pvcSize": map[string]any{
					"type":        "string",
					"description": "Custom volume size for remote builds",
				},
				"serviceAccount": map[string]any{
					"type":        "string",
					"description": "Kubernetes ServiceAccount",
				},
				"remoteStorageClass": map[string]any{
					"type":        "string",
					"description": "Storage class for remote volume",
				},
				"push": map[string]any{
					"type":        "boolean",
					"description": "Push image to registry before deployment (default true)",
				},
				"registryInsecure": map[string]any{
					"type":        "boolean",
					"description": "Skip TLS verification for registry",
				},
				"buildTimestamp": map[string]any{
					"type":        "boolean",
					"description": "Use actual time in image metadata",
				},
				"remote": map[string]any{
					"type":        "boolean",
					"description": "Trigger remote deployment",
				},
				"verbose": map[string]any{
					"type":        "boolean",
					"description": "Enable verbose logging output",
				},
			},
		},
	}
}

func (t deployTool) handle(ctx context.Context, request toolRequestInterface, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	// Unmarshal to typed struct
	input, err := unmarshalToolInput[DeployInput](request)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid input: %v", err)), nil
	}

	// Build command with only provided flags (operates in current directory)
	args := []string{"deploy"}

	// Optional flags - only add if non-nil
	args = appendStringFlag(args, "--builder", input.Builder)
	args = appendStringFlag(args, "--registry", input.Registry)
	args = appendStringFlag(args, "--image", input.Image)
	args = appendStringFlag(args, "--namespace", input.Namespace)
	args = appendStringFlag(args, "--git-url", input.GitURL)
	args = appendStringFlag(args, "--git-branch", input.GitBranch)
	args = appendStringFlag(args, "--git-dir", input.GitDir)
	args = appendStringFlag(args, "--builder-image", input.BuilderImage)
	args = appendStringFlag(args, "--domain", input.Domain)
	args = appendStringFlag(args, "--platform", input.Platform)
	args = appendStringFlag(args, "--build", input.Build)
	args = appendStringFlag(args, "--pvc-size", input.PVCSize)
	args = appendStringFlag(args, "--service-account", input.ServiceAccount)
	args = appendStringFlag(args, "--remote-storage-class", input.RemoteStorageClass)
	args = appendBoolFlag(args, "--push", input.Push)
	args = appendBoolFlag(args, "--registry-insecure", input.RegistryInsecure)
	args = appendBoolFlag(args, "--build-timestamp", input.BuildTimestamp)
	args = appendBoolFlag(args, "--remote", input.Remote)
	args = appendBoolFlag(args, "--verbose", input.Verbose)

	// Parse command prefix and execute in current directory
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	output, err := executor.Execute(ctx, ".", cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func deploy failed: %s\nOutput: %s", err, string(output))), nil
	}

	// Build structured output
	result := DeployOutput{
		Message: string(output),
	}

	return jsonResult(result), nil
}
