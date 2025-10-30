package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// parseCommand splits a command string like "kn func" into its parts
func parseCommand(cmdPrefix string) []string {
	return strings.Fields(cmdPrefix)
}

func handleHealthCheckTool(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	body := []byte(fmt.Sprintf(`{"message": "%s"}`, "The MCP server is running!"))
	return mcp.NewToolResultText(string(body)), nil
}

func handleCreateTool(
	ctx context.Context,
	request mcp.CallToolRequest,
	cmdPrefix string,
) (*mcp.CallToolResult, error) {
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

	body := []byte(fmt.Sprintf(`{"result": "%s"}`, out))
	return mcp.NewToolResultText(string(body)), nil
}

func handleDeployTool(
	ctx context.Context,
	request mcp.CallToolRequest,
	cmdPrefix string,
) (*mcp.CallToolResult, error) {
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
	body := []byte(fmt.Sprintf(`{"result": "%s"}`, out))
	return mcp.NewToolResultText(string(body)), nil
}

func handleListTool(
	ctx context.Context,
	request mcp.CallToolRequest,
	cmdPrefix string,
) (*mcp.CallToolResult, error) {
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
	body := []byte(fmt.Sprintf(`{"result": "%s"}`, out))
	return mcp.NewToolResultText(string(body)), nil
}

func handleBuildTool(
	ctx context.Context,
	request mcp.CallToolRequest,
	cmdPrefix string,
) (*mcp.CallToolResult, error) {
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

	if v := request.GetBool("confirm", false); v {
		args = append(args, "--confirm")
	}
	if v := request.GetBool("push", false); v {
		args = append(args, "--push")
	}
	if v := request.GetBool("verbose", false); v {
		args = append(args, "--verbose")
	}
	if v := request.GetBool("registry-insecure", false); v {
		args = append(args, "--registry-insecure")
	}
	if v := request.GetBool("build-timestamp", false); v {
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
	body := []byte(fmt.Sprintf(`{"result": "%s"}`, out))
	return mcp.NewToolResultText(string(body)), nil
}

func handleDeleteTool(
	ctx context.Context,
	request mcp.CallToolRequest,
	cmdPrefix string,
) (*mcp.CallToolResult, error) {
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

	body := []byte(fmt.Sprintf(`{"result": "%s"}`, out))
	return mcp.NewToolResultText(string(body)), nil
}

func handleConfigVolumesTool(
	ctx context.Context,
	request mcp.CallToolRequest,
	cmdPrefix string,
) (*mcp.CallToolResult, error) {
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

		cmd := exec.Command("func", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("func config volumes list failed: %s", out)), nil
		}
		body := []byte(fmt.Sprintf(`{"result": "%s"}`, out))
		return mcp.NewToolResultText(string(body)), nil
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

	body := []byte(fmt.Sprintf(`{"result": "%s"}`, out))
	return mcp.NewToolResultText(string(body)), nil
}

func handleConfigLabelsTool(
	ctx context.Context,
	request mcp.CallToolRequest,
	cmdPrefix string,
) (*mcp.CallToolResult, error) {
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

		cmd := exec.Command("func", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("func config labels list failed: %s", out)), nil
		}
		body := []byte(fmt.Sprintf(`{"result": "%s"}`, out))
		return mcp.NewToolResultText(string(body)), nil
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

	body := []byte(fmt.Sprintf(`{"result": "%s"}`, out))
	return mcp.NewToolResultText(string(body)), nil
}

func handleConfigEnvsTool(
	ctx context.Context,
	request mcp.CallToolRequest,
	cmdPrefix string,
) (*mcp.CallToolResult, error) {
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

		cmd := exec.Command("func", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("func config envs list failed: %s", out)), nil
		}
		body := []byte(fmt.Sprintf(`{"result": "%s"}`, out))
		return mcp.NewToolResultText(string(body)), nil
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

	body := []byte(fmt.Sprintf(`{"result": "%s"}`, out))
	return mcp.NewToolResultText(string(body)), nil
}
