package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ory/viper"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/client/pkg/util"

	fn "knative.dev/kn-plugin-func"
)

var exampleTemplate = template.Must(template.New("example").Parse(`
# Create a node function called "node-sample" and enter the directory
{{.}} create myfunc && cd myfunc

# Build the container image, push it to a registry and deploy it to the connected Knative cluster
# (replace <registry/user> with something like quay.io/user with an account that have you access to)
{{.}} deploy --registry <registry/user>

# Curl the service with the service URL
curl $(kn service describe myfunc -o url)
`))

// The root of the command tree defines the command name, description, globally
// available flags, etc.  It has no action of its own, such that running the
// resultant binary with no arguments prints the help/usage text.
var root = &cobra.Command{
	Use:           "func",
	Short:         "Serverless functions",
	SilenceErrors: true, // we explicitly handle errors in Execute()
	SilenceUsage:  true, // no usage dump on error
	Long: `Serverless functions

Create, build and deploy functions in serverless containers for multiple runtimes on Knative`,
}

func init() {
	var err error
	root.Example, err = replaceNameInTemplate("func", "example")
	if err != nil {
		root.Example = "Usage could not be loaded"
	}
}

func replaceNameInTemplate(name, template string) (string, error) {
	var buffer bytes.Buffer
	err := exampleTemplate.ExecuteTemplate(&buffer, template, name)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// pluginPrefix returns an optional prefix for help commands based on the
// value of the FUNC_PARENT_COMMAND environment variable.
func pluginPrefix() string {
	parent := os.Getenv("FUNC_PARENT_COMMAND")
	if parent != "" {
		return parent + " "
	}
	return ""
}

// NewRootCmd can be used to embed the func commands as a library, such as
// a plugin in 'kn'.
func NewRootCmd() (*cobra.Command, error) {
	// TODO: have 'kn' provide this environment variable, such that this works
	// generically, and works whether it is compiled in as a library or used via
	// the `kn-func` binary naming convention.
	os.Setenv("FUNC_PARENT_COMMAND", "kn")

	// TODO: update the below to use the environment variable, and the general
	// structure seen in the create command's help text.
	root.Use = "kn func"
	var err error
	root.Example, err = replaceNameInTemplate("kn func", "example")
	if err != nil {
		root.Example = "Usage could not be loaded"
	}
	return root, err
}

// When the code is loaded into memory upon invocation, the cobra/viper packages
// are invoked to gather system context.  This includes reading the configuration
// file, environment variables, and parsing the command flags.
func init() {
	// read in environment variables that match
	viper.AutomaticEnv()

	verbose := viper.GetBool("verbose")

	// Populate the `verbose` flag with the value of --verbose, if provided,
	// which thus overrides both the default and the value read in from the
	// config file (i.e. flags always take highest precidence).
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", verbose, "print verbose logs")
	err := viper.BindPFlag("verbose", root.PersistentFlags().Lookup("verbose"))
	if err != nil {
		panic(err)
	}

	// Override the --version template to match the output format from the
	// version subcommand: nothing but the version.
	root.SetVersionTemplate(`{{printf "%s\n" .Version}}`)

	// Prefix all environment variables with "FUNC_" to avoid collisions with other apps.
	viper.SetEnvPrefix("func")
}

// Execute the command tree by executing the root command, which runs
// according to the context defined by:  the optional config file,
// Environment Variables, command arguments and flags.
func Execute(ctx context.Context) {
	// Sets version to a string partially populated by compile-time flags.
	root.Version = version.String()
	// Execute the root of the command tree.
	if err := root.ExecuteContext(ctx); err != nil {
		if ctx.Err() != nil {
			os.Exit(130)
			return
		}
		// Errors are printed to STDERR output and the process exits with code of 1.
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Helpers
// ------------------------------------------

// interactiveTerminal returns whether or not the currently attached process
// terminal is interactive.  Used for determining whether or not to
// interactively prompt the user to confirm default choices, etc.
func interactiveTerminal() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && ((fi.Mode() & os.ModeCharDevice) != 0)
}

// cwd returns the current working directory or exits 1 printing the error.
func cwd() (cwd string) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to determine current working directory: %v", err)
		os.Exit(1)
	}
	return cwd
}

// The anme of the repositories directory within config dir (usually ~/.config)
const repositoriesDirName = "repositories"

// repositoriesPath is the effective path to the optional repositories directory
// used for extensible language packs.
func repositoriesPath() string {
	return filepath.Join(fn.ConfigPath(), repositoriesDirName)
}

// bindFunc which conforms to the cobra PreRunE method signature
type bindFunc func(*cobra.Command, []string) error

// bindEnv returns a bindFunc that binds env vars to the namd flags.
func bindEnv(flags ...string) bindFunc {
	return func(cmd *cobra.Command, args []string) (err error) {
		for _, flag := range flags {
			if err = viper.BindPFlag(flag, cmd.Flags().Lookup(flag)); err != nil {
				return
			}
		}
		return
	}
}

