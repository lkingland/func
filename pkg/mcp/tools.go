package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Executor abstracts command execution for testability
type Executor interface {
	Execute(ctx context.Context, dir string, name string, args ...string) ([]byte, error)
}

// binaryExecutor implements Executor using os/exec
type binaryExecutor struct{}

func (e binaryExecutor) Execute(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd.CombinedOutput()
}

// toolHandlerFunc decorates mcp tool handlers with a command prefix and executor
type toolHandlerFunc func(context.Context, *mcp.CallToolRequest, string, Executor) (*mcp.CallToolResult, error)

// with creates a mcp.ToolHandler that injects the prefix and executor
func with(prefix string, executor Executor, impl toolHandlerFunc) mcp.ToolHandler {
	return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return impl(ctx, request, prefix, executor)
	}
}

// Helper functions for the new API

func unmarshalArgs(request *mcp.CallToolRequest) (map[string]interface{}, error) {
	var args map[string]interface{}
	params := request.GetParams().(*mcp.CallToolParamsRaw)
	if err := json.Unmarshal(params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
	}
	return args, nil
}

func requireString(request *mcp.CallToolRequest, key string) (string, error) {
	args, err := unmarshalArgs(request)
	if err != nil {
		return "", err
	}
	val, ok := args[key]
	if !ok {
		return "", fmt.Errorf("%s is required", key)
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}
	if str == "" {
		return "", fmt.Errorf("%s cannot be empty", key)
	}
	return str, nil
}

func getString(request *mcp.CallToolRequest, key string, defaultVal string) string {
	args, err := unmarshalArgs(request)
	if err != nil {
		return defaultVal
	}
	val, ok := args[key]
	if !ok {
		return defaultVal
	}
	str, ok := val.(string)
	if !ok {
		return defaultVal
	}
	return str
}

func getBool(request *mcp.CallToolRequest, key string, defaultVal bool) bool {
	args, err := unmarshalArgs(request)
	if err != nil {
		return defaultVal
	}
	val, ok := args[key]
	if !ok {
		return defaultVal
	}
	b, ok := val.(bool)
	if !ok {
		return defaultVal
	}
	return b
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}

// healthCheck
type healthCheck struct{}

func (t healthCheck) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "healthcheck",
		Description: "Checks if the server is running",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}
}

func (t healthCheck) handle(ctx context.Context, request *mcp.CallToolRequest, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	body := fmt.Sprintf(`{"message": "%s"}`, "The MCP server is running!")
	return textResult(body), nil
}

// func create
type create struct{}

func (t create) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "create",
		Description: "Create a Knative function project in the current working directory (should be empty)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"cwd": map[string]interface{}{
					"type":        "string",
					"description": "Current working directory of the MCP client",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the function to create",
				},
				"language": map[string]interface{}{
					"type":        "string",
					"description": "Language runtime to use (e.g., node, go, python)",
				},
				"template": map[string]interface{}{
					"type":        "string",
					"description": "Function template (e.g., http, cloudevents)",
				},
				"repository": map[string]interface{}{
					"type":        "string",
					"description": "URI to Git repo containing the template. Overrides default template selection when provided.",
				},
				"verbose": map[string]interface{}{
					"type":        "boolean",
					"description": "Print verbose logs",
				},
			},
			"required": []string{"cwd", "name", "language"},
		},
	}
}

func (t create) handle(ctx context.Context, request *mcp.CallToolRequest, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	cwd, err := requireString(request, "cwd")
	if err != nil {
		return errorResult(err.Error()), nil
	}
	name, err := requireString(request, "name")
	if err != nil {
		return errorResult(err.Error()), nil
	}
	language, err := requireString(request, "language")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	args := []string{"create", "-l", language}

	// Optional flags
	if v := getString(request, "template", ""); v != "" {
		args = append(args, "--template", v)
	}
	if v := getString(request, "repository", ""); v != "" {
		args = append(args, "--repository", v)
	}
	if getBool(request, "confirm", false) {
		args = append(args, "--confirm")
	}
	if getBool(request, "verbose", false) {
		args = append(args, "--verbose")
	}

	// `name` is passed as a positional argument (directory to create in)
	args = append(args, name)

	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	out, err := executor.Execute(ctx, cwd, cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func create failed: %s", out)), nil
	}

	body := fmt.Sprintf(`{"result": "%s"}`, out)
	return textResult(body), nil
}

// deploy
type deploy struct{}

