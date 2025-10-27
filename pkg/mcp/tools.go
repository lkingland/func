package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// toolHandlerFunc decorates mcp tool handlers with a command prefix
type toolHandlerFunc func(context.Context, mcp.CallToolRequest, string) (*mcp.CallToolResult, error)

func withPrefix(prefix string, impl toolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return impl(ctx, request, prefix)
	}
}

// healthCheck
type healthCheck struct{}

func (t healthCheck) desc() mcp.Tool {
	return mcp.NewTool("healthcheck",
		mcp.WithDescription("Checks if the server is running"),
	)
}

func (t healthCheck) handle(ctx context.Context, request mcp.CallToolRequest, cmdPrefix string) (*mcp.CallToolResult, error) {
	body := fmt.Sprintf(`{"message": "%s"}`, "The MCP server is running!")
	return mcp.NewToolResultText(body), nil
}

// func create
type create struct{}

func (t create) desc() mcp.Tool {
	return mcp.NewTool("create",
		mcp.WithDescription("Create a Knative function project in the current working directory (should be empty)"),
		mcp.WithString("cwd",
			mcp.Required(),
			mcp.Description("Current working directory of the MCP client"),
		),
		mcp.WithString("language",
			mcp.Required(),
			mcp.Description("Language runtime to use (e.g., node, go, python)"),
		),

		// Optional flags
		mcp.WithString("template", mcp.Description("Function template (e.g., http, cloudevents)")),
		mcp.WithString("repository", mcp.Description("URI to Git repo containing the template. Overrides default template selection when provided.")),
		mcp.WithBoolean("verbose", mcp.Description("Print verbose logs")),
	)
}

func (t create) handle(ctx context.Context, request mcp.CallToolRequest, cmdPrefix string) (*mcp.CallToolResult, error) {
	cwd, err := request.RequireString("cwd")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	language, err := request.RequireString("language")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"create", "-l", language}

	// Optional flags
	if v := request.GetString("template", ""); v != "" {
		args = append(args, "--template", v)
	}
	if v := request.GetString("repository", ""); v != "" {
		args = append(args, "--repository", v)
	}
	if request.GetBool("confirm", false) {
		args = append(args, "--confirm")
	}
	if request.GetBool("verbose", false) {
		args = append(args, "--verbose")
	}

	// `name` is passed as a positional argument (directory to create in)
	args = append(args, name)

	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Dir = cwd

	out, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("func create failed: %s", out)), nil
	}

	body := fmt.Sprintf(`{"result": "%s"}`, out)
	return mcp.NewToolResultText(body), nil
}

// deploy
type deploy struct{}

func (t deploy) desc() mcp.Tool {
	return mcp.NewTool("deploy",
		mcp.WithDescription("Deploys the function to the cluster"),
		mcp.WithString("registry",
			mcp.Required(),
			mcp.Description("Registry to be used to push the function image"),
		),
		mcp.WithString("cwd",
			mcp.Required(),
			mcp.Description("Full path of the function to be deployed"),
		),
		mcp.WithString("builder",
			mcp.Required(),
			mcp.Description("Builder to be used to build the function image"),
		),

		// Optional flags
		mcp.WithString("image", mcp.Description("Full image name (overrides registry)")),
		mcp.WithString("namespace", mcp.Description("Namespace to deploy the function into")),
		mcp.WithString("git-url", mcp.Description("Git URL containing the function source")),
		mcp.WithString("git-branch", mcp.Description("Git branch for remote deployment")),
		mcp.WithString("git-dir", mcp.Description("Directory inside the Git repository")),
		mcp.WithString("builder-image", mcp.Description("Custom builder image")),
		mcp.WithString("domain", mcp.Description("Domain for the function route")),
		mcp.WithString("platform", mcp.Description("Target platform to build for (e.g., linux/amd64)")),
		mcp.WithString("path", mcp.Description("Path to the function directory")),
		mcp.WithString("build", mcp.Description(`Build control: "true", "false", or "auto"`)),
		mcp.WithString("pvc-size", mcp.Description("Custom volume size for remote builds")),
		mcp.WithString("service-account", mcp.Description("Kubernetes ServiceAccount to use")),
		mcp.WithString("remote-storage-class", mcp.Description("Storage class for remote volume")),

		mcp.WithBoolean("confirm", mcp.Description("Prompt for confirmation before deploying")),
		mcp.WithBoolean("push", mcp.Description("Push image to registry before deployment")),
		mcp.WithBoolean("verbose", mcp.Description("Print verbose logs")),
		mcp.WithBoolean("registry-insecure", mcp.Description("Skip TLS verification for registry")),
		mcp.WithBoolean("build-timestamp", mcp.Description("Use actual time in image metadata")),
		mcp.WithBoolean("remote", mcp.Description("Trigger remote deployment")),
	)
}

