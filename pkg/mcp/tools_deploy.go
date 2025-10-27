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
type DeployInput struct {
	Path               string  `json:"path" jsonschema:"required,description=Path to the function project directory"`
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
	Confirm            *bool   `json:"confirm,omitempty" jsonschema:"description=Prompt for confirmation before deploying"`
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
		Description: "Deploys a function to a Kubernetes cluster. Builds the image if needed and creates/updates the Knative service. Reads configuration from func.yaml.",
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
					"description": "Container registry for function image",
				},
				"image": map[string]any{
					"type":        "string",
					"description": "Full image name (overrides registry)",
				},
				"namespace": map[string]any{
					"type":        "string",
					"description": "Kubernetes namespace to deploy into",
				},
				"gitUrl": map[string]any{
					"type":        "string",
					"description": "Git URL containing the function source",
				},
				"gitBranch": map[string]any{
					"type":        "string",
					"description": "Git branch for remote deployment",
				},
				"gitDir": map[string]any{
					"type":        "string",
					"description": "Directory inside the Git repository",
				},
				"builderImage": map[string]any{
					"type":        "string",
					"description": "Custom builder image",
				},
				"domain": map[string]any{
					"type":        "string",
					"description": "Domain for the function route",
				},
				"platform": map[string]any{
					"type":        "string",
					"description": "Target platform (e.g., linux/amd64)",
				},
				"build": map[string]any{
					"type":        "string",
					"description": "Build control: true, false, or auto",
				},
				"pvcSize": map[string]any{
					"type":        "string",
					"description": "Custom volume size for remote builds",
				},
				"serviceAccount": map[string]any{
					"type":        "string",
					"description": "Kubernetes ServiceAccount to use",
				},
				"remoteStorageClass": map[string]any{
					"type":        "string",
					"description": "Storage class for remote volume",
				},
				"push": map[string]any{
					"type":        "boolean",
					"description": "Push image to registry before deployment",
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
				"confirm": map[string]any{
					"type":        "boolean",
					"description": "Prompt for confirmation before deploying",
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

func (t deployTool) handle(ctx context.Context, request toolRequestInterface, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	// Unmarshal to typed struct
	input, err := unmarshalToolInput[DeployInput](request)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid input: %v", err)), nil
	}

	// Validate path exists (simple filesystem check)
	if err := validatePathExists(input.Path); err != nil {
		return errorResult(err.Error()), nil
	}

	// Build command with only provided flags
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
	args = appendBoolFlag(args, "--confirm", input.Confirm)
	args = appendBoolFlag(args, "--verbose", input.Verbose)

	// Parse command prefix and execute
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	output, err := executor.Execute(ctx, input.Path, cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func deploy failed: %s\nOutput: %s", err, string(output))), nil
	}

	// Build structured output
	result := DeployOutput{
		Message: string(output),
	}

	return jsonResult(result), nil
}
