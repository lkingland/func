package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// KNOWN ISSUES
// 1.  Some LLM CLients are unable to invoke the prompts.
// 2.  Invoking prompts with all optional parameters requires at least one
//     character of input:
//       https://github.com/anthropics/claude-code/issues/5597

const DefaultCommand = "func"

var templateRepos = []string{"https://github.com/functions-dev/templates"}

type Server struct {
	impl      *mcp.Server
	prefix    string // Command prefix to use (e.g., "func" or "kn func")
	tools     []tool
	resources []resource
	prompts   []prompt
	executor  Executor // func command executor
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
	handle(context.Context, *mcp.GetPromptRequest, string, Executor) (*mcp.GetPromptResult, error)
}

type Executor interface {
	Execute(ctx context.Context, dir string, name string, args ...string) ([]byte, error)
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

func New(options ...Option) *Server {
	s := &Server{
		prefix:   DefaultCommand,   // Default prefix
		executor: binaryExecutor{}, // Default executor
		impl: mcp.NewServer(
			&mcp.Implementation{Name: "func", Version: "1.0.0"},
			&mcp.ServerOptions{
				HasPrompts:   true,
				HasResources: true,
				HasTools:     true,
				Instructions: instructions()}),
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
			cmdHelpResource{[]string{"create"}, "func://help/create"},
			cmdHelpResource{[]string{"build"}, "func://help/build"},
			cmdHelpResource{[]string{"deploy"}, "func://help/deploy"},
			cmdHelpResource{[]string{"list"}, "func://help/list"},
			cmdHelpResource{[]string{"delete"}, "func://help/delete"},
			cmdHelpResource{[]string{"config", "volumes", "add"}, "func://help/config/volumes/add"},
			cmdHelpResource{[]string{"config", "volumes", "remove"}, "func://help/config/volumes/remove"},
			cmdHelpResource{[]string{"config", "labels", "add"}, "func://help/config/labels/add"},
			cmdHelpResource{[]string{"config", "labels", "remove"}, "func://help/config/labels/remove"},
			cmdHelpResource{[]string{"config", "envs", "add"}, "func://help/config/envs/add"},
			cmdHelpResource{[]string{"config", "envs", "remove"}, "func://help/config/envs/remove"},
			currentFunctionResource{},
			languagesResource{},
			templatesResource{},
		},
		prompts: []prompt{
			createFunctionPrompt{},
			deployFunctionPrompt{},
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
		s.impl.AddPrompt(prompt.desc(), withPromptPrefix(s.prefix, s.executor, prompt.handle))
	}

	return s
}

func (s *Server) Start(ctx context.Context) error {
	return s.impl.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) Connect(ctx context.Context, transport mcp.Transport) (*mcp.ServerSession, error) {
	return s.impl.Connect(ctx, transport, nil)
}

type binaryExecutor struct{}

func (e binaryExecutor) Execute(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd.CombinedOutput()
}

// Helpers
// --------

// checkMCPEnabled verifies the experimental MCP feature is enabled via environment variable.
// Returns an error with setup instructions if not enabled.
func checkMCPEnabled() error {
	if os.Getenv("FUNC_ENABLE_MCP") != "true" {
		return fmt.Errorf(`CANNOT EXECUTE: Function tools are disabled.

REMEMBER: Use "Function" (capitalized) for deployable Functions, "function" for code constructs.

You must tell the user to:
1. Close/exit this application completely
2. Set FUNC_ENABLE_MCP=true in their MCP client configuration
3. Restart the application

Do NOT attempt to work around this. Do NOT offer to run 'func' commands manually.
The user must restart with the environment variable set.

For setup instructions: https://github.com/knative/func/blob/main/docs/mcp-integration/integration.md`)
	}
	return nil
}