func (t deploy) handle(ctx context.Context, request mcp.CallToolRequest, cmdPrefix string) (*mcp.CallToolResult, error) {
	cwd, err := request.RequireString("cwd")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	registry, err := request.RequireString("registry")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	builder, err := request.RequireString("builder")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"deploy", "--builder", builder, "--registry", registry}

	// Optional flags
	if v := request.GetString("image", ""); v != "" {
		args = append(args, "--image", v)
	}
	if v := request.GetString("namespace", ""); v != "" {
		args = append(args, "--namespace", v)
	}
	if v := request.GetString("git-url", ""); v != "" {
		args = append(args, "--git-url", v)
	}
	if v := request.GetString("git-branch", ""); v != "" {
		args = append(args, "--git-branch", v)
	}
	if v := request.GetString("git-dir", ""); v != "" {
		args = append(args, "--git-dir", v)
	}
	if v := request.GetString("builder-image", ""); v != "" {
		args = append(args, "--builder-image", v)
	}
	if v := request.GetString("domain", ""); v != "" {
		args = append(args, "--domain", v)
	}
	if v := request.GetString("platform", ""); v != "" {
		args = append(args, "--platform", v)
	}
	if v := request.GetString("path", ""); v != "" {
		args = append(args, "--path", v)
	}
	if v := request.GetString("build", ""); v != "" {
		args = append(args, "--build", v)
	}
	if v := request.GetString("pvc-size", ""); v != "" {
		args = append(args, "--pvc-size", v)
	}
	if v := request.GetString("service-account", ""); v != "" {
		args = append(args, "--service-account", v)
	}
	if v := request.GetString("remote-storage-class", ""); v != "" {
		args = append(args, "--remote-storage-class", v)
	}

	if request.GetBool("confirm", false) {
		args = append(args, "--confirm")
	}
	if request.GetBool("push", false) {
		args = append(args, "--push")
	}
	if request.GetBool("verbose", false) {
		args = append(args, "--verbose")
	}
	if request.GetBool("registry-insecure", false) {
		args = append(args, "--registry-insecure")
	}
	if request.GetBool("build-timestamp", false) {
		args = append(args, "--build-timestamp")
	}
	if request.GetBool("remote", false) {
		args = append(args, "--remote")
	}

	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("func deploy failed: %s", out)), nil
	}
	body := fmt.Sprintf(`{"result": "%s"}`, out)
	return mcp.NewToolResultText(body), nil
}

// list
type list struct{}

func (t list) desc() mcp.Tool {
	return mcp.NewTool("list",
		mcp.WithDescription("Lists all deployed functions in the current or specified namespace"),

		// Optional flags
		mcp.WithBoolean("all-namespaces", mcp.Description("List functions in all namespaces (overrides --namespace)")),
		mcp.WithString("namespace", mcp.Description("The namespace to list functions in (default is current/active)")),
		mcp.WithString("output", mcp.Description("Output format: human, plain, json, xml, yaml")),
		mcp.WithBoolean("verbose", mcp.Description("Enable verbose output")),
	)
}

