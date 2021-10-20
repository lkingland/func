package cmd

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/ory/viper"
	"github.com/spf13/cobra"
	"knative.dev/client/pkg/util"

	fn "knative.dev/kn-plugin-func"
	"knative.dev/kn-plugin-func/buildpacks"
	"knative.dev/kn-plugin-func/docker"
	"knative.dev/kn-plugin-func/knative"
	"knative.dev/kn-plugin-func/progress"
)

func init() {
	root.AddCommand(NewDeployCmd(newDeployClient))
}

func newDeployClient(cfg deployConfig) (*fn.Client, error) {
	listener := progress.New()

	builder := buildpacks.NewBuilder()

	credentialsProvider := docker.NewCredentialsProvider(
		newCredentialsCallback(),
		docker.CheckAuth,
		newChooseHelperCallback())
	pusher, err := docker.NewPusher(
		docker.WithCredentialsProvider(credentialsProvider),
		docker.WithProgressListener(listener))
	if err != nil {
		return nil, err
	}

	deployer, err := knative.NewDeployer(cfg.Namespace)
	if err != nil {
		return nil, err
	}

	listener.Verbose = cfg.Verbose
	builder.Verbose = cfg.Verbose
	pusher.Verbose = cfg.Verbose
	deployer.Verbose = cfg.Verbose

	return fn.New(
		fn.WithProgressListener(listener),
		fn.WithBuilder(builder),
		fn.WithPusher(pusher),
		fn.WithDeployer(deployer),
		fn.WithRegistry(cfg.Registry), // for deriving name when no --image value
		fn.WithVerbose(cfg.Verbose),
	), nil
}

type deployClientFn func(deployConfig) (*fn.Client, error)

func NewDeployCmd(clientFn deployClientFn) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy a function",
		Long: `Deploy a function

Builds a container image for the function and deploys it to the connected Knative enabled cluster. 
The function is picked up from the project in the current directory or from the path provided
with --path.
If not already configured, either --registry or --image has to be provided and is then stored 
in the configuration file.

If the function is already deployed, it is updated with a new container image
that is pushed to an image registry, and finally the function's Knative service is updated.
`,
		Example: `
# Build and deploy the function from the current directory's project. The image will be
# pushed to "quay.io/myuser/<function name>" and deployed as Knative service with the 
# same name as the function to the currently connected cluster.
kn func deploy --registry quay.io/myuser

# Same as above but using a full image name, that will create a Knative service "myfunc" in 
# the namespace "myns"
kn func deploy --image quay.io/myuser/myfunc -n myns
`,
		SuggestFor: []string{"delpoy", "deplyo"},
		PreRunE:    bindEnv("image", "namespace", "path", "registry", "confirm", "build"),
	}

	cmd.Flags().BoolP("confirm", "c", false, "Prompt to confirm all configuration options (Env: $FUNC_CONFIRM)")
	cmd.Flags().StringArrayP("env", "e", []string{}, "Environment variable to set in the form NAME=VALUE. "+
		"You may provide this flag multiple times for setting multiple environment variables. "+
		"To unset, specify the environment variable name followed by a \"-\" (e.g., NAME-).")
	cmd.Flags().StringP("image", "i", "", "Full image name in the form [registry]/[namespace]/[name]:[tag] (optional). This option takes precedence over --registry (Env: $FUNC_IMAGE)")
	cmd.Flags().StringP("namespace", "n", "", "Namespace of the function to undeploy. By default, the namespace in func.yaml is used or the actual active namespace if not set in the configuration. (Env: $FUNC_NAMESPACE)")
	cmd.Flags().StringP("path", "p", cwd(), "Path to the project directory (Env: $FUNC_PATH)")
	cmd.Flags().StringP("registry", "r", "", "Registry + namespace part of the image to build, ex 'quay.io/myuser'.  The full image name is automatically determined based on the local directory name. If not provided the registry will be taken from func.yaml (Env: $FUNC_REGISTRY)")
	cmd.Flags().BoolP("build", "b", true, "Build the image before deploying (Env: $FUNC_BUILD)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runDeploy(cmd, args, clientFn)
	}

	return cmd
}

func runDeploy(cmd *cobra.Command, _ []string, clientFn deployClientFn) (err error) {
	config, err := newDeployConfig(cmd)
	if err != nil {
		return err
	}

	config, err = config.Prompt()
	if err != nil {
		if err == terminal.InterruptErr {
			return nil
		}
		return
	}

	function, err := functionWithOverrides(config.Path, functionOverrides{Namespace: config.Namespace, Image: config.Image})
	if err != nil {
		return
	}

	function.Envs, err = mergeEnvs(function.Envs, config.EnvToUpdate, config.EnvToRemove)
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

	// Deafult conig namespace is the function's namespace
	if config.Namespace == "" {
		config.Namespace = function.Namespace
	}

	// if registry was not changed via command line flag meaning it's empty
	// keep the same registry by setting the config.registry to empty otherwise
	// trust viper to override the env variable with the given flag if both are specified
	if regFlag, _ := cmd.Flags().GetString("registry"); regFlag == "" {
		config.Registry = ""
	}

	client, err := clientFn(config)
	if err != nil {
		if err == terminal.InterruptErr {
			return nil
		}
		return err
	}

	if config.Build {
		if err := client.Build(cmd.Context(), config.Path); err != nil {
			return err
		}
	}

	return client.Deploy(cmd.Context(), config.Path)

	// NOTE: Namespace is optional, default is that used by k8s client
	// (for example kubectl usually uses ~/.kube/config)
}

