package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// createFunctionPrompt - comprehensive workflow for creating a new function
type createFunctionPrompt struct{}

func (p createFunctionPrompt) desc() *mcp.Prompt {
	return &mcp.Prompt{
		Name:        "create_function",
		Title:       "Guided Function Creation",
		Description: "Interactive step-by-step guidance for creating a new Function.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "language",
				Description: "Function language (should be in the list supplied by the func://languages resource)",
				Required:    false,
			},
			{
				Name:        "template",
				Description: "Function template (optional.  Will default to http, and should be in the list supplied by the func://templates resource)",
				Required:    false,
			},
		},
	}
}

func (p createFunctionPrompt) handle(ctx context.Context, request *mcp.GetPromptRequest, cmdPrefix string, executor Executor) (*mcp.GetPromptResult, error) {
	// Extract arguments with defaults
	args := request.Params.Arguments
	language := getArgOrDefault(args, "language", "")
	template := getArgOrDefault(args, "template", "") // func automatically chooses a default template if not provided

	// Build the conversation messages
	messages := []*mcp.PromptMessage{
		{
			Role: "user",
			Content: &mcp.TextContent{
				Text: "I want to create a new Function",
			},
		},
		{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: "I'll help you create a new function. First, let ensure you are IN the directory you want the function to be created in. Your new Function's name will be derived from this directory name. (e.g., `mkdir my-function && cd my-function`).\n\n" +
					"Next, I'll confirm the available language and template options\n\n" +
					"[Reading func://languages]\n\n" +
					"[Reading func://templates]\n\n" +
					"To create your function, I need:\n" +
					"1. **Language**: What programming language would you like to use?\n" +
					"2. **Template**: What type of function?\n" +
					"Once created, I can help you deploy it.",
			},
		},
	}

	if language != "" {
		messages = append(messages, &mcp.PromptMessage{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: fmt.Sprintf("I see you specified the language %s already.  I will take that into account.", language),
			},
		})
	}
	if template != "" {
		messages = append(messages, &mcp.PromptMessage{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: fmt.Sprintf("I see you specified the language %s already. I will take that into account.", template),
			},
		})
	}

	// Add ACTION directive
	messages = append(messages, &mcp.PromptMessage{
		Role: "assistant",
		Content: &mcp.TextContent{
			Text: "**ACTION**: When ready, call the 'create' tool with the language and template parameters to create the Function in the current directory.",
		},
	})

	return &mcp.GetPromptResult{
		Description: "Interactive workflow for creating a new Function",
		Messages:    messages,
	}, nil
}
