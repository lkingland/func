package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/ory/viper"
	"github.com/spf13/cobra"

	fn "knative.dev/kn-plugin-func"
	"knative.dev/kn-plugin-func/buildpacks"
	"knative.dev/kn-plugin-func/progress"
)

func init() {
	// Add to the root a new "Build" command which obtains an appropriate
	// instance of fn.Client from the given client creator function.
	root.AddCommand(NewBuildCmd(newBuildClient))
}

func newBuildClient(cfg buildConfig) (*fn.Client, error) {
	builder := buildpacks.NewBuilder()
	listener := progress.New()

	builder.Verbose = cfg.Verbose
	listener.Verbose = cfg.Verbose

	return fn.New(
		fn.WithBuilder(builder),
		fn.WithProgressListener(listener),
		fn.WithRegistry(cfg.Registry), // for image name when --image not provided
		fn.WithVerbose(cfg.Verbose)), nil
}

type buildClientFn func(buildConfig) (*fn.Client, error)

func NewBuildCmd(clientFn buildClientFn) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a function project as a container image",
		Long: `Build a function project as a container image

This command builds the function project in the current directory or in the directory
specified by --path. The result will be a container image that is pushed to a registry.
The func.yaml file is read to determine the image name and registry. 
If the project has not already been built, either --registry or --image must be provided 
and the image name is stored in the configuration file.
`,
		Example: `
# Build from the local directory, using the given registry as target.
# The full image name will be determined automatically based on the
# project directory name
kn func build --registry quay.io/myuser

# Build from the local directory, specifying the full image name
kn func build --image quay.io/myuser/myfunc

# Re-build, picking up a previously supplied image name from a local func.yml
kn func build

# Build with a custom buildpack builder
kn func build --builder cnbs/sample-builder:bionic
`,
		SuggestFor: []string{"biuld", "buidl", "built"},
		PreRunE:    bindEnv("image", "path", "builder", "registry", "confirm"),
	}

	cmd.Flags().StringP("builder", "b", "", "Buildpack builder, either an as a an image name or a mapping name.\nSpecified value is stored in func.yaml for subsequent builds.")
	cmd.Flags().BoolP("confirm", "c", false, "Prompt to confirm all configuration options (Env: $FUNC_CONFIRM)")
	cmd.Flags().StringP("image", "i", "", "Full image name in the form [registry]/[namespace]/[name]:[tag] (optional). This option takes precedence over --registry (Env: $FUNC_IMAGE)")
	cmd.Flags().StringP("path", "p", cwd(), "Path to the project directory (Env: $FUNC_PATH)")
	cmd.Flags().StringP("registry", "r", "", "Registry + namespace part of the image to build, ex 'quay.io/myuser'.  The full image name is automatically determined based on the local directory name. If not provided the registry will be taken from func.yaml (Env: $FUNC_REGISTRY)")

	if err := cmd.RegisterFlagCompletionFunc("builder", CompleteBuilderList); err != nil {
		fmt.Println("internal: error while calling RegisterFlagCompletionFunc: ", err)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runBuild(cmd, args, clientFn)
	}

	return cmd
}

func runBuild(cmd *cobra.Command, _ []string, clientFn buildClientFn) (err error) {
	config, err := newBuildConfig().Prompt()
	if err != nil {
		if err == terminal.InterruptErr {
			return nil
		}
		return
	}

	function, err := functionWithOverrides(config.Path, functionOverrides{Builder: config.Builder, Image: config.Image})
	if err != nil {
		return
	}

	// Check if the Function has been initialized
	if !function.Initialized() {
		return fmt.Errorf("the given path '%v' does not contain an initialized function. Please create one at this path before deploying", config.Path)
	}

	// If the Function does not yet have an image name and one was not provided on the command line
	if function.Image == "" {
		//  AND a --registry was not provided, then we need to
		// prompt for a registry from which we can derive an image name.
		if config.Registry == "" {
			fmt.Println("A registry for Function images is required. For example, 'docker.io/tigerteam'.")

			err = survey.AskOne(
				&survey.Input{Message: "Registry for Function images:"},
				&config.Registry, survey.WithValidator(survey.Required))
			if err != nil {
				if err == terminal.InterruptErr {
					return nil
				}
				return
			}
			fmt.Println("Note: building a Function the first time will take longer than subsequent builds")
		}

		// We have the registry, so let's use it to derive the Function image name
		config.Image = deriveImage(config.Image, config.Registry, config.Path)
		function.Image = config.Image
	}

	// All set, let's write changes in the config to the disk
	err = function.WriteConfig()
	if err != nil {
		return
	}

	// if registry was not changed via command line flag meaning it's empty
	// keep the same registry by setting the config.registry to empty otherwise
	// trust viper to override the env variable with the given flag if both are specified
	if regFlag, _ := cmd.Flags().GetString("registry"); regFlag == "" {
		config.Registry = ""
	}

	client, err := clientFn(config)
	if err != nil {
		return err
	}

	return client.Build(cmd.Context(), config.Path)
}

type buildConfig struct {
	// Image name in full, including registry, repo and tag (overrides
	// image name derivation based on Registry and Function Name)
	Image string

	// Path of the Function implementation on local disk. Defaults to current
	// working directory of the process.
	Path string

	// Push the resulting image to the registry after building.
	Push bool

	// Registry at which interstitial build artifacts should be kept.
	// This setting is ignored if Image is specified, which includes the full
	Registry string

	// Verbose logging.
	Verbose bool

	// Confirm: confirm values arrived upon from environment plus flags plus defaults,
	// with interactive prompting (only applicable when attached to a TTY).
	Confirm bool
	Builder string
}

func newBuildConfig() buildConfig {
	return buildConfig{
		Image:    viper.GetString("image"),
		Path:     viper.GetString("path"),
		Registry: viper.GetString("registry"),
		Verbose:  viper.GetBool("verbose"), // defined on root
		Confirm:  viper.GetBool("confirm"),
		Builder:  viper.GetString("builder"),
	}
}

// Prompt the user with value of config members, allowing for interaractive changes.
// Skipped if not in an interactive terminal (non-TTY), or if --confirm false (agree to
// all prompts) was set (default).
func (c buildConfig) Prompt() (buildConfig, error) {
	imageName := deriveImage(c.Image, c.Registry, c.Path)
	if !interactiveTerminal() || !c.Confirm {
		return c, nil
	}

	bc := buildConfig{Verbose: c.Verbose}

	var qs = []*survey.Question{
		{
			Name: "path",
			Prompt: &survey.Input{
				Message: "Project path:",
				Default: c.Path,
			},
			Validate: survey.Required,
		},
		{
			Name: "image",
			Prompt: &survey.Input{
				Message: "Full image name (e.g. quay.io/boson/node-sample):",
				Default: imageName,
			},
			Validate: survey.Required,
		},
	}
	err := survey.Ask(qs, &bc)

	return bc, err
}
