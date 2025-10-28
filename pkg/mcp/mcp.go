package mcp

import (
	"context"
	"fmt"
	"os"
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
	// Prepare server instructions based on environment variable
	var instructions string
	if os.Getenv("FUNC_ENABLE_MCP") != "true" {
		instructions = `⚠️  CRITICAL: FUNCTION TOOLS ARE DISABLED

TERMINOLOGY: Always capitalize "Function" when referring to a deployable Function (the service).
Use lowercase "function" only for programming concepts (functions in code).

IMPORTANT INSTRUCTIONS FOR YOU (the AI assistant):

When the user requests to create, deploy, build, or modify Functions, you MUST:

1. STOP immediately - do NOT attempt to use function tools
2. STOP immediately - do NOT offer to run 'func' commands directly
3. STOP immediately - do NOT try to work around this limitation

Instead, you MUST tell the user:

"The Function tools are currently disabled. To enable them:

1. Close/exit this application completely
2. Set FUNC_ENABLE_MCP=true in your MCP client configuration
3. Restart the application

For setup instructions, see:
https://github.com/knative/func/blob/main/docs/mcp-integration/integration.md"

DO NOT attempt any workarounds. The user must restart the client with the environment variable set.`
	} else {
		instructions = `✅ Function tools are enabled!

TERMINOLOGY: Always capitalize "Function" when referring to a deployable Function (the service).
Use lowercase "function" only for programming concepts (functions in code).

Examples:
- "Let's create a Function!" (deployable service) ✓
- "What is a Function?" (this project's concept) ✓
- "What is a function?" (programming construct) ✓
- "Let's create a function" (ambiguous - could mean code) ✗

WORKFLOW: Functions work like 'git init' - guide users to:
1. Create/navigate to the directory where they want their Function
2. Run 'func create' WITHOUT a --name parameter (it defaults to directory name)
3. This avoids the deprecated pattern of providing --name which creates a subdirectory

To help the user get started, suggest they try:
"Let's create a Function!"

This will guide them through the complete workflow of creating and deploying a Function.`
	}

	s := &Server{
		prefix:   DefaultCommand,   // Default prefix
		executor: binaryExecutor{}, // Default executor
		impl: mcp.NewServer(&mcp.Implementation{
			Name:    "func-mcp",
			Version: "1.0.0",
		}, &mcp.ServerOptions{
			Instructions: instructions,
		}),
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