func newCredentialsCallback() func(registry string) (docker.Credentials, error) {
	firstTime := true
	return func(registry string) (docker.Credentials, error) {
		var result docker.Credentials
		var qs = []*survey.Question{
			{
				Name: "username",
				Prompt: &survey.Input{
					Message: "Username:",
				},
				Validate: survey.Required,
			},
			{
				Name: "password",
				Prompt: &survey.Password{
					Message: "Password:",
				},
				Validate: survey.Required,
			},
		}
		if firstTime {
			firstTime = false
			fmt.Printf("Please provide credentials for image registry (%s).\n", registry)
		} else {
			fmt.Println("Incorrect credentials, please try again.")
		}
		err := survey.Ask(qs, &result)
		if err != nil {
			return docker.Credentials{}, err
		}
		return result, nil
	}
}

func newChooseHelperCallback() docker.ChooseCredentialHelperCallback {
	return func(availableHelpers []string) (string, error) {
		if len(availableHelpers) < 1 {
			fmt.Fprintf(os.Stderr, `Credentials will not be saved.
If you would like to save your credentials in the future,
you can install docker credential helper https://github.com/docker/docker-credential-helpers.
`)
			return "", nil
		}

		var resp string
		err := survey.AskOne(&survey.Select{
			Message: "Choose credentials helper",
			Options: append(availableHelpers, "None"),
		}, &resp, survey.WithValidator(survey.Required))
		if err != nil {
			return "", err
		}
		if resp == "None" {
			fmt.Fprintf(os.Stderr, "No helper selected. Credentials will not be saved.\n")
			return "", nil
		}
		return resp, nil
	}
}

type deployConfig struct {
	buildConfig

	// Namespace override for the deployed function.  If provided, the
	// underlying platform will be instructed to deploy the function to the given
	// namespace (if such a setting is applicable; such as for Kubernetes
	// clusters).  If not provided, the currently configured namespace will be
	// used.  For instance, that which would be used by default by `kubectl`
	// (~/.kube/config) in the case of Kubernetes.
	Namespace string

	// Path of the Function implementation on local disk. Defaults to current
	// working directory of the process.
	Path string

	// Verbose logging.
	Verbose bool

	// Confirm: confirm values arrived upon from environment plus flags plus defaults,
	// with interactive prompting (only applicable when attached to a TTY).
	Confirm bool

	// Build the associated Function before deploying.
	Build bool

	// Envs passed via cmd to be added/updated
	EnvToUpdate *util.OrderedMap

	// Envs passed via cmd to removed
	EnvToRemove []string
}

// newDeployConfig creates a buildConfig populated from command flags and
// environment variables; in that precedence.
func newDeployConfig(cmd *cobra.Command) (deployConfig, error) {
	envToUpdate, envToRemove, err := envFromCmd(cmd)
	if err != nil {
		return deployConfig{}, err
	}

	return deployConfig{
		buildConfig: newBuildConfig(),
		Namespace:   viper.GetString("namespace"),
		Path:        viper.GetString("path"),
		Verbose:     viper.GetBool("verbose"), // defined on root
		Confirm:     viper.GetBool("confirm"),
		Build:       viper.GetBool("build"),
		EnvToUpdate: envToUpdate,
		EnvToRemove: envToRemove,
	}, nil
}

// Prompt the user with value of config members, allowing for interaractive changes.
// Skipped if not in an interactive terminal (non-TTY), or if --yes (agree to
// all prompts) was explicitly set.
func (c deployConfig) Prompt() (deployConfig, error) {
	if !interactiveTerminal() || !c.Confirm {
		return c, nil
	}

	var qs = []*survey.Question{
		{
			Name: "registry",
			Prompt: &survey.Input{
				Message: "Registry for Function images:",
				Default: c.buildConfig.Registry,
			},
			Validate: survey.Required,
		},
		{
			Name: "namespace",
			Prompt: &survey.Input{
				Message: "Namespace:",
				Default: c.Namespace,
			},
		},
		{
			Name: "path",
			Prompt: &survey.Input{
				Message: "Project path:",
				Default: c.Path,
			},
			Validate: survey.Required,
		},
	}
	answers := struct {
		Registry  string
		Namespace string
		Path      string
	}{}
	err := survey.Ask(qs, &answers)
	if err != nil {
		return deployConfig{}, err
	}

	dc := deployConfig{
		buildConfig: buildConfig{
			Registry: answers.Registry,
		},
		Namespace: answers.Namespace,
		Path:      answers.Path,
		Verbose:   c.Verbose,
	}

	dc.Image = deriveImage(dc.Image, dc.Registry, dc.Path)

	return dc, nil
}
