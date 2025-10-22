## func mcp

Manage Model Context Protocol (MCP) server

### Synopsis


NAME
	func mcp - manage a Model Context Protocol (MCP) server

SYNOPSIS
	func mcp [command] [flags]

DESCRIPTION
	Manages a Model Context Protocol (MCP) server over standard input/output (stdio) transport.
	This implementation aims to support tools for deploying and creating serverless functions.

	Note: This command is still under development.

AVAILABLE COMMANDS
	start    Start the MCP server

EXAMPLES

	o Start an MCP server:
		func mcp start

	o Display this help:
		func mcp
		func mcp --help


### Options

```
  -h, --help   help for mcp
```

### SEE ALSO

* [func](func.md)	 - func manages Knative Functions
* [func mcp start](func_mcp_start.md)	 - Start the MCP server

