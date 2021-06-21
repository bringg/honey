package place

import (
	"context"

	"github.com/rclone/rclone/fs/config/configmap"
)

// Constants Option.Hide
const (
	OptionHideCommandLine OptionVisibility = 1 << iota
	OptionHideConfigurator
	OptionHideBoth = OptionHideCommandLine | OptionHideConfigurator
)

type (
	// Backend _
	Backend interface {
		Name() string
		CacheKeyName(pattern string) string
		List(ctx context.Context, backendName string, pattern string) (Printable, error)
	}

	// Commander is an interface to wrap the Command function
	Commander interface {
		// Command the backend to run a named command
		//
		// The command run is name
		// args may be used to read arguments from
		// opts may be used to read optional arguments from
		//
		// The result should be capable of being JSON encoded
		// If it is a string or a []string it will be shown to the user
		// otherwise it will be JSON encoded and shown to the user like that
		Command(ctx context.Context, name string, arg []string, opt map[string]string) (interface{}, error)
	}

	Model struct {
		ID          string `json:"id"`
		BackendName string `json:"backend_name" mapstructure:"backend_name"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		Status      string `json:"status"`
		PrivateIP   string `json:"private_ip" mapstructure:"private_ip"`
		PublicIP    string `json:"public_ip" mapstructure:"public_ip"`
	}

	// Instance _
	Instance struct {
		Model `mapstructure:",squash"`
		Raw   interface{}
	}

	Printable []*Instance

	// RegInfo _
	RegInfo struct {
		// Name of this backend
		Name string
		// Description of this fs - defaults to Name
		Description string
		// Prefix for command line flags for this fs - defaults to Name if not set
		Prefix string

		NewBackend func(ctx context.Context, config configmap.Mapper) (Backend, error) `json:"-"`
		// Function to call to help with config
		Config func(ctx context.Context, name string, config configmap.Mapper) `json:"-"`
		// Options for the Backend configuration
		Options Options
		// The command help, if any
		CommandHelp []CommandHelp
	}

	// Options is a slice of configuration Option for a backend
	Options []Option

	// Option _
	Option struct {
		Name       string           // name of the option in snake_case
		Help       string           // Help, the first line only is used for the command line help
		Provider   string           // Set to filter on provider
		Default    interface{}      // default value, nil => ""
		Value      interface{}      // value to be set by flags
		Examples   OptionExamples   `json:",omitempty"` // config examples
		ShortOpt   string           // the short option for this if required
		Hide       OptionVisibility // set this to hide the config from the configurator or the command line
		Required   bool             // this option is required
		IsPassword bool             // set if the option is a passwords
		NoPrefix   bool             // set if the option for this should not use the backend prefix
		Advanced   bool             // set if this is an advanced config option
	}

	// OptionVisibility controls whether the options are visible in the
	// configurator or the command line.
	OptionVisibility byte

	// OptionExamples is a slice of examples
	OptionExamples []OptionExample

	// OptionExample describes an example for an Option
	OptionExample struct {
		Value    string
		Help     string
		Provider string
	}

	// CommandHelp describes a single backend Command
	//
	// These are automatically inserted in the docs
	CommandHelp struct {
		Name  string            // Name of the command, e.g. "link"
		Short string            // Single line description
		Long  string            // Long multi-line description
		Opts  map[string]string // maps option name to a single line help
	}

	// A configmap.Getter to read either the default value or the set
	// value from the RegInfo.Options
	regInfoValues struct {
		backendInfo *RegInfo
		useDefault  bool
	}

	// A configmap.Getter to read from the config file
	getConfigFile string

	// A configmap.Setter to read from the config file
	setConfigFile string
)
