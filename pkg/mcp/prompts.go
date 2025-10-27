package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Prompt handler types
type promptHandlerFunc func(context.Context, *mcp.GetPromptRequest, string) (*mcp.GetPromptResult, error)

func withPromptPrefix(prefix string, impl promptHandlerFunc) mcp.PromptHandler {
	return func(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return impl(ctx, request, prefix)
	}
}

// helpPrompt - root help prompt
type helpPrompt struct{}

func (p helpPrompt) desc() *mcp.Prompt {
	return &mcp.Prompt{
		Name:        "help",
		Description: "help prompt for the root command",
	}
}

func (p helpPrompt) handler(prefix string) mcp.PromptHandler {
	return withPromptPrefix(prefix, p.handle)
}

func (p helpPrompt) handle(ctx context.Context, request *mcp.GetPromptRequest, cmdPrefix string) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "Help Prompt",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf("What can I do with the %s command?", cmdPrefix),
				},
			},
			{
				Role: "assistant",
				Content: &mcp.EmbeddedResource{
					Resource: &mcp.ResourceContents{
						URI:      "func://docs",
						MIMEType: "text/plain",
					},
				},
			},
		},
	}, nil
}

// cmdHelpPrompt - command-specific help prompt
type cmdHelpPrompt struct{}

func (p cmdHelpPrompt) desc() *mcp.Prompt {
	return &mcp.Prompt{
		Name:        "cmd_help",
		Description: "help prompt for a specific command",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "cmd",
				Description: "The command for which help is requested",
				Required:    true,
			},
		},
	}
}

func (p cmdHelpPrompt) handler(prefix string) mcp.PromptHandler {
	return withPromptPrefix(prefix, p.handle)
}

func (p cmdHelpPrompt) handle(ctx context.Context, request *mcp.GetPromptRequest, cmdPrefix string) (*mcp.GetPromptResult, error) {
	cmd, ok := request.Params.Arguments["cmd"]
	if !ok || cmd == "" {
		return nil, fmt.Errorf("cmd is required and must be a non-empty string")
	}

	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid cmd: %s", cmd)
	}

	return &mcp.GetPromptResult{
		Description: "Cmd Help Prompt",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf("What can I do with this %s command? Please provide help for the command: %s", cmdPrefix, cmd),
				},
			},
			{
				Role: "assistant",
				Content: &mcp.EmbeddedResource{
					Resource: &mcp.ResourceContents{
						URI:      fmt.Sprintf("func://%s/docs", strings.Join(parts, "/")),
						MIMEType: "text/plain",
					},
				},
			},
		},
	}, nil
}

// listTemplatesPrompt - prompt to list available function templates
type listTemplatesPrompt struct{}

func (p listTemplatesPrompt) desc() *mcp.Prompt {
	return &mcp.Prompt{
		Name:        "list_templates",
		Description: "prompt to list available function templates",
	}
}

func (p listTemplatesPrompt) handler(_ string) mcp.PromptHandler {
	return func(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return p.handle(ctx, request)
	}
}

func (p listTemplatesPrompt) handle(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "List Templates Prompt",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: "List available function templates",
				},
			},
			{
				Role: "assistant",
				Content: &mcp.EmbeddedResource{
					Resource: &mcp.ResourceContents{
						URI:      "func://templates",
						MIMEType: "text/plain",
					},
				},
			},
		},
	}, nil
}
