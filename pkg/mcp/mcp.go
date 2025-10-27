package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var templateRepos = []string{
	"https://github.com/functions-dev/templates",
}

type Server struct {
	impl      *mcp.Server
	prefix    string   // Command prefix to use (e.g., "func" or "kn func")
	executor  Executor // Command executor for this server instance
	tools     []tool
	resources []resource
}

type tool interface {
	desc() *mcp.Tool
	handle(context.Context, *mcp.CallToolRequest, string, Executor) (*mcp.CallToolResult, error)
}

type resource interface {
	desc() *mcp.Resource
	handler(prefix string) mcp.ResourceHandler
}

// Option is a functional option for configuring a Server
type Option func(*Server)

// WithPrefix sets the command prefix (e.g., "func" or "kn func")
func WithPrefix(prefix string) Option {
	return func(s *Server) {
		s.prefix = prefix
	}
}

// WithExecutor sets the command executor for the server
func WithExecutor(executor Executor) Option {
	return func(s *Server) {
		s.executor = executor
	}
}

// DefaultCommand is the default function command to use when executing.
const DefaultCommand = "func"

func New(options ...Option) *Server {
	s := &Server{
		prefix:   DefaultCommand, // Default prefix
		executor: realExecutor{}, // Default executor
		impl: mcp.NewServer(&mcp.Implementation{
			Name:    "func-mcp",
			Version: "1.0.0",
		}, nil),
		tools: []tool{
			healthCheck{},
			create{},
			deploy{},
			list{},
			build{},
			del{},
			configVolumes{},
			configLabels{},
			configEnvs{},
		},
		resources: []resource{
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
		},
	}

	// Apply functional options
	for _, o := range options {
		o(s)
	}

	for _, tool := range s.tools {
		s.impl.AddTool(tool.desc(), with(s.prefix, s.executor, tool.handle))
	}

	for _, resource := range s.resources {
		s.impl.AddResource(resource.desc(), resource.handler(s.prefix))
	}

	s.impl.AddPrompt(&mcp.Prompt{
		Name:        "help",
		Description: "help prompt for the root command",
	}, func(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return handleRootHelpPrompt(ctx, request, s.prefix)
	})

	s.impl.AddPrompt(&mcp.Prompt{
		Name:        "cmd_help",
		Description: "help prompt for a specific command",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "cmd",
				Description: "The command for which help is requested",
				Required:    true,
			},
		},
	}, func(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return handleCmdHelpPrompt(ctx, request, s.prefix)
	})

	s.impl.AddPrompt(&mcp.Prompt{
		Name:        "list_templates",
		Description: "prompt to list available function templates",
	}, handleListTemplatesPrompt)

	return s
}

func (s *Server) Start() error {
	return s.impl.Run(context.Background(), &mcp.StdioTransport{})
}
