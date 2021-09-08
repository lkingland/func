package cmd

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/ory/viper"
	"github.com/spf13/cobra"

	fn "knative.dev/kn-plugin-func"
)

type functionLoader interface {
	Load(path string) (fn.Function, error)
}

type functionSaver interface {
	Save(f fn.Function) error
}

type functionLoaderSaver interface {
	functionLoader
	functionSaver
}

type standardLoaderSaver struct{}

func (s standardLoaderSaver) Load(path string) (fn.Function, error) {
	f, err := fn.NewFunction(path)
	if err != nil {
		return fn.Function{}, err
	}
	if !f.Initialized() {
		return fn.Function{}, fmt.Errorf("the given path '%v' does not contain an initialized function", path)
	}
	return f, nil
}

func (s standardLoaderSaver) Save(f fn.Function) error {
	return f.WriteConfig()
}

var defaultLoaderSaver standardLoaderSaver

func init() {
	root.AddCommand(configCmd)
	configCmd.Flags().StringP("path", "p", cwd(), "Path to the project directory (Env: $FUNC_PATH)")
	configCmd.AddCommand(NewConfigLabelsCmd(defaultLoaderSaver))
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure a function",
	Long: `Configure a function

Interactive propmt that allows configuration of Volume mounts, Environment
variables, and Labels for a function project present in the current directory
or from the directory specified with --path.
`,
	SuggestFor: []string{"cfg", "cofnig"},
	PreRunE:    bindEnv("path"),
	RunE:       runConfigCmd,
}

func runConfigCmd(cmd *cobra.Command, args []string) (err error) {

	function, err := initConfigCommand(args, defaultLoaderSaver)
	if err != nil {
		return
	}

	var qs = []*survey.Question{
		{
			Name: "selectedConfig",
			Prompt: &survey.Select{
				Message: "What do you want to configure?",
				Options: []string{"Environment values", "Volumes", "Labels"},
				Default: "Environment values",
			},
		},
		{
			Name: "selectedOperation",
			Prompt: &survey.Select{
				Message: "What operation do you want to perform?",
				Options: []string{"Add", "Remove", "List"},
				Default: "List",
			},
		},
	}

	answers := struct {
		SelectedConfig    string
		SelectedOperation string
	}{}

	err = survey.Ask(qs, &answers)
	if err != nil {
		if err == terminal.InterruptErr {
			return nil
		}
		return
	}

	switch answers.SelectedOperation {
	case "Add":
		if answers.SelectedConfig == "Volumes" {
			err = runAddVolumesPrompt(cmd.Context(), function)
		} else if answers.SelectedConfig == "Environment values" {
			err = runAddEnvsPrompt(cmd.Context(), function)
		} else if answers.SelectedConfig == "Labels" {
			err = runAddLabelsPrompt(cmd.Context(), function, defaultLoaderSaver)
		}
	case "Remove":
		if answers.SelectedConfig == "Volumes" {
			err = runRemoveVolumesPrompt(function)
		} else if answers.SelectedConfig == "Environment values" {
			err = runRemoveEnvsPrompt(function)
		} else if answers.SelectedConfig == "Labels" {
			err = runRemoveLabelsPrompt(function, defaultLoaderSaver)
		}
	case "List":
		if answers.SelectedConfig == "Volumes" {
			listVolumes(function)
		} else if answers.SelectedConfig == "Environment values" {
			listEnvs(function)
		} else if answers.SelectedConfig == "Labels" {
			listLabels(function)
		}
	}

	return
}

// CLI Configuration (parameters)
// ------------------------------

type configCmdConfig struct {
	Name    string
	Path    string
	Verbose bool
}

func newConfigCmdConfig(args []string) configCmdConfig {
	var name string
	if len(args) > 0 {
		name = args[0]
	}
	return configCmdConfig{
		Name: deriveName(name, viper.GetString("path")),
		Path: viper.GetString("path"),
	}

}

func initConfigCommand(args []string, loader functionLoader) (fn.Function, error) {
	config := newConfigCmdConfig(args)

	function, err := loader.Load(config.Path)
	if err != nil {
		return fn.Function{}, err
	}

	return function, nil
}
