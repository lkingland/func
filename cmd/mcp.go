package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"knative.dev/func/pkg/mcp"
)

func NewMCPServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "The Functions Model Context Protocol (MCP) server",
		Long: `
NAME
	{{rootCmdUse}} mcp - The Functions Model Context Protocol (MCP) server

SYNOPSIS
	{{rootCmdUse}} mcp [command] [flags]

DESCRIPTION
	The Functions Model Context Protocol (MCP) server can be used to give agents
	the power of Functions.
	
	Configure your agentic client to use the MCP server with command
	"{{rootCmdUse}} mcp start".  Then get the conversation started with

	"Let's create a Function!".

	IMPORTANT: This is an EXPERIMENTAL feature. The FUNC_ENABLE_MCP environment
	variable must be set before the server will take any meaninigful action.

	IMPORTANT: "{{rootCmdUse}} mcp start" is designed to be invoked by MCP
	clients (such as Claude Code, Cursor, VS Code, Windsurf, etc.).  Do not run
	this directly. Instead, configure your client to invoke.  For setup
	instructions and client configuration examples, see:
	https://github.com/knative/func/blob/main/docs/mcp-integration/integration.md

AVAILABLE COMMANDS
	start    Start the MCP server (for use by your agent)

EXAMPLES

	o View this help:
		{{rootCmdUse}} mcp --help
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
	Starts the Model Context Protocol (MCP) server.

	IMPORTANT: This command is designed to be invoked by MCP clients (such as
	Claude Desktop, Cursor, VS Code, Windsurf, etc.), not run directly.

	IMPORTANT:
	- The FUNC_ENABLE_MCP environment variable must be set to "true"

	For detailed setup instructions and client configuration examples, see:
	https://github.com/knative/func/blob/main/docs/mcp-integration/integration.md
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPServer(cmd, args)
		},
	}
	return cmd
}

func runMCPServer(cmd *cobra.Command, args []string) error {
	// Get the root command's path to determine if we're "func" or "kn func"
	rootCmd := cmd.Root()
	cmdPrefix := rootCmd.Use

	s := mcp.New(mcp.WithPrefix(cmdPrefix))
	if err := s.Start(cmd.Context()); err != nil {
		log.Fatalf("Server error: %v", err)
		return err
	}
	return nil
}
