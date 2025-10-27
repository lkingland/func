package cmd

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"knative.dev/func/pkg/mcp"
)

func NewMCPServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage Model Context Protocol (MCP) server",
		Long: `
NAME
	{{rootCmdUse}} mcp - manage a Model Context Protocol (MCP) server

SYNOPSIS
	{{rootCmdUse}} mcp [command] [flags]

DESCRIPTION
	Manages a Model Context Protocol (MCP) server over standard input/output (stdio) transport.
	This server enables AI language models to interact with Knative Functions through the
	Model Context Protocol.

	IMPORTANT: This command is designed to be invoked by MCP clients (such as Claude Desktop,
	Cursor, VS Code, Windsurf, etc.), not run directly by users. The MCP client automatically
	launches and manages the server based on its configuration.

	For setup instructions and client configuration examples, see:
	https://github.com/knative/func/blob/main/docs/mcp-integration/integration.md

	Note: This is an EXPERIMENTAL feature. The FUNC_ENABLE_MCP environment variable must be
	set to "true" for the server to start. See documentation for details.

AVAILABLE COMMANDS
	start    Start the MCP server

EXAMPLES

	o View this help:
		{{rootCmdUse}} mcp --help

	Note: End users should configure their MCP client, not run these commands directly.
	See the documentation link above for configuration instructions.
`,
	}

	// Add the start subcommand
	cmd.AddCommand(NewMCPStartCmd())

	return cmd
}

func NewMCPStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the MCP server",
		Long: `
NAME
	{{rootCmdUse}} mcp start - start the Model Context Protocol (MCP) server

SYNOPSIS
	{{rootCmdUse}} mcp start [flags]

DESCRIPTION
	Starts a Model Context Protocol (MCP) server over standard input/output (stdio) transport.
	This server provides tools for AI language models to deploy and create Knative serverless
	functions through the Model Context Protocol.

	IMPORTANT: This command is designed to be invoked by MCP clients (such as Claude Desktop,
	Cursor, VS Code, Windsurf, etc.), not run directly by users. The MCP client automatically
	launches this command based on its configuration and manages the server lifecycle.

	PREREQUISITES:
	- The FUNC_ENABLE_MCP environment variable must be set to "true"
	- The MCP client must be properly configured

	For detailed setup instructions and client configuration examples, see:
	https://github.com/knative/func/blob/main/docs/mcp-integration/integration.md

EXAMPLES

	This command is typically not run directly. Instead, configure your MCP client with:

	  {
	    "mcpServers": {
	      "func-mcp": {
	        "command": "{{rootCmdUse}}",
	        "args": ["mcp", "start"]
	      }
	    }
	  }

	For complete configuration instructions, see the documentation link above.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPServer(cmd, args)
		},
	}
	return cmd
}

func getEnvVarInstructions() string {
	switch runtime.GOOS {
	case "windows":
		return `  Windows (PowerShell):
    Set permanently via System Environment Variables:
    [System.Environment]::SetEnvironmentVariable('FUNC_ENABLE_MCP', 'true', 'User')

    Then restart your terminal/IDE.

    Alternatively, add to your PowerShell profile ($PROFILE):
    $env:FUNC_ENABLE_MCP = "true"

  Windows (CMD):
    Set permanently via System Environment Variables GUI:
    1. Open System Properties → Advanced → Environment Variables
    2. Add User variable: FUNC_ENABLE_MCP=true
    3. Restart terminal/IDE`

	case "darwin":
		return `  macOS:
    Add to your shell config (~/.zshrc, ~/.bash_profile, etc.):
    export FUNC_ENABLE_MCP=true

    Then reload: source ~/.zshrc (or restart terminal)`

	case "linux":
		return `  Linux:
    Add to your shell config (~/.bashrc, ~/.zshrc, etc.):
    export FUNC_ENABLE_MCP=true

    Then reload: source ~/.bashrc (or restart terminal)`

	default:
		// Fallback for unknown OS - show Unix-style instructions
		return `  Unix/Linux/macOS:
    Add to your shell config (~/.bashrc, ~/.zshrc, etc.):
    export FUNC_ENABLE_MCP=true

    Then reload your shell configuration or restart terminal`
	}
}

func runMCPServer(cmd *cobra.Command, args []string) error {
	// Check if experimental MCP feature is explicitly enabled
	if os.Getenv("FUNC_ENABLE_MCP") != "true" {
		errorMsg := `Error: MCP server is currently an EXPERIMENTAL feature

WARNING: The MCP (Model Context Protocol) server enables AI language models to
execute commands on your system, including:
  - Creating and modifying code
  - Building container images
  - Deploying to Kubernetes clusters
  - Modifying function configurations

AI models can generate incorrect code and make unintended changes.
USER DISCRETION IS ADVISED.

To enable this experimental feature, set the following environment variable:

%s

For detailed instructions, see: docs/mcp-integration/integration.md`

		fmt.Fprintf(os.Stderr, errorMsg, getEnvVarInstructions())
		fmt.Fprintln(os.Stderr) // Add final newline
		os.Exit(1)
	}

	// Get the root command's path to determine if we're "func" or "kn func"
	rootCmd := cmd.Root()
	cmdPrefix := rootCmd.Use

	s := mcp.NewServer(cmdPrefix)
	if err := s.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
		return err
	}
	return nil
}