func (t list) handle(ctx context.Context, request mcp.CallToolRequest, cmdPrefix string) (*mcp.CallToolResult, error) {
	args := []string{"list"}

	// Optional flags
	if request.GetBool("all-namespaces", false) {
		args = append(args, "--all-namespaces")
	}
	if v := request.GetString("namespace", ""); v != "" {
		args = append(args, "--namespace", v)
	}
	if v := request.GetString("output", ""); v != "" {
		args = append(args, "--output", v)
	}
	if request.GetBool("verbose", false) {
		args = append(args, "--verbose")
	}

	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("func list failed: %s", out)), nil
	}
	body := fmt.Sprintf(`{"result": "%s"}`, out)
	return mcp.NewToolResultText(body), nil
}

// build
type build struct{}

func (t build) desc() mcp.Tool {
	return mcp.NewTool("build",
		mcp.WithDescription("Builds the function image in the current directory"),
		mcp.WithString("cwd",
			mcp.Required(),
			mcp.Description("Current working directory of the MCP client"),
		),
		mcp.WithString("builder",
			mcp.Required(),
			mcp.Description("Builder to be used to build the function image (pack, s2i, host)"),
		),
		mcp.WithString("registry",
			mcp.Required(),
			mcp.Description("Registry to be used to push the function image (e.g. ghcr.io/user)"),
		),

		// Optional flags
		mcp.WithString("builder-image", mcp.Description("Custom builder image to use with buildpacks")),
		mcp.WithString("image", mcp.Description("Full image name (overrides registry + function name)")),
		mcp.WithString("path", mcp.Description("Path to the function directory (default is current dir)")),
		mcp.WithString("platform", mcp.Description("Target platform, e.g. linux/amd64 (for s2i builds)")),

		mcp.WithBoolean("confirm", mcp.Description("Prompt for confirmation before proceeding")),
		mcp.WithBoolean("push", mcp.Description("Push image to registry after building")),
		mcp.WithBoolean("verbose", mcp.Description("Enable verbose logging output")),
		mcp.WithBoolean("registry-insecure", mcp.Description("Skip TLS verification for insecure registries")),
		mcp.WithBoolean("build-timestamp", mcp.Description("Use actual time for image timestamp (buildpacks only)")),
	)
}

func (t build) handle(ctx context.Context, request mcp.CallToolRequest, cmdPrefix string) (*mcp.CallToolResult, error) {
	cwd, err := request.RequireString("cwd")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	builder, err := request.RequireString("builder")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	registry, err := request.RequireString("registry")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"build", "--builder", builder, "--registry", registry}

	// Optional flags
	if v := request.GetString("builder-image", ""); v != "" {
		args = append(args, "--builder-image", v)
	}
	if v := request.GetString("image", ""); v != "" {
		args = append(args, "--image", v)
	}
	if v := request.GetString("path", ""); v != "" {
		args = append(args, "--path", v)
	}
	if v := request.GetString("platform", ""); v != "" {
		args = append(args, "--platform", v)
	}

	if request.GetBool("confirm", false) {
		args = append(args, "--confirm")
	}
	if request.GetBool("push", false) {
		args = append(args, "--push")
	}
	if request.GetBool("verbose", false) {
		args = append(args, "--verbose")
	}
	if request.GetBool("registry-insecure", false) {
		args = append(args, "--registry-insecure")
	}
	if request.GetBool("build-timestamp", false) {
		args = append(args, "--build-timestamp")
	}

	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("func build failed: %s", out)), nil
	}
	body := fmt.Sprintf(`{"result": "%s"}`, out)
	return mcp.NewToolResultText(body), nil
}

// delete
type del struct{}

func (t del) desc() mcp.Tool {
	return mcp.NewTool("delete",
		mcp.WithDescription("Deletes a function from the cluster"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the function to be deleted"),
		),

		// Optional flags
		mcp.WithString("namespace", mcp.Description("Namespace to delete from (default: current or active)")),
		mcp.WithString("path", mcp.Description("Path to the function project (default is current directory)")),
		mcp.WithString("all", mcp.Description(`Delete all related resources like Pipelines, Secrets ("true"/"false")`)),

		mcp.WithBoolean("confirm", mcp.Description("Prompt to confirm before deletion")),
		mcp.WithBoolean("verbose", mcp.Description("Enable verbose output")),
	)
}

