package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var templateRepos = []string{
	"https://github.com/functions-dev/templates",
}

type Server struct {
	impl      *server.MCPServer
	cmdPrefix string // Command prefix to use (e.g., "func" or "kn func")
}

type tool interface {
	desc() mcp.Tool
	handle(context.Context, mcp.CallToolRequest, string) (*mcp.CallToolResult, error)
}

var tools = []tool{
	healthCheck{},
	create{},
	deploy{},
	list{},
	build{},
	del{},
	configVolumes{},
	configLabels{},
	configEnvs{},
}

func New(cmdPrefix string) *Server {
	s := &Server{
		cmdPrefix: cmdPrefix,
		impl: server.NewMCPServer("func-mcp", "1.0.0",
			server.WithToolCapabilities(true),
		),
	}
	if s.cmdPrefix == "" {
		s.cmdPrefix = "func"
	}

	for _, tool := range tools {
		s.impl.AddTool(tool.desc(), withPrefix(s.cmdPrefix, tool.handle))
	}

	s.impl.AddResource(mcp.NewResource(
		"func://docs",
		"Root Help Command",
		mcp.WithResourceDescription("--help output of the func command"),
		mcp.WithMIMEType("text/plain"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return handleRootHelpResource(ctx, request, s.cmdPrefix)
	})

	// Static help resources for each command
	s.impl.AddResource(mcp.NewResource(
		"func://create/docs",
		"Create Command Help",
		mcp.WithResourceDescription("--help output of the 'create' command"),
		mcp.WithMIMEType("text/plain"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return runHelpCommand([]string{"create"}, "func://create/docs", s.cmdPrefix)
	})

	s.impl.AddResource(mcp.NewResource(
		"func://build/docs",
		"Build Command Help",
		mcp.WithResourceDescription("--help output of the 'build' command"),
		mcp.WithMIMEType("text/plain"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return runHelpCommand([]string{"build"}, "func://build/docs", s.cmdPrefix)
	})

	s.impl.AddResource(mcp.NewResource(
		"func://deploy/docs",
		"Deploy Command Help",
		mcp.WithResourceDescription("--help output of the 'deploy' command"),
		mcp.WithMIMEType("text/plain"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return runHelpCommand([]string{"deploy"}, "func://deploy/docs", s.cmdPrefix)
	})

	s.impl.AddResource(mcp.NewResource(
		"func://list/docs",
		"List Command Help",
		mcp.WithResourceDescription("--help output of the 'list' command"),
		mcp.WithMIMEType("text/plain"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return runHelpCommand([]string{"list"}, "func://list/docs", s.cmdPrefix)
	})

	s.impl.AddResource(mcp.NewResource(
		"func://delete/docs",
		"Delete Command Help",
		mcp.WithResourceDescription("--help output of the 'delete' command"),
		mcp.WithMIMEType("text/plain"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return runHelpCommand([]string{"delete"}, "func://delete/docs", s.cmdPrefix)
	})

	s.impl.AddResource(mcp.NewResource(
		"func://config/volumes/add/docs",
		"Config Volumes Add Command Help",
		mcp.WithResourceDescription("--help output of the 'config volumes add' command"),
		mcp.WithMIMEType("text/plain"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return runHelpCommand([]string{"config", "volumes", "add"}, "func://config/volumes/add/docs", s.cmdPrefix)
	})

	s.impl.AddResource(mcp.NewResource(
		"func://config/volumes/remove/docs",
		"Config Volumes Remove Command Help",
		mcp.WithResourceDescription("--help output of the 'config volumes remove' command"),
		mcp.WithMIMEType("text/plain"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return runHelpCommand([]string{"config", "volumes", "remove"}, "func://config/volumes/remove/docs", s.cmdPrefix)
	})

	s.impl.AddResource(mcp.NewResource(
		"func://config/labels/add/docs",
		"Config Labels Add Command Help",
		mcp.WithResourceDescription("--help output of the 'config labels add' command"),
		mcp.WithMIMEType("text/plain"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return runHelpCommand([]string{"config", "labels", "add"}, "func://config/labels/add/docs", s.cmdPrefix)
	})

	s.impl.AddResource(mcp.NewResource(
		"func://config/labels/remove/docs",
		"Config Labels Remove Command Help",
		mcp.WithResourceDescription("--help output of the 'config labels remove' command"),
		mcp.WithMIMEType("text/plain"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return runHelpCommand([]string{"config", "labels", "remove"}, "func://config/labels/remove/docs", s.cmdPrefix)
	})

	s.impl.AddResource(mcp.NewResource(
		"func://config/envs/add/docs",
		"Config Envs Add Command Help",
		mcp.WithResourceDescription("--help output of the 'config envs add' command"),
		mcp.WithMIMEType("text/plain"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return runHelpCommand([]string{"config", "envs", "add"}, "func://config/envs/add/docs", s.cmdPrefix)
	})

	s.impl.AddResource(mcp.NewResource(
		"func://config/envs/remove/docs",
		"Config Envs Remove Command Help",
		mcp.WithResourceDescription("--help output of the 'config envs remove' command"),
		mcp.WithMIMEType("text/plain"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return runHelpCommand([]string{"config", "envs", "remove"}, "func://config/envs/remove/docs", s.cmdPrefix)
	})

	// Static resource for listing available templates
	s.impl.AddResource(mcp.NewResource(
		"func://templates",
		"Available Templates",
		mcp.WithResourceDescription("List of available function templates"),
		mcp.WithMIMEType("plain/text"),
	), handleListTemplatesResource)

	s.impl.AddPrompt(mcp.NewPrompt("help",
		mcp.WithPromptDescription("help prompt for the root command"),
	), func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return handleRootHelpPrompt(ctx, request, s.cmdPrefix)
	})

	s.impl.AddPrompt(mcp.NewPrompt("cmd_help",
		mcp.WithPromptDescription("help prompt for a specific command"),
		mcp.WithArgument("cmd",
			mcp.ArgumentDescription("The command for which help is requested"),
			mcp.RequiredArgument(),
		),
	), func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return handleCmdHelpPrompt(ctx, request, s.cmdPrefix)
	})

	s.impl.AddPrompt(mcp.NewPrompt("list_templates",
		mcp.WithPromptDescription("prompt to list available function templates"),
	), handleListTemplatesPrompt)

	return s
}

func (s *Server) Start() error {
	return server.ServeStdio(s.impl)
}