func (t deploy) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "deploy",
		Description: "Deploys the function to the cluster",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"registry": map[string]interface{}{
					"type":        "string",
					"description": "Registry to be used to push the function image",
				},
				"cwd": map[string]interface{}{
					"type":        "string",
					"description": "Full path of the function to be deployed",
				},
				"builder": map[string]interface{}{
					"type":        "string",
					"description": "Builder to be used to build the function image",
				},
				"image":                map[string]interface{}{"type": "string", "description": "Full image name (overrides registry)"},
				"namespace":            map[string]interface{}{"type": "string", "description": "Namespace to deploy the function into"},
				"git-url":              map[string]interface{}{"type": "string", "description": "Git URL containing the function source"},
				"git-branch":           map[string]interface{}{"type": "string", "description": "Git branch for remote deployment"},
				"git-dir":              map[string]interface{}{"type": "string", "description": "Directory inside the Git repository"},
				"builder-image":        map[string]interface{}{"type": "string", "description": "Custom builder image"},
				"domain":               map[string]interface{}{"type": "string", "description": "Domain for the function route"},
				"platform":             map[string]interface{}{"type": "string", "description": "Target platform to build for (e.g., linux/amd64)"},
				"path":                 map[string]interface{}{"type": "string", "description": "Path to the function directory"},
				"build":                map[string]interface{}{"type": "string", "description": `Build control: "true", "false", or "auto"`},
				"pvc-size":             map[string]interface{}{"type": "string", "description": "Custom volume size for remote builds"},
				"service-account":      map[string]interface{}{"type": "string", "description": "Kubernetes ServiceAccount to use"},
				"remote-storage-class": map[string]interface{}{"type": "string", "description": "Storage class for remote volume"},
				"confirm":              map[string]interface{}{"type": "boolean", "description": "Prompt for confirmation before deploying"},
				"push":                 map[string]interface{}{"type": "boolean", "description": "Push image to registry before deployment"},
				"verbose":              map[string]interface{}{"type": "boolean", "description": "Print verbose logs"},
				"registry-insecure":    map[string]interface{}{"type": "boolean", "description": "Skip TLS verification for registry"},
				"build-timestamp":      map[string]interface{}{"type": "boolean", "description": "Use actual time in image metadata"},
				"remote":               map[string]interface{}{"type": "boolean", "description": "Trigger remote deployment"},
			},
			"required": []string{"cwd", "registry", "builder"},
		},
	}
}

func (t deploy) handle(ctx context.Context, request *mcp.CallToolRequest, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	cwd, err := requireString(request, "cwd")
	if err != nil {
		return errorResult(err.Error()), nil
	}
	registry, err := requireString(request, "registry")
	if err != nil {
		return errorResult(err.Error()), nil
	}
	builder, err := requireString(request, "builder")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	args := []string{"deploy", "--builder", builder, "--registry", registry}

	// Optional flags
	if v := getString(request, "image", ""); v != "" {
		args = append(args, "--image", v)
	}
	if v := getString(request, "namespace", ""); v != "" {
		args = append(args, "--namespace", v)
	}
	if v := getString(request, "git-url", ""); v != "" {
		args = append(args, "--git-url", v)
	}
	if v := getString(request, "git-branch", ""); v != "" {
		args = append(args, "--git-branch", v)
	}
	if v := getString(request, "git-dir", ""); v != "" {
		args = append(args, "--git-dir", v)
	}
	if v := getString(request, "builder-image", ""); v != "" {
		args = append(args, "--builder-image", v)
	}
	if v := getString(request, "domain", ""); v != "" {
		args = append(args, "--domain", v)
	}
	if v := getString(request, "platform", ""); v != "" {
		args = append(args, "--platform", v)
	}
	if v := getString(request, "path", ""); v != "" {
		args = append(args, "--path", v)
	}
	if v := getString(request, "build", ""); v != "" {
		args = append(args, "--build", v)
	}
	if v := getString(request, "pvc-size", ""); v != "" {
		args = append(args, "--pvc-size", v)
	}
	if v := getString(request, "service-account", ""); v != "" {
		args = append(args, "--service-account", v)
	}
	if v := getString(request, "remote-storage-class", ""); v != "" {
		args = append(args, "--remote-storage-class", v)
	}

	if getBool(request, "confirm", false) {
		args = append(args, "--confirm")
	}
	if getBool(request, "push", false) {
		args = append(args, "--push")
	}
	if getBool(request, "verbose", false) {
		args = append(args, "--verbose")
	}
	if getBool(request, "registry-insecure", false) {
		args = append(args, "--registry-insecure")
	}
	if getBool(request, "build-timestamp", false) {
		args = append(args, "--build-timestamp")
	}
	if getBool(request, "remote", false) {
		args = append(args, "--remote")
	}

	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	out, err := executor.Execute(ctx, cwd, cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func deploy failed: %s", out)), nil
	}
	body := fmt.Sprintf(`{"result": "%s"}`, out)
	return textResult(body), nil
}