func (t del) handle(ctx context.Context, request mcp.CallToolRequest, cmdPrefix string) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"delete", name}

	// Optional flags
	if v := request.GetString("namespace", ""); v != "" {
		args = append(args, "--namespace", v)
	}
	if v := request.GetString("path", ""); v != "" {
		args = append(args, "--path", v)
	}
	if v := request.GetString("all", ""); v != "" {
		args = append(args, "--all", v)
	}

	if request.GetBool("confirm", false) {
		args = append(args, "--confirm")
	}
	if request.GetBool("verbose", false) {
		args = append(args, "--verbose")
	}

	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("func delete failed: %s", out)), nil
	}

	body := fmt.Sprintf(`{"result": "%s"}`, out)
	return mcp.NewToolResultText(body), nil
}

// configVolumes
type configVolumes struct{}

func (t configVolumes) desc() mcp.Tool {
	return mcp.NewTool("config_volumes",
		mcp.WithDescription("Lists and manages configured volumes for a function"),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("The action to perform: 'add' to add a volume, 'remove' to remove a volume, 'list' to list volumes"),
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Path to the function. Default is current directory ($FUNC_PATH)"),
		),

		// Optional flags
		mcp.WithString("type", mcp.Description("Volume type: configmap, secret, pvc, or emptydir")),
		mcp.WithString("mount_path", mcp.Description("Mount path for the volume in the function container")),
		mcp.WithString("source", mcp.Description("Name of the ConfigMap, Secret, or PVC to mount (not used for emptydir)")),
		mcp.WithString("medium", mcp.Description("Storage medium for EmptyDir volume: 'Memory' or '' (default)")),
		mcp.WithString("size", mcp.Description("Maximum size limit for EmptyDir volume (e.g., 1Gi)")),
		mcp.WithBoolean("read_only", mcp.Description("Mount volume as read-only (only for PVC)")),
		mcp.WithBoolean("verbose", mcp.Description("Print verbose logs ($FUNC_VERBOSE)")),
	)
}

func (t configVolumes) handle(ctx context.Context, request mcp.CallToolRequest, cmdPrefix string) (*mcp.CallToolResult, error) {
	action, err := request.RequireString("action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if action == "list" {
		args := []string{"config", "volumes", "--path", path}
		if request.GetBool("verbose", false) {
			args = append(args, "--verbose")
		}

		cmdParts := parseCommand(cmdPrefix)
		cmdParts = append(cmdParts, args...)
		cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("func config volumes list failed: %s", out)), nil
		}
		body := fmt.Sprintf(`{"result": "%s"}`, out)
		return mcp.NewToolResultText(body), nil
	}

	args := []string{"config", "volumes", action}

	if action == "add" {
		volumeType, err := request.RequireString("type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		args = append(args, "--type", volumeType)
	}
	mountPath, err := request.RequireString("mount_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	args = append(args, "--mount-path", mountPath, "--path", path)

	// Optional flags
	if v := request.GetString("source", ""); v != "" {
		args = append(args, "--source", v)
	}
	if v := request.GetString("medium", ""); v != "" {
		args = append(args, "--medium", v)
	}
	if v := request.GetString("size", ""); v != "" {
		args = append(args, "--size", v)
	}
	if request.GetBool("read_only", false) {
		args = append(args, "--read-only")
	}
	if request.GetBool("verbose", false) {
		args = append(args, "--verbose")
	}

	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("func config volumes failed: %s", out)), nil
	}

	body := fmt.Sprintf(`{"result": "%s"}`, out)
	return mcp.NewToolResultText(body), nil
}

// configLabels
type configLabels struct{}

