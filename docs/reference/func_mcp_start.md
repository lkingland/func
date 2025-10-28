## func mcp start

Start the MCP server

### Synopsis


NAME
	func mcp start - start the Model Context Protocol (MCP) server

SYNOPSIS
	func mcp start [flags]

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
	        "command": "func",
	        "args": ["mcp", "start"]
	      }
	    }
	  }

	For complete configuration instructions, see the documentation link above.


```
func mcp start
```

### Options

```
  -h, --help   help for start
```

### SEE ALSO

* [func mcp](func_mcp.md)	 - Manage Model Context Protocol (MCP) server