// list
type list struct{}

func (t list) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list",
		Description: "Lists all deployed functions in the current or specified namespace",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"all-namespaces": map[string]interface{}{"type": "boolean", "description": "List functions in all namespaces (overrides --namespace)"},
				"namespace":      map[string]interface{}{"type": "string", "description": "The namespace to list functions in (default is current/active)"},
				"output":         map[string]interface{}{"type": "string", "description": "Output format: human, plain, json, xml, yaml"},
				"verbose":        map[string]interface{}{"type": "boolean", "description": "Enable verbose output"},
			},
		},
	}
}

func (t list) handle(ctx context.Context, request *mcp.CallToolRequest, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	args := []string{"list"}

	// Optional flags
	if getBool(request, "all-namespaces", false) {
		args = append(args, "--all-namespaces")
	}
	if v := getString(request, "namespace", ""); v != "" {
		args = append(args, "--namespace", v)
	}
	if v := getString(request, "output", ""); v != "" {
		args = append(args, "--output", v)
	}
	if getBool(request, "verbose", false) {
		args = append(args, "--verbose")
	}

	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	out, err := executor.Execute(ctx, "", cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func list failed: %s", out)), nil
	}
	body := fmt.Sprintf(`{"result": "%s"}`, out)
	return textResult(body), nil
}

// build
type build struct{}

func (t build) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "build",
		Description: "Builds the function image in the current directory",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"cwd": map[string]interface{}{
					"type":        "string",
					"description": "Current working directory of the MCP client",
				},
				"builder": map[string]interface{}{
					"type":        "string",
					"description": "Builder to be used to build the function image (pack, s2i, host)",
				},
				"registry": map[string]interface{}{
					"type":        "string",
					"description": "Registry to be used to push the function image (e.g. ghcr.io/user)",
				},
				"builder-image":     map[string]interface{}{"type": "string", "description": "Custom builder image to use with buildpacks"},
				"image":             map[string]interface{}{"type": "string", "description": "Full image name (overrides registry + function name)"},
				"path":              map[string]interface{}{"type": "string", "description": "Path to the function directory (default is current dir)"},
				"platform":          map[string]interface{}{"type": "string", "description": "Target platform, e.g. linux/amd64 (for s2i builds)"},
				"confirm":           map[string]interface{}{"type": "boolean", "description": "Prompt for confirmation before proceeding"},
				"push":              map[string]interface{}{"type": "boolean", "description": "Push image to registry after building"},
				"verbose":           map[string]interface{}{"type": "boolean", "description": "Enable verbose logging output"},
				"registry-insecure": map[string]interface{}{"type": "boolean", "description": "Skip TLS verification for insecure registries"},
				"build-timestamp":   map[string]interface{}{"type": "boolean", "description": "Use actual time for image timestamp (buildpacks only)"},
			},
			"required": []string{"cwd", "builder", "registry"},
		},
	}
}

func (t build) handle(ctx context.Context, request *mcp.CallToolRequest, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	cwd, err := requireString(request, "cwd")
	if err != nil {
		return errorResult(err.Error()), nil
	}
	builder, err := requireString(request, "builder")
	if err != nil {
		return errorResult(err.Error()), nil
	}
	registry, err := requireString(request, "registry")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	args := []string{"build", "--builder", builder, "--registry", registry}

	// Optional flags
	if v := getString(request, "builder-image", ""); v != "" {
		args = append(args, "--builder-image", v)
	}
	if v := getString(request, "image", ""); v != "" {
		args = append(args, "--image", v)
	}
	if v := getString(request, "path", ""); v != "" {
		args = append(args, "--path", v)
	}
	if v := getString(request, "platform", ""); v != "" {
		args = append(args, "--platform", v)
	}

	if getBool(request, "confirm", false) {
		args = append(args, "--confirm")
	}
	if getBool(request, "push", false) {
		args = append(args, "--push")
	}
	if getBool(request, "verbose", false) {
		args = append(args, "--verbose")
	}
	if getBool(request, "registry-insecure", false) {
		args = append(args, "--registry-insecure")
	}
	if getBool(request, "build-timestamp", false) {
		args = append(args, "--build-timestamp")
	}

	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	out, err := executor.Execute(ctx, cwd, cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func build failed: %s", out)), nil
	}
	body := fmt.Sprintf(`{"result": "%s"}`, out)
	return textResult(body), nil
}

// delete
type del struct{}

