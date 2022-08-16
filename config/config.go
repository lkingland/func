package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"
)

const (
	// Filename into which Config is serialized
	Filename = "config.yaml"

	// DefaultConfigPath is used in the unlikely event that
	// the user has no home directory (no ~), there is no
	// XDG_CONFIG_HOME set
	DefaultConfigPath = ".config/func"

	// DefaultLanguage is intentionaly undefined.
	DefaultLanguage = ""
)

type Config struct {
	// Language runtime to use by default for new functions.
	Language string `yaml:"language"`

	// Confirm prompts by default on all commands which support them.
	Confirm bool `yaml:"confirm"`
}

// New Config struct with all members set to static defaults.  See NewDefaults
// for one which further takes into account the optional config file.
func New() Config {
	return Config{
		Language: DefaultLanguage,
	}
}

// Creates a new config populated by global defaults as defined by the
// config file located in .Path() (the global func settings path, which is
//  usually ~/.config/func)
func NewDefault() (cfg Config, err error) {
	// a config populated only by static defaults
	cfg = New()       // cfg now populated by static defaults
	p := ConfigPath() // applies ~/.config/func/config.yaml if it exists
	if _, err = os.Stat(p); err != nil {
		if os.IsNotExist(err) {
			err = nil // config file is not required
		}
		return
	}
	bb, err := os.ReadFile(p)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(bb, &cfg) // cfg now has applied config.yaml
	return
}

// Load the config exactly as it exists at path.
func Load(path string) (c Config, err error) {
	if _, err = os.Stat(path); err != nil {
		return
	}
	bb, err := os.ReadFile(path)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(bb, &c)
	return
}

// Save the config to the given path
func (c Config) Save(path string) (err error) {
	var bb []byte
	if bb, err = yaml.Marshal(&c); err != nil {
		return
	}
	return ioutil.WriteFile(path, bb, 0644)
}

// Path is derived in the following order, from lowest
// to highest precedence.
// 1.  The static default is DefaultConfigPath (./.config/func)
// 2.  ~/.config/func if it exists (can be expanded: user has a home dir)
// 3.  The value of $XDG_CONFIG_PATH/func if the environment variable exists.
// The path is created if it does not already exist.
func Path() (path string) {
	path = DefaultConfigPath

	// ~/.config/func is the default if ~ can be expanded
	if home, err := homedir.Expand("~"); err == nil {
		path = filepath.Join(home, ".config", "func")
	}

	// 'XDG_CONFIG_HOME/func' takes precidence if defined
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		path = filepath.Join(xdg, "func")
	}

	mkdir(path)
	return
}

// ConfigPath returns the full path to look for a config file.
func ConfigPath() string {
	// TODO: this will likely end up including considering environment variable
	// FUNC_CONFIG_FILE which will allow one to explicitly set a config file
	// location.  At present a custom XDG_CONFIG_HOME is all needed.

	// usually ~/.config/func/config.yaml
	return filepath.Join(Path(), Filename)
}

func mkdir(path string) {
	if err := os.MkdirAll(path, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating '%v': %v", path, err)
		debug.PrintStack()
	}
}
