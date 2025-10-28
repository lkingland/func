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

	Please note this is an experimental feature, and using an LLM to create
	and modify code running on your cluster requires careful supervision.
	Functions is an inherently "mutative" tool, so enabling it for your LLM
	is essentially giving (sometimes unpredictable) AI the ability to create,
	modify and delete Function instances on your currently connected cluster.
	Please set FUNC_ENABLE_MCP to acknowledge this warning and enable the
	server.

	The command "{{rootCmdUse}} mcp start" is be invoked by your MCP client
	(such as Claude Code, Cursor, VS Code, Windsurf, etc.); not run directly.
	Configure your client of choice with a new MCP server which runs this
	command.  For example, in Claude Code this can be accomplished with:
		claude mcp add func func mcp start
	Instructions for other clients can be found at:
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

	This command is designed to be invoked by MCP clients directly
	(such as Claude Code, Claude Desktop, Cursor, VS Code, Windsurf, etc.);
	not run directly.

	Please see '{{rootCMdUse}} mcp --help' for more information and
	important warnings.
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