func (t del) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "delete",
		Description: "Deletes a function from the cluster",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the function to be deleted",
				},
				"namespace": map[string]interface{}{"type": "string", "description": "Namespace to delete from (default: current or active)"},
				"path":      map[string]interface{}{"type": "string", "description": "Path to the function project (default is current directory)"},
				"all":       map[string]interface{}{"type": "string", "description": `Delete all related resources like Pipelines, Secrets ("true"/"false")`},
				"confirm":   map[string]interface{}{"type": "boolean", "description": "Prompt to confirm before deletion"},
				"verbose":   map[string]interface{}{"type": "boolean", "description": "Enable verbose output"},
			},
			"required": []string{"name"},
		},
	}
}

func (t del) handle(ctx context.Context, request *mcp.CallToolRequest, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	name, err := requireString(request, "name")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	args := []string{"delete", name}

	// Optional flags
	if v := getString(request, "namespace", ""); v != "" {
		args = append(args, "--namespace", v)
	}
	if v := getString(request, "path", ""); v != "" {
		args = append(args, "--path", v)
	}
	if v := getString(request, "all", ""); v != "" {
		args = append(args, "--all", v)
	}

	if getBool(request, "confirm", false) {
		args = append(args, "--confirm")
	}
	if getBool(request, "verbose", false) {
		args = append(args, "--verbose")
	}

	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	out, err := executor.Execute(ctx, "", cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func delete failed: %s", out)), nil
	}

	body := fmt.Sprintf(`{"result": "%s"}`, out)
	return textResult(body), nil
}

// configVolumes
type configVolumes struct{}

func (t configVolumes) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "config_volumes",
		Description: "Lists and manages configured volumes for a function",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"description": "The action to perform: 'add' to add a volume, 'remove' to remove a volume, 'list' to list volumes",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the function. Default is current directory ($FUNC_PATH)",
				},
				"type":       map[string]interface{}{"type": "string", "description": "Volume type: configmap, secret, pvc, or emptydir"},
				"mount_path": map[string]interface{}{"type": "string", "description": "Mount path for the volume in the function container"},
				"source":     map[string]interface{}{"type": "string", "description": "Name of the ConfigMap, Secret, or PVC to mount (not used for emptydir)"},
				"medium":     map[string]interface{}{"type": "string", "description": "Storage medium for EmptyDir volume: 'Memory' or '' (default)"},
				"size":       map[string]interface{}{"type": "string", "description": "Maximum size limit for EmptyDir volume (e.g., 1Gi)"},
				"read_only":  map[string]interface{}{"type": "boolean", "description": "Mount volume as read-only (only for PVC)"},
				"verbose":    map[string]interface{}{"type": "boolean", "description": "Print verbose logs ($FUNC_VERBOSE)"},
			},
			"required": []string{"action", "path"},
		},
	}
}

func (t configVolumes) handle(ctx context.Context, request *mcp.CallToolRequest, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	action, err := requireString(request, "action")
	if err != nil {
		return errorResult(err.Error()), nil
	}
	path, err := requireString(request, "path")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	if action == "list" {
		args := []string{"config", "volumes", "--path", path}
		if getBool(request, "verbose", false) {
			args = append(args, "--verbose")
		}

		cmdParts := parseCommand(cmdPrefix)
		cmdParts = append(cmdParts, args...)
		out, err := executor.Execute(ctx, "", cmdParts[0], cmdParts[1:]...)
		if err != nil {
			return errorResult(fmt.Sprintf("func config volumes list failed: %s", out)), nil
		}
		body := fmt.Sprintf(`{"result": "%s"}`, out)
		return textResult(body), nil
	}

	args := []string{"config", "volumes", action}

	if action == "add" {
		volumeType, err := requireString(request, "type")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		args = append(args, "--type", volumeType)
	}
	mountPath := getString(request, "mount_path", "")
	if mountPath == "" && action != "remove" {
		return errorResult(errors.New("mount_path is required for add action").Error()), nil
	}
	if mountPath != "" {
		args = append(args, "--mount-path", mountPath)
	}
	args = append(args, "--path", path)

	// Optional flags
	if v := getString(request, "source", ""); v != "" {
		args = append(args, "--source", v)
	}
	if v := getString(request, "medium", ""); v != "" {
		args = append(args, "--medium", v)
	}
	if v := getString(request, "size", ""); v != "" {
		args = append(args, "--size", v)
	}
	if getBool(request, "read_only", false) {
		args = append(args, "--read-only")
	}
	if getBool(request, "verbose", false) {
		args = append(args, "--verbose")
	}

	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	out, err := executor.Execute(ctx, "", cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func config volumes failed: %s", out)), nil
	}

	body := fmt.Sprintf(`{"result": "%s"}`, out)
	return textResult(body), nil
}

