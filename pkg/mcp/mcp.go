package mcp

import (
	"context"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var templateRepos = []string{
	"https://github.com/functions-dev/templates",
}

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
	handle(context.Context, *mcp.ReadResourceRequest, string, Executor) (*mcp.ReadResourceResult, error)
}

type prompt interface {
	desc() *mcp.Prompt
	handle(context.Context, *mcp.GetPromptRequest, string) (*mcp.GetPromptResult, error)
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
			cmdHelpResource{[]string{"create"}, "function://help/create"},
			cmdHelpResource{[]string{"build"}, "function://help/build"},
			cmdHelpResource{[]string{"deploy"}, "function://help/deploy"},
			cmdHelpResource{[]string{"list"}, "function://help/list"},
			cmdHelpResource{[]string{"delete"}, "function://help/delete"},
			cmdHelpResource{[]string{"config", "volumes", "add"}, "function://help/config/volumes/add"},
			cmdHelpResource{[]string{"config", "volumes", "remove"}, "function://help/config/volumes/remove"},
			cmdHelpResource{[]string{"config", "labels", "add"}, "function://help/config/labels/add"},
			cmdHelpResource{[]string{"config", "labels", "remove"}, "function://help/config/labels/remove"},
			cmdHelpResource{[]string{"config", "envs", "add"}, "function://help/config/envs/add"},
			cmdHelpResource{[]string{"config", "envs", "remove"}, "function://help/config/envs/remove"},
			currentFunctionResource{},
			templatesResource{},
		},
		prompts: []prompt{
			createFunctionPrompt{},
		},
	}

	for _, o := range options {
		o(s)
	}

	for _, tool := range s.tools {
		s.impl.AddTool(tool.desc(), with(s.prefix, s.executor, tool.handle))
	}

	for _, resource := range s.resources {
		s.impl.AddResource(resource.desc(), withResourcePrefix(s.prefix, s.executor, resource.handle))
	}

	for _, prompt := range s.prompts {
		s.impl.AddPrompt(prompt.desc(), withPromptPrefix(s.prefix, prompt.handle))
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
