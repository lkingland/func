package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// deployFunctionPrompt - comprehensive workflow for deploying a function to Kubernetes
type deployFunctionPrompt struct{}

func (p deployFunctionPrompt) desc() *mcp.Prompt {
	return &mcp.Prompt{
		Name:        "deploy_function",
		Title:       "Guided Function Deployment",
		Description: "Interactive step-by-step guidance for deploying a Function to Kubernetes.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "language",
				Description: "Language of the current Function (for determining default builder)",
				Required:    false,
			},
			{
				Name:        "registry",
				Description: "Container registry in format domain/namespace (e.g., docker.io/alice, ghcr.io/bob, quay.io/myorg) or for local test clusters localhost:50000/func",
				Required:    false,
			},
			{
				Name:        "image",
				Description: "Complete image name override (e.g., docker.io/alice/myfunc:latest)",
				Required:    false,
			},
			{
				Name:        "builder",
				Description: "Builder: 'host', 's2i' or 'pack'",
				Required:    false,
			},
			{
				Name:        "k8s_namespace",
				Description: "Kubernetes namespace to deploy to",
				Required:    false,
			},
		},
	}
}

func (p deployFunctionPrompt) handle(ctx context.Context, request *mcp.GetPromptRequest, cmdPrefix string, executor Executor) (*mcp.GetPromptResult, error) {
	// Extract arguments with defaults
	args := request.Params.Arguments
	language := getArgOrDefault(args, "language", "")
	registry := getArgOrDefault(args, "registry", "")
	imageOverride := getArgOrDefault(args, "image", "")
	builder := getArgOrDefault(args, "builder", "")
	k8sNamespace := getArgOrDefault(args, "k8s_namespace", "")

	// Determine default builder based on language
	if builder == "" && (language == "go" || language == "python") {
		builder = "host"
	}

	// Build the conversation messages
	messages := []*mcp.PromptMessage{
		{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: "I'll help you deploy your Function to Kubernetes.\n\nFirst, let's make sure you're connected to the correct cluster and namespace.",
			},
		},
	}

	// Cluster context verification
	messages = append(messages, &mcp.PromptMessage{
		Role: "user",
		Content: &mcp.TextContent{
			Text: "How do I verify my cluster context?",
		},
	})

	clusterGuidance := "Please verify:\n\n1. **Current cluster**: Run `kubectl config current-context`\n2. **Current namespace**: Run `kubectl config view --minify --output 'jsonpath={..namespace}'`\n3. **Available namespaces**: Run `kubectl get namespaces`\n\nMake sure you're connected to the correct cluster before we proceed."
	if k8sNamespace != "" {
		clusterGuidance = fmt.Sprintf("Please verify you're connected to the correct cluster:\n\nRun: `kubectl config current-context`\n\nI'll deploy your Function to the **%s** namespace.", k8sNamespace)
	}

	messages = append(messages, &mcp.PromptMessage{
		Role: "assistant",
		Content: &mcp.TextContent{
			Text: clusterGuidance,
		},
	})

	messages = append(messages, &mcp.PromptMessage{
		Role: "user",
		Content: &mcp.TextContent{
			Text: "Confirmed! I'm connected to the right cluster.",
		},
	})

	// Registry configuration
	if imageOverride != "" {
		// User provided complete image override
		messages = append(messages, &mcp.PromptMessage{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: fmt.Sprintf("I'll use the complete image name you specified: `%s`", imageOverride),
			},
		})
	} else if registry != "" {
		// User provided registry
		messages = append(messages, &mcp.PromptMessage{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: fmt.Sprintf("I'll store your container image at: `%s/[function-name]`\n\nMake sure you're authenticated to that registry (e.g., `docker login`).", registry),
			},
		})
	} else {
		// Need to guide user through registry selection
		messages = append(messages, &mcp.PromptMessage{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: "Great! Now I need to know where to store your Function's container image.\n\n**Registry format**: `domain/namespace` (or just `domain` for localhost)\n\n**Examples**:\n- `docker.io/alice` - Docker Hub with username 'alice'\n- `ghcr.io/bob` - GitHub Container Registry with username 'bob'\n- `quay.io/myorg` - Quay.io with organization 'myorg'\n- `localhost:50000` - Local registry (no namespace needed)\n\nWhat registry would you like to use?",
			},
		})
		messages = append(messages, &mcp.PromptMessage{
			Role: "user",
			Content: &mcp.TextContent{
				Text: "I'll use [registry choice]",
			},
		})
		messages = append(messages, &mcp.PromptMessage{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: "Perfect! Your container image will be stored at: `[registry]/[function-name]`\n\nMake sure you're authenticated to that registry (e.g., `docker login`).",
			},
		})
	}

	messages = append(messages, &mcp.PromptMessage{
		Role: "user",
		Content: &mcp.TextContent{
			Text: "Registry is configured. What about the builder?",
		},
	})

	// Builder selection
	if builder != "" {
		// Builder already determined
		builderExplanation := "I'll use the **" + builder + "** builder"
		if builder == "host" {
			builderExplanation += " (uses your local environment - recommended for Go and Python)."
		} else {
			builderExplanation += " (Cloud Native Buildpacks - recommended for Node, TypeScript, Java, Rust)."
		}
		if language != "" {
			builderExplanation += fmt.Sprintf(" This is the recommended builder for %s Functions.", language)
		}
		messages = append(messages, &mcp.PromptMessage{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: builderExplanation,
			},
		})
	} else {
		// Don't know builder yet
		messages = append(messages, &mcp.PromptMessage{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: "Now let's choose a builder to create your container image:\n\n- **host** (DEFAULT): Uses your local environment\n  - Recommended for: Go, Python\n  - Faster builds, uses local dependencies\n\n- **pack**: Uses Cloud Native Buildpacks\n  - Recommended for: Node, TypeScript, Java, Rust\n  - Reproducible builds, automatic dependency detection\n\nWhich builder would you like to use? (or let me use the default)",
			},
		})
		messages = append(messages, &mcp.PromptMessage{
			Role: "user",
			Content: &mcp.TextContent{
				Text: "I'll use [builder choice]",
			},
		})
	}

	messages = append(messages, &mcp.PromptMessage{
		Role: "user",
		Content: &mcp.TextContent{
			Text: "Ready to deploy!",
		},
	})

	// Build ACTION message based on what's known
	var actionMessage string
	if imageOverride != "" || registry != "" || builder != "" || k8sNamespace != "" {
		// We know at least some parameters
		actionMessage = "Perfect! I'll now deploy your Function by calling the 'deploy' tool.\n\n**ACTION**: I will call the 'deploy' tool with:\n- path: . (current directory)"
		if imageOverride != "" {
			actionMessage += "\n- image: " + imageOverride
		} else if registry != "" {
			actionMessage += "\n- registry: " + registry
		}
		if builder != "" {
			actionMessage += "\n- builder: " + builder
		}
		if k8sNamespace != "" {
			actionMessage += "\n- namespace: " + k8sNamespace
		}
		actionMessage += "\n\nThis will:\n1. Build your container image\n2. Push it to your registry\n3. Create or update the Knative Service on your cluster\n4. Return the Function's URL"
	} else {
		// Don't know any parameters yet
		actionMessage = "Perfect! I'll now deploy your Function by calling the 'deploy' tool with your selected settings.\n\n**ACTION**: Calling the 'deploy' tool to deploy your Function to Kubernetes..."
	}

	messages = append(messages, &mcp.PromptMessage{
		Role: "assistant",
		Content: &mcp.TextContent{
			Text: actionMessage,
		},
	})

	messages = append(messages, &mcp.PromptMessage{
		Role: "user",
		Content: &mcp.TextContent{
			Text: "Deployed! What's next?",
		},
	})

	messages = append(messages, &mcp.PromptMessage{
		Role: "assistant",
		Content: &mcp.TextContent{
			Text: "Congratulations! Your Function is now deployed and running on Kubernetes.\n\nThe deployment output shows your Function's URL. You can test it with:\n```bash\ncurl <your-function-url>\n```\n\n**Next steps**:\n- **List Functions**: I can call the 'list' tool to show all deployed Functions\n- **View logs**: Run `kubectl logs -l serving.knative.dev/service=<function-name>`\n- **Check status**: Run `kubectl get ksvc`\n- **Update Function**: Modify code and I'll call 'deploy' again (it will reuse registry/builder from func.yaml)\n- **Delete Function**: I can call the 'delete' tool when you're done\n\nYour Function will auto-scale based on traffic, including scaling to zero when idle!",
		},
	})

	return &mcp.GetPromptResult{
		Description: "Deploy Function to Kubernetes Workflow",
		Messages:    messages,
	}, nil
}
