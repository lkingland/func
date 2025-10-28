## func mcp start

Start the MCP server

### Synopsis


NAME
	func mcp start - start the Model Context Protocol (MCP) server

SYNOPSIS
	func mcp start [flags]

DESCRIPTION
	Starts the Model Context Protocol (MCP) server.

	IMPORTANT: This command is designed to be invoked by MCP clients (such as
	Claude Desktop, Cursor, VS Code, Windsurf, etc.), not run directly.

	IMPORTANT:
	- The FUNC_ENABLE_MCP environment variable must be set to "true"

	For detailed setup instructions and client configuration examples, see:
	https://github.com/knative/func/blob/main/docs/mcp-integration/integration.md


```
func mcp start
```

### Options

```
  -h, --help   help for start
```

### SEE ALSO

* [func mcp](func_mcp.md)	 - The Functions Model Context Protocol (MCP) server

