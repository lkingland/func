package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"knative.dev/func/pkg/functions"
)

// rootHelpResource
type rootHelpResource struct{}

func (r rootHelpResource) desc() *mcp.Resource {
	return &mcp.Resource{
		URI:         "function://help/root",
		Name:        "Root Help",
		Description: "--help output of the func command",
		MIMEType:    "text/plain",
	}
}

func (r rootHelpResource) handle(ctx context.Context, request *mcp.ReadResourceRequest, cmdPrefix string, executor Executor) (*mcp.ReadResourceResult, error) {
	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := strings.Fields(cmdPrefix)
	cmdParts = append(cmdParts, "--help")

	content, err := executor.Execute(ctx, "", cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return nil, err
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "function://help/root",
				MIMEType: "text/plain",
				Text:     string(content),
			},
		},
	}, nil
}

// cmdHelpResource - reusable for command help docs
type cmdHelpResource struct {
	cmd []string
	uri string
}

func (r cmdHelpResource) desc() *mcp.Resource {
	return &mcp.Resource{
		URI:         r.uri,
		Name:        r.title(),
		Description: r.description(),
		MIMEType:    "text/plain",
	}
}

// title of the help command resource as a string.
func (r cmdHelpResource) title() string {
	titleCaser := cases.Title(language.English)
	var title string
	if len(r.cmd) == 1 {
		title = titleCaser.String(r.cmd[0]) + " Command Help"
	} else {
		parts := make([]string, len(r.cmd))
		for i, part := range r.cmd {
			parts[i] = titleCaser.String(part)
		}
		title = strings.Join(parts, " ") + " Command Help"
	}
	return title
}

// description of the command being resourcified as a string.
func (r cmdHelpResource) description() string {
	var desc string
	if len(r.cmd) == 1 {
		desc = fmt.Sprintf("--help output of the '%s' command", r.cmd[0])
	} else {
		desc = fmt.Sprintf("--help output of the '%s' command", strings.Join(r.cmd, " "))
	}
	return desc
}

// handle a request for command help
func (r cmdHelpResource) handle(ctx context.Context, request *mcp.ReadResourceRequest, cmdPrefix string, executor Executor) (*mcp.ReadResourceResult, error) {
	// {executable} {subcommand} --help
	cmdParts := append(strings.Fields(cmdPrefix), r.cmd...)
	cmdParts = append(cmdParts, "--help")

	content, err := executor.Execute(ctx, "", cmdParts[0], cmdParts[1:]...)
	if err != nil {
		return nil, err
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      r.uri,
				MIMEType: "text/plain",
				Text:     string(content),
			},
		},
	}, nil
}

// templatesResource
type templatesResource struct{}

func (r templatesResource) desc() *mcp.Resource {
	return &mcp.Resource{
		URI:         "function://templates",
		Name:        "Available Templates",
		Description: "List of available function templates",
		MIMEType:    "text/plain",
	}
}

func (r templatesResource) handle(ctx context.Context, request *mcp.ReadResourceRequest, _ string, _ Executor) (*mcp.ReadResourceResult, error) {
	templates, err := fetchTemplates() // populate a []template
	if err != nil {
		return nil, err
	}
	content, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "function://templates",
				MIMEType: "text/plain",
				Text:     string(content),
			},
		},
	}, nil
}

type template struct {
	Repository   string `json:"repository"`
	Language     string `json:"language"`
	TemplateName string `json:"template"`
}

func fetchTemplates() ([]template, error) {
	var out []template
	seen := make(map[string]bool)

	for _, repoURL := range templateRepos {
		owner, repo := parseGitHubURL(repoURL)
		api := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/main?recursive=1", owner, repo)

		resp, err := http.Get(api)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var tree struct {
			Tree []struct {
				Path string `json:"path"`
			} `json:"tree"`
		}
		if err := json.Unmarshal(body, &tree); err != nil {
			return nil, err
		}

		for _, item := range tree.Tree {
			parts := strings.Split(item.Path, "/")
			if len(parts) >= 2 && !strings.HasPrefix(parts[0], ".") {
				lang, name := parts[0], parts[1]
				key := lang + "/" + name
				if !seen[key] {
					out = append(out, template{
						Language:     lang,
						TemplateName: name,
						Repository:   repoURL,
					})
					seen[key] = true
				}
			}
		}
	}
	return out, nil
}

func parseGitHubURL(url string) (owner, repo string) {
	trim := strings.TrimPrefix(url, "https://github.com/")
	parts := strings.Split(trim, "/")
	return parts[0], parts[1]
}

// currentFunctionResource
type currentFunctionResource struct{}

func (r currentFunctionResource) desc() *mcp.Resource {
	return &mcp.Resource{
		URI:         "function://current",
		Name:        "Current Function",
		Description: "Current function configuration from working directory",
		MIMEType:    "application/json",
	}
}

func (r currentFunctionResource) handle(ctx context.Context, request *mcp.ReadResourceRequest, _ string, _ Executor) (*mcp.ReadResourceResult, error) {
	// Load function from current working directory
	f, err := functions.NewFunction("")
	if err != nil {
		// Return friendly error if loading failed
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      "function://current",
					MIMEType: "text/plain",
					Text:     fmt.Sprintf("Error loading function: %v", err),
				},
			},
		}, nil
	}

	// Check if function is initialized (and thus written to disk)
	if !f.Initialized() {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      "function://current",
					MIMEType: "text/plain",
					Text:     "No Function in current directory (has one been created?)",
				},
			},
		}, nil
	}

	// Marshal Function struct to JSON for use by the client LLM
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "function://current",
				MIMEType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

// resourceHandlerFunc decorates mcp resource handlers with a command prefix and executor
type resourceHandlerFunc func(context.Context, *mcp.ReadResourceRequest, string, Executor) (*mcp.ReadResourceResult, error)

// withResourcePrefix creates a mcp.ResourceHandler that injects the prefix and executor
func withResourcePrefix(prefix string, executor Executor, impl resourceHandlerFunc) mcp.ResourceHandler {
	return func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return impl(ctx, request, prefix, executor)
	}
}