type functionOverrides struct {
	Image     string
	Namespace string
	Builder   string
}

// functionWithOverrides sets the namespace and image strings for the
// Function project at root, if provided, and returns the Function
// configuration values.
// Please note that When this function is called, the overrides are not persisted.
func functionWithOverrides(root string, overrides functionOverrides) (f fn.Function, err error) {
	f, err = fn.NewFunction(root)
	if err != nil {
		return
	}

	overrideMapping := []struct {
		src  string
		dest *string
	}{
		{overrides.Builder, &f.Builder},
		{overrides.Image, &f.Image},
		{overrides.Namespace, &f.Namespace},
	}

	for _, m := range overrideMapping {
		if m.src != "" {
			*m.dest = m.src
		}
	}

	return
}

// deriveName returns the explicit value (if provided) or attempts to derive
// from the given path.  Path is defaulted to current working directory, where
// a Function configuration, if it exists and contains a name, is used.
func deriveName(explicitName string, path string) string {
	// If the name was explicitly provided, use it.
	if explicitName != "" {
		return explicitName
	}

	// If the directory at path contains an initialized Function, use the name therein
	f, err := fn.NewFunction(path)
	if err == nil && f.Name != "" {
		return f.Name
	}

	return ""
}

// deriveNameAndAbsolutePathFromPath returns resolved Function name and absolute path
// to the Function project root. The input parameter path could be one of:
// 'relative/path/to/foo', '/absolute/path/to/foo', 'foo' or ''
func deriveNameAndAbsolutePathFromPath(path string) (string, string) {
	var absPath string

	// If path is not specifed, we would like to use current working dir
	if path == "" {
		path = cwd()
	}

	// Expand the passed Function name to its absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", ""
	}

	// Get the name of the Function, which equals to name of the current directory
	pathParts := strings.Split(strings.TrimRight(path, string(os.PathSeparator)), string(os.PathSeparator))
	return pathParts[len(pathParts)-1], absPath
}

// deriveImage returns the same image name which will be used if no explicit
// image is provided.  I.e. derived from the configured registry (registry
// plus username) and the Function's name.
//
// This is calculated preemptively here in the CLI (prior to invoking the
// client), only in order to provide information to the user via the prompt.
// The client will calculate this same value if the image override is not
// provided.
//
// Derivation logic:
// deriveImage attempts to arrive at a final, full image name:
//   format:  [registry]/[username]/[FunctionName]:[tag]
//   example: quay.io/myname/my.function.name:tag.
//
// Registry can optionally be omitted, in which case DefaultRegistry
// will be prepended.
//
// If the image flag is provided, this value is used directly (the user supplied
// --image or $FUNC_IMAGE).  Otherwise, the Function at 'path' is loaded, and
// the Image name therein is used (i.e. it was previously calculated).
// Finally, the default registry is used, which is prepended to the Function
// name, and appended with ':latest':
func deriveImage(explicitImage, defaultRegistry, path string) string {
	if explicitImage != "" {
		return explicitImage // use the explicit value provided.
	}
	f, err := fn.NewFunction(path)
	if err != nil {
		return "" // unable to derive due to load error (uninitialized?)
	}
	if f.Image != "" {
		return f.Image // use value previously provided or derived.
	}
	derivedValue, _ := fn.DerivedImage(path, defaultRegistry)
	return derivedValue // Use the func system's derivation logic.
}

func envFromCmd(cmd *cobra.Command) (*util.OrderedMap, []string, error) {
	if cmd.Flags().Changed("env") {
		env, err := cmd.Flags().GetStringArray("env")
		if err != nil {
			return nil, []string{}, fmt.Errorf("Invalid --env: %w", err)
		}
		return util.OrderedMapAndRemovalListFromArray(env, "=")
	}
	return util.NewOrderedMap(), []string{}, nil
}

func mergeEnvs(envs fn.Envs, envToUpdate *util.OrderedMap, envToRemove []string) (fn.Envs, error) {
	updated := sets.NewString()

	for i := range envs {
		if envs[i].Name != nil {
			value, present := envToUpdate.GetString(*envs[i].Name)
			if present {
				envs[i].Value = &value
				updated.Insert(*envs[i].Name)
			}
		}
	}

	it := envToUpdate.Iterator()
	for name, value, ok := it.NextString(); ok; name, value, ok = it.NextString() {
		if !updated.Has(name) {
			n := name
			v := value
			envs = append(envs, fn.Env{Name: &n, Value: &v})
		}
	}

	for _, name := range envToRemove {
		for i, envVar := range envs {
			if *envVar.Name == name {
				envs = append(envs[:i], envs[i+1:]...)
				break
			}
		}
	}

	errMsg := fn.ValidateEnvs(envs)
	if len(errMsg) > 0 {
		return fn.Envs{}, fmt.Errorf(strings.Join(errMsg, "\n"))
	}

	return envs, nil
}
