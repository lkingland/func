package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// promptHandlerFunc decorates mcp prompt handlers with a command prefix
type promptHandlerFunc func(context.Context, *mcp.GetPromptRequest, string) (*mcp.GetPromptResult, error)

// withPromptPrefix creates a mcp.PromptHandler that injects the prefix
func withPromptPrefix(prefix string, impl promptHandlerFunc) mcp.PromptHandler {
	return func(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return impl(ctx, request, prefix)
	}
}

// createFunctionPrompt - comprehensive workflow for creating and deploying a new function
type createFunctionPrompt struct{}

func (p createFunctionPrompt) desc() *mcp.Prompt {
	return &mcp.Prompt{
		Name:        "create_function",
		Description: "Guide me through creating a new function: choose language and template, configure container registry, select builder, and deploy to Kubernetes. Use this when starting a new function project.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "language",
				Description: "Programming language (e.g., go, python, node, typescript, rust)",
				Required:    false,
			},
			{
				Name:        "template",
				Description: "Function template type (default: http)",
				Required:    false,
			},
			{
				Name:        "registry",
				Description: "Container registry domain (e.g., docker.io, ghcr.io, quay.io)",
				Required:    false,
			},
			{
				Name:        "namespace",
				Description: "Registry namespace (your username or organization)",
				Required:    false,
			},
			{
				Name:        "builder",
				Description: "Builder type: 'host' or 'pack' (default: host for Go/Python, pack for others)",
				Required:    false,
			},
		},
	}
}