func (t configLabels) desc() mcp.Tool {
	return mcp.NewTool("config_labels",
		mcp.WithDescription("Lists and manages labels for a function"),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("The action to perform: 'add' to add a label, 'remove' to remove a label, 'list' to list labels"),
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Path to the function. Default is current directory ($FUNC_PATH)"),
		),

		// Optional flags
		mcp.WithString("name", mcp.Description("Name of the label.")),
		mcp.WithString("value", mcp.Description("Value of the label.")),
		mcp.WithBoolean("verbose", mcp.Description("Print verbose logs ($FUNC_VERBOSE)")),
	)
}

func (t configLabels) handle(ctx context.Context, request mcp.CallToolRequest, cmdPrefix string) (*mcp.CallToolResult, error) {
	action, err := request.RequireString("action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if action == "list" {
		args := []string{"config", "labels", "--path", path}
		if request.GetBool("verbose", false) {
			args = append(args, "--verbose")
		}

		cmdParts := parseCommand(cmdPrefix)
		cmdParts = append(cmdParts, args...)
		cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("func config labels list failed: %s", out)), nil
		}
		body := fmt.Sprintf(`{"result": "%s"}`, out)
		return mcp.NewToolResultText(body), nil
	}

	args := []string{"config", "labels", action, "--path", path}

	// Optional flags
	if name := request.GetString("name", ""); name != "" {
		args = append(args, "--name", name)
	}
	if value := request.GetString("value", ""); value != "" {
		args = append(args, "--value", value)
	}
	if request.GetBool("verbose", false) {
		args = append(args, "--verbose")
	}

	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("func config labels %s failed: %s", action, out)), nil
	}

	body := fmt.Sprintf(`{"result": "%s"}`, out)
	return mcp.NewToolResultText(body), nil
}

// configEnvs
type configEnvs struct{}

func (t configEnvs) desc() mcp.Tool {
	return mcp.NewTool("config_envs",
		mcp.WithDescription("Lists and manages environment variables for a function"),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("The action to perform: 'add' to add an env var, 'remove' to remove, 'list' to list env vars"),
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Path to the function. Default is current directory ($FUNC_PATH)"),
		),

		// Optional flags
		mcp.WithString("name", mcp.Description("Name of the environment variable.")),
		mcp.WithString("value", mcp.Description("Value of the environment variable.")),
		mcp.WithBoolean("verbose", mcp.Description("Print verbose logs ($FUNC_VERBOSE)")),
	)
}

func (t configEnvs) handle(ctx context.Context, request mcp.CallToolRequest, cmdPrefix string) (*mcp.CallToolResult, error) {
	action, err := request.RequireString("action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Handle 'list' action separately
	if action == "list" {
		args := []string{"config", "envs", "--path", path}
		if request.GetBool("verbose", false) {
			args = append(args, "--verbose")
		}

		cmdParts := parseCommand(cmdPrefix)
		cmdParts = append(cmdParts, args...)
		cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("func config envs list failed: %s", out)), nil
		}
		body := fmt.Sprintf(`{"result": "%s"}`, out)
		return mcp.NewToolResultText(body), nil
	}

	// Handle 'add' and 'remove' actions
	args := []string{"config", "envs", action, "--path", path}

	// Optional flags
	if name := request.GetString("name", ""); name != "" {
		args = append(args, "--name", name)
	}
	if value := request.GetString("value", ""); value != "" {
		args = append(args, "--value", value)
	}
	if request.GetBool("verbose", false) {
		args = append(args, "--verbose")
	}

	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := parseCommand(cmdPrefix)
	cmdParts = append(cmdParts, args...)

	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("func config envs %s failed: %s", action, out)), nil
	}

	body := fmt.Sprintf(`{"result": "%s"}`, out)
	return mcp.NewToolResultText(body), nil
}

// parseCommand splits a command string like "kn func" into its parts
func parseCommand(cmdPrefix string) []string {
	return strings.Fields(cmdPrefix)
}
