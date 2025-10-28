## func mcp

The Functions Model Context Protocol (MCP) server

### Synopsis


NAME
	func mcp - The Functions Model Context Protocol (MCP) server

SYNOPSIS
	func mcp [command] [flags]

DESCRIPTION
	The Functions Model Context Protocol (MCP) server can be used to give agents
	the power of Functions.

	Configure your agentic client to use the MCP server with command
	"func mcp start".  Then get the conversation started with

	"Let's create a Function!".

	IMPORTANT: This is an EXPERIMENTAL feature. The FUNC_ENABLE_MCP environment
	variable must be set before the server will take any meaninigful action.

	IMPORTANT: "func mcp start" is designed to be invoked by MCP
	clients (such as Claude Code, Cursor, VS Code, Windsurf, etc.).  Do not run
	this directly. Instead, configure your client to invoke.  For setup
	instructions and client configuration examples, see:
	https://github.com/knative/func/blob/main/docs/mcp-integration/integration.md

AVAILABLE COMMANDS
	start    Start the MCP server (for use by your agent)

EXAMPLES

	o View this help:
		func mcp --help


### Options

```
  -h, --help   help for mcp
```

### SEE ALSO

* [func](func.md)	 - func manages Knative Functions
* [func mcp start](func_mcp_start.md)	 - Start the MCP server