func (p createFunctionPrompt) handle(ctx context.Context, request *mcp.GetPromptRequest, cmdPrefix string) (*mcp.GetPromptResult, error) {
	// Extract arguments with defaults
	args := request.Params.Arguments
	language := getArgOrDefault(args, "language", "")
	template := getArgOrDefault(args, "template", "http")
	registry := getArgOrDefault(args, "registry", "")
	namespace := getArgOrDefault(args, "namespace", "")
	builder := getArgOrDefault(args, "builder", "")

	// Determine default builder based on language
	if builder == "" && language != "" {
		if language == "go" || language == "python" {
			builder = "host"
		} else {
			builder = "pack"
		}
	}

	// Build the conversation messages
	messages := []*mcp.PromptMessage{
		{
			Role: "user",
			Content: &mcp.TextContent{
				Text: "I want to create a new Function and deploy it to my cluster.",
			},
		},
		{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: fmt.Sprintf("I'll guide you through creating and deploying a new Function using %s.\n\nIMPORTANT: Like 'git init', you should be IN the directory where you want your Function to live. The Function name will default to your directory name.\n\nCreate and navigate to your Function directory:\n```bash\nmkdir my-function && cd my-function\n```\n\nThe Function will be named 'my-function' (from the directory name).", cmdPrefix),
			},
		},
		{
			Role: "user",
			Content: &mcp.TextContent{
				Text: "Directory is ready. What's next?",
			},
		},
	}

	// Language and template selection
	if language == "" {
		messages = append(messages, &mcp.PromptMessage{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: "Now let's choose a language and template. Here are the available templates:",
			},
		})
		messages = append(messages, &mcp.PromptMessage{
			Role: "assistant",
			Content: &mcp.EmbeddedResource{
				Resource: &mcp.ResourceContents{
					URI:      "function://templates",
					MIMEType: "text/plain",
				},
			},
		})
		messages = append(messages, &mcp.PromptMessage{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: fmt.Sprintf("Use the 'create' tool to initialize your Function. For example, to create a Go HTTP Function:\n```bash\n%s create --language go --template http\n```\n\nNote: No --name needed! It will use your directory name.", cmdPrefix),
			},
		})
	} else {
		messages = append(messages, &mcp.PromptMessage{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: fmt.Sprintf("I'll help you create a %s Function with the '%s' template.\n\nRun this in your Function directory:\n```bash\n%s create --language %s --template %s\n```\n\nThe Function name will default to your directory name.", language, template, cmdPrefix, language, template),
			},
		})
	}

	messages = append(messages, &mcp.PromptMessage{
		Role: "user",
		Content: &mcp.TextContent{
			Text: "Function initialized! What about the container registry?",
		},
	})

	// Registry configuration
	if registry == "" || namespace == "" {
		messages = append(messages, &mcp.PromptMessage{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: "You'll need to configure where your function's container image will be stored. This requires:\n\n1. **Registry domain**: Where images are hosted (e.g., docker.io, ghcr.io, quay.io)\n2. **Namespace**: Your username or organization name\n\nExample configurations:\n- Docker Hub: `docker.io/alice`\n- GitHub Container Registry: `ghcr.io/alice`\n- Quay.io: `quay.io/alice`\n\nYour final image location will be: `{registry}/{namespace}/{function-name}`\n\nFor a function named 'my-function' with registry 'docker.io/alice', the image would be: `docker.io/alice/my-function`",
			},
		})
	} else {
		messages = append(messages, &mcp.PromptMessage{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: fmt.Sprintf("Your container image will be stored at: `%s/%s/{function-name}`\n\nFor a function named 'my-function', the full image location will be:\n`%s/%s/my-function`", registry, namespace, registry, namespace),
			},
		})
	}

	// Builder selection
	messages = append(messages, &mcp.PromptMessage{
		Role: "user",
		Content: &mcp.TextContent{
			Text: "Got it. What about the builder?",
		},
	})

	if builder != "" {
		builderExplanation := "The builder creates your container image. "
		if builder == "host" {
			builderExplanation += "The 'host' builder uses your local environment (recommended for Go and Python)."
		} else {
			builderExplanation += "The 'pack' builder uses Cloud Native Buildpacks (recommended for other languages)."
		}
		messages = append(messages, &mcp.PromptMessage{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: fmt.Sprintf("%s\n\nYour function will use the '%s' builder (good choice for %s!).", builderExplanation, builder, language),
			},
		})
	} else {
		messages = append(messages, &mcp.PromptMessage{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: "You can choose between two builders:\n\n- **host**: Uses your local environment (default for Go and Python)\n- **pack**: Uses Cloud Native Buildpacks (default for other languages)\n\nThe builder will be selected automatically based on your language, or you can specify it with `--builder` flag.",
			},
		})
	}

	// Deployment
	messages = append(messages, &mcp.PromptMessage{
		Role: "user",
		Content: &mcp.TextContent{
			Text: "Ready to deploy!",
		},
	})

	deployCmd := fmt.Sprintf("%s deploy", cmdPrefix)
	if registry != "" && namespace != "" {
		deployCmd += fmt.Sprintf(" --registry %s/%s", registry, namespace)
	}
	if builder != "" {
		deployCmd += fmt.Sprintf(" --builder %s", builder)
	}

	messages = append(messages, &mcp.PromptMessage{
		Role: "assistant",
		Content: &mcp.TextContent{
			Text: fmt.Sprintf("Perfect! Before deploying, make sure you're connected to your target Kubernetes cluster:\n```bash\nkubectl config current-context\nkubectl get ns\n```\n\nOnce confirmed, deploy your function:\n```bash\n%s\n```\n\nThe deploy command will:\n1. Build your container image\n2. Push it to your registry\n3. Deploy it to your cluster\n4. Return the function's URL", deployCmd),
		},
	})

	messages = append(messages, &mcp.PromptMessage{
		Role: "user",
		Content: &mcp.TextContent{
			Text: "Deployed! How do I test it?",
		},
	})

	messages = append(messages, &mcp.PromptMessage{
		Role: "assistant",
		Content: &mcp.TextContent{
			Text: fmt.Sprintf("Great! After deployment, %s will display your function's URL. Test it with:\n\n```bash\ncurl https://your-function-url.example.com\n```\n\nYou can also:\n- View function details: `%s describe`\n- Check logs: `%s logs`\n- List all functions: `%s list`\n- Update and redeploy: Make changes and run `%s deploy` again\n\nYour function is now live and ready to handle requests!", cmdPrefix, cmdPrefix, cmdPrefix, cmdPrefix, cmdPrefix),
		},
	})

	return &mcp.GetPromptResult{
		Description: "Create and Deploy New Function Workflow",
		Messages:    messages,
	}, nil
}

// getArgOrDefault extracts an argument value or returns a default
func getArgOrDefault(args map[string]string, key, defaultValue string) string {
	if val, ok := args[key]; ok && val != "" {
		return val
	}
	return defaultValue
}