// configLabels
type configLabels struct{}

func (t configLabels) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "config_labels",
		Description: "Lists and manages labels for a function",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"description": "The action to perform: 'add' to add a label, 'remove' to remove a label, 'list' to list labels",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the function. Default is current directory ($FUNC_PATH)",
				},
				"name":    map[string]interface{}{"type": "string", "description": "Name of the label."},
				"value":   map[string]interface{}{"type": "string", "description": "Value of the label."},
				"verbose": map[string]interface{}{"type": "boolean", "description": "Print verbose logs ($FUNC_VERBOSE)"},
			},
			"required": []string{"action", "path"},
		},
	}
}

func (t configLabels) handle(ctx context.Context, request *mcp.CallToolRequest, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	action, err := requireString(request, "action")
	if err != nil {
		return errorResult(err.Error()), nil
	}
	path, err := requireString(request, "path")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	if action == "list" {
		args := []string{"config", "labels", "--path", path}
		if getBool(request, "verbose", false) {
			args = append(args, "--verbose")
		}

		cmdParts := parseCommand(cmdPrefix)
		cmdParts = append(cmdParts, args...)
		out, err := executor.Execute(ctx, "", cmdParts[0], cmdParts[1:]...)
		if err != nil {
			return errorResult(fmt.Sprintf("func config labels list failed: %s", out)), nil
		}
		body := fmt.Sprintf(`{"result": "%s"}`, out)
		return textResult(body), nil
	}

	args := []string{"config", "labels", action, "--path", path}

	// Optional flags
	if name := getString(request, "name", ""); name != "" {
		args = append(args, "--name", name)
	}
	if value := getString(request, "value", ""); value != "" {
		args = append(args, "--value", value)
	}
	if getBool(request, "verbose", false) {
		args = append(args, "--verbose")
	}

	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	out, err := executor.Execute(ctx, "", cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func config labels %s failed: %s", action, out)), nil
	}

	body := fmt.Sprintf(`{"result": "%s"}`, out)
	return textResult(body), nil
}

// configEnvs
type configEnvs struct{}

func (t configEnvs) desc() *mcp.Tool {
	return &mcp.Tool{
		Name:        "config_envs",
		Description: "Lists and manages environment variables for a function",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"description": "The action to perform: 'add' to add an env var, 'remove' to remove, 'list' to list env vars",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the function. Default is current directory ($FUNC_PATH)",
				},
				"name":    map[string]interface{}{"type": "string", "description": "Name of the environment variable."},
				"value":   map[string]interface{}{"type": "string", "description": "Value of the environment variable."},
				"verbose": map[string]interface{}{"type": "boolean", "description": "Print verbose logs ($FUNC_VERBOSE)"},
			},
			"required": []string{"action", "path"},
		},
	}
}

func (t configEnvs) handle(ctx context.Context, request *mcp.CallToolRequest, cmdPrefix string, executor Executor) (*mcp.CallToolResult, error) {
	action, err := requireString(request, "action")
	if err != nil {
		return errorResult(err.Error()), nil
	}
	path, err := requireString(request, "path")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	// Handle 'list' action separately
	if action == "list" {
		args := []string{"config", "envs", "--path", path}
		if getBool(request, "verbose", false) {
			args = append(args, "--verbose")
		}

		cmdParts := parseCommand(cmdPrefix)
		cmdParts = append(cmdParts, args...)
		out, err := executor.Execute(ctx, "", cmdParts[0], cmdParts[1:]...)
		if err != nil {
			return errorResult(fmt.Sprintf("func config envs list failed: %s", out)), nil
		}
		body := fmt.Sprintf(`{"result": "%s"}`, out)
		return textResult(body), nil
	}

	// Handle 'add' and 'remove' actions
	args := []string{"config", "envs", action, "--path", path}

	// Optional flags
	if name := getString(request, "name", ""); name != "" {
		args = append(args, "--name", name)
	}
	if value := getString(request, "value", ""); value != "" {
		args = append(args, "--value", value)
	}
	if getBool(request, "verbose", false) {
		args = append(args, "--verbose")
	}

	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	out, err := executor.Execute(ctx, "", cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return errorResult(fmt.Sprintf("func config envs %s failed: %s", action, out)), nil
	}

	body := fmt.Sprintf(`{"result": "%s"}`, out)
	return textResult(body), nil
}

// parseCommand splits a command string like "kn func" into its parts
func parseCommand(cmdPrefix string) []string {
	return strings.Fields(cmdPrefix)
}
