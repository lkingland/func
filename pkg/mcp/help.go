package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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

func handleRootHelpResource(ctx context.Context, request *mcp.ReadResourceRequest, cmdPrefix string) (*mcp.ReadResourceResult, error) {
	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := strings.Fields(cmdPrefix)
	cmdParts = append(cmdParts, "--help")

	content, err := exec.Command(cmdParts[0], cmdParts[1:]...).Output()
	if err != nil {
		return nil, err
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "func://docs",
				MIMEType: "text/plain",
				Text:     string(content),
			},
		},
	}, nil
}

func runHelpCommand(args []string, uri string, cmdPrefix string) (*mcp.ReadResourceResult, error) {
	// Parse the command prefix (might be "func" or "kn func")
	cmdParts := strings.Fields(cmdPrefix)
	cmdParts = append(cmdParts, args...)
	cmdParts = append(cmdParts, "--help")

	content, err := exec.Command(cmdParts[0], cmdParts[1:]...).Output()
	if err != nil {
		return nil, err
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      uri,
				MIMEType: "text/plain",
				Text:     string(content),
			},
		},
	}, nil
}

func handleListTemplatesResource(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	templates, err := fetchTemplates()
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
				URI:      "func://templates",
				MIMEType: "text/plain",
				Text:     string(content),
			},
		},
	}, nil
}

// Resource handler types
type resourceHandlerFunc func(context.Context, *mcp.ReadResourceRequest, string) (*mcp.ReadResourceResult, error)

func withResourcePrefix(prefix string, impl resourceHandlerFunc) mcp.ResourceHandler {
	return func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return impl(ctx, request, prefix)
	}
}

// rootHelpResource
type rootHelpResource struct{}

func (r rootHelpResource) desc() *mcp.Resource {
	return &mcp.Resource{
		URI:         "func://docs",
		Name:        "Root Help Command",
		Description: "--help output of the func command",
		MIMEType:    "text/plain",
	}
}

func (r rootHelpResource) handler(prefix string) mcp.ResourceHandler {
	return withResourcePrefix(prefix, r.handle)
}

func (r rootHelpResource) handle(ctx context.Context, request *mcp.ReadResourceRequest, cmdPrefix string) (*mcp.ReadResourceResult, error) {
	return handleRootHelpResource(ctx, request, cmdPrefix)
}

// cmdHelpResource - reusable for command help docs
type cmdHelpResource struct {
	cmd []string
	uri string
}

func (r cmdHelpResource) desc() *mcp.Resource {
	// Extract command name from URI for description
	var desc string
	if len(r.cmd) == 1 {
		desc = fmt.Sprintf("--help output of the '%s' command", r.cmd[0])
	} else {
		desc = fmt.Sprintf("--help output of the '%s' command", strings.Join(r.cmd, " "))
	}

	// Extract title from command
	var title string
	if len(r.cmd) == 1 {
		title = strings.Title(r.cmd[0]) + " Command Help"
	} else {
		parts := make([]string, len(r.cmd))
		for i, part := range r.cmd {
			parts[i] = strings.Title(part)
		}
		title = strings.Join(parts, " ") + " Command Help"
	}

	return &mcp.Resource{
		URI:         r.uri,
		Name:        title,
		Description: desc,
		MIMEType:    "text/plain",
	}
}

func (r cmdHelpResource) handler(prefix string) mcp.ResourceHandler {
	return withResourcePrefix(prefix, r.handle)
}

func (r cmdHelpResource) handle(ctx context.Context, request *mcp.ReadResourceRequest, cmdPrefix string) (*mcp.ReadResourceResult, error) {
	return runHelpCommand(r.cmd, r.uri, cmdPrefix)
}

// templatesResource
type templatesResource struct{}

func (r templatesResource) desc() *mcp.Resource {
	return &mcp.Resource{
		URI:         "func://templates",
		Name:        "Available Templates",
		Description: "List of available function templates",
		MIMEType:    "text/plain",
	}
}

func (r templatesResource) handler(_ string) mcp.ResourceHandler {
	return func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return handleListTemplatesResource(ctx, request)
	}
}
