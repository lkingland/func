package function

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

// Path to builtin repositories.
// note: this constant must be defined in the same file in which it is used due
// to pkger performing static analysis on source files separately.
const builtinRepositories = "/templates"

// Repository
type Repository struct {
	Name      string
	URL       string // (empty if not a git repo with an upstream URL)
	Templates []Template
	Runtimes  []string
}

// NewRepository from path.
// Represents the file structure of 'path' at time of construction as
// a Repository with Templates, each of which has a Name and its Runtime.
// a convenience member of Runtimes is the unique, sorted list of all
// runtimes
func NewRepositoryFromPath(path string) (Repository, error) {
	// TODO: read and use manifest if it exists

	r := Repository{
		Name:      filepath.Base(path),
		URL:       readURL(path),
		Templates: []Template{},
		Runtimes:  []string{}}

	// Each subdirectory is a Runtime
	runtimes, err := ioutil.ReadDir(path)
	if err != nil {
		return r, err
	}
	for _, runtime := range runtimes {
		if !runtime.IsDir() || strings.HasPrefix(runtime.Name(), ".") {
			continue // ignore files and hidden
		}
		r.Runtimes = append(r.Runtimes, runtime.Name())

		// Each subdirectory is a Template
		templates, err := ioutil.ReadDir(filepath.Join(path, runtime.Name()))
		if err != nil {
			return r, err
		}
		for _, template := range templates {
			if !template.IsDir() || strings.HasPrefix(template.Name(), ".") {
				continue // ignore files and hidden
			}
			r.Templates = append(r.Templates, Template{
				Runtime:    runtime.Name(),
				Repository: r.Name,
				Name:       template.Name()})
		}
	}
	return r, nil
}

// GetTemplate from repo with given runtime
func (r *Repository) GetTemplate(runtime, name string) (Template, error) {
	// TODO: return a typed RuntimeNotFound in repo X
	// rather than the generic Template Not Found
	for _, t := range r.Templates {
		if t.Runtime == runtime && t.Name == name {
			return t, nil
		}
	}
	// TODO: Typed TemplateNotFound in repo X
	return Template{}, errors.New("template not found")
}

// readURL attempts to read the remote git origin URL of the repository.  Best
// effort; returns empty string if the repository is not a git repo or the repo
// has been mutated beyond recognition on disk (ex: removing the origin remote)
func readURL(path string) string {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return "" // not a git repository
	}

	c, err := repo.Config()
	if err != nil {
		return "" // Has no .git/config or other error.
	}

	if _, ok := c.Remotes["origin"]; ok {
		urls := c.Remotes["origin"].URLs
		if len(urls) > 0 {
			return urls[0]
		}
	}
	return ""
}
