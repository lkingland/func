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
	prompts   []prompt
}

type tool interface {
	desc() *mcp.Tool
	handle(context.Context, toolRequestInterface, string, Executor) (*mcp.CallToolResult, error)
}

type resource interface {
	desc() *mcp.Resource
	handler(prefix string) mcp.ResourceHandler
}

type prompt interface {
	desc() *mcp.Prompt
	handler(prefix string) mcp.PromptHandler
}

type Option func(*Server)

func WithPrefix(prefix string) Option {
	return func(s *Server) {
		s.prefix = prefix
	}
}

func WithExecutor(executor Executor) Option {
	return func(s *Server) {
		s.executor = executor
	}
}

const DefaultCommand = "func"

func New(options ...Option) *Server {
	s := &Server{
		prefix:   DefaultCommand,   // Default prefix
		executor: binaryExecutor{}, // Default executor
		impl: mcp.NewServer(&mcp.Implementation{
			Name:    "func-mcp",
			Version: "1.0.0",
		}, nil),
		tools: []tool{
			healthCheck{},
			createTool{},
			deployTool{},
			listTool{},
			buildTool{},
			deleteTool{},
			configVolumesTool{},
			configLabelsTool{},
			configEnvsTool{},
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
		prompts: []prompt{
			helpPrompt{},
			cmdHelpPrompt{},
			listTemplatesPrompt{},
		},
	}

	for _, o := range options {
		o(s)
	}

	for _, tool := range s.tools {
		s.impl.AddTool(tool.desc(), with(s.prefix, s.executor, tool.handle))
	}

	for _, resource := range s.resources {
		s.impl.AddResource(resource.desc(), resource.handler(s.prefix))
	}

	for _, prompt := range s.prompts {
		s.impl.AddPrompt(prompt.desc(), prompt.handler(s.prefix))
	}

	return s
}

// Start the server in blocking mode using default STDIO transport.
func (s *Server) Start(ctx context.Context) error {
	return s.impl.Run(ctx, &mcp.StdioTransport{})
}

// Connect starts the server with a custom context and transport (for testing)
// Non-blocking mode for testing.
func (s *Server) Connect(ctx context.Context, transport mcp.Transport) (*mcp.ServerSession, error) {
	return s.impl.Connect(ctx, transport, nil)
}
