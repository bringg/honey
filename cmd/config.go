package cmd

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rclone/rclone/fs/config/flags"
	"github.com/rclone/rclone/fs/rc"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bringg/honey/pkg/config"
)

var (
	configObscure   bool
	configNoObscure bool

	configCommand = &cobra.Command{
		Use:   "config",
		Short: `Enter an interactive configuration session.`,
		Long: `Enter an interactive configuration session where you can setup new
backends and manage existing ones.
`,
		Run: func(command *cobra.Command, args []string) {
			CheckArgs(0, 0, command, args)
			config.EditConfig(context.Background())
		},
	}

	configCreateCommand = &cobra.Command{
		Use:   "create `name` `type` [`key` `value`]*",
		Short: `Create a new backend with name, type and options.`,
		Long: `
Create a new backend of ` + "`name`" + ` with ` + "`type`" + ` and options.  The options
should be passed in pairs of ` + "`key` `value`" + `.

For example to make a k8s backend of name mybackend using auto config
you would do:

    honey config create mybackend k8s context minikube

    honey config create c1 aws region us-east-1
`,
		RunE: func(command *cobra.Command, args []string) error {
			CheckArgs(2, 256, command, args)
			in, err := argsToMap(args[2:])
			if err != nil {
				return err
			}

			err = config.CreateBackend(context.Background(), args[0], args[1], in, configObscure, configNoObscure)
			if err != nil {
				return err
			}

			config.ShowBackend(args[0])

			return nil
		},
	}
)

func init() {
	for _, cmdFlags := range []*pflag.FlagSet{configCreateCommand.Flags() /* , configUpdateCommand.Flags() */} {
		flags.BoolVarP(cmdFlags, &configObscure, "obscure", "", false, "Force any passwords to be obscured.")
		flags.BoolVarP(cmdFlags, &configNoObscure, "no-obscure", "", false, "Force any passwords not to be obscured.")
	}
}

// This takes a list of arguments in key value key value form and
// converts it into a map
func argsToMap(args []string) (out rc.Params, err error) {
	if len(args)%2 != 0 {
		return nil, errors.New("found key without value")
	}

	out = rc.Params{}
	// Set the config
	for i := 0; i < len(args); i += 2 {
		out[args[i]] = args[i+1]
	}

	return out, nil
}
