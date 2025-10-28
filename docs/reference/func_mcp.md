## func mcp

Manage Model Context Protocol (MCP) server

### Synopsis


NAME
	func mcp - manage a Model Context Protocol (MCP) server

SYNOPSIS
	func mcp [command] [flags]

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
		func mcp --help

	Note: End users should configure their MCP client, not run these commands directly.
	See the documentation link above for configuration instructions.


### Options

```
  -h, --help   help for mcp
```

### SEE ALSO

* [func](func.md)	 - func manages Knative Functions
* [func mcp start](func_mcp_start.md)	 - Start the MCP server

