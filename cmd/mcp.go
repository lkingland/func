package cmd

import (
	"log"

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
	This implementation aims to support tools for deploying and creating serverless functions.

	Note: This command is still under development.

AVAILABLE COMMANDS
	start    Start the MCP server

EXAMPLES

	o Start an MCP server:
		{{rootCmdUse}} mcp start

	o Display this help:
		{{rootCmdUse}} mcp
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
	Starts a Model Context Protocol (MCP) server over standard input/output (stdio) transport.
	This server provides tools for deploying and creating serverless functions.

EXAMPLES

	o Start an MCP server:
		{{rootCmdUse}} mcp start
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPServer(cmd, args)
		},
	}
	return cmd
}

func runMCPServer(cmd *cobra.Command, args []string) error {
	s := mcp.NewServer()
	if err := s.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
		return err
	}
	return nil
}
