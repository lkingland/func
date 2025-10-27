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

type resource interface {
	desc() mcp.Resource
	handler(prefix string) server.ResourceHandlerFunc
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

var resources = []resource{
	rootHelpResource{},
	cmdHelpResource{[]string{"create"}, "func://create/docs"},
	cmdHelpResource{[]string{"build"}, "func://build/docs"},
	cmdHelpResource{[]string{"deploy"}, "func://deploy/docs"},
	cmdHelpResource{[]string{"list"}, "func://list/docs"},
	cmdHelpResource{[]string{"delete"}, "func://delete/docs"},
	cmdHelpResource{[]string{"config", "volumes", "add"}, "func://config/volumes/add/docs"},
	cmdHelpResource{[]string{"config", "volumes", "remove"}, "func://config/volumes/remove/docs"},
	cmdHelpResource{[]string{"config", "labels", "add"}, "func://config/labels/add/docs"},
	cmdHelpResource{[]string{"config", "labels", "remove"}, "func://config/labels/remove/docs"},
	cmdHelpResource{[]string{"config", "envs", "add"}, "func://config/envs/add/docs"},
	cmdHelpResource{[]string{"config", "envs", "remove"}, "func://config/envs/remove/docs"},
	templatesResource{},
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

	for _, resource := range resources {
		s.impl.AddResource(resource.desc(), resource.handler(s.cmdPrefix))
	}

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
