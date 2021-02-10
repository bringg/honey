package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const completionDesc = `
Generate autocompletion scripts for Honey for the specified shell.
`
const bashCompDesc = `
Generate the autocompletion script for Honey for the bash shell.

To load completions in your current shell session:
$ source <(honey completion bash)

To load completions for every new session, execute once:
Linux:
  $ honey completion bash > /etc/bash_completion.d/honey
MacOS:
  $ honey completion bash > /usr/local/etc/bash_completion.d/honey
`

const zshCompDesc = `
Generate the autocompletion script for Honey for the zsh shell.

To load completions in your current shell session:
$ source <(honey completion zsh)

To load completions for every new session, execute once:
$ honey completion zsh > "${fpath[1]}/_honey"
`

const fishCompDesc = `
Generate the autocompletion script for Honey for the fish shell.

To load completions in your current shell session:
$ honey completion fish | source

To load completions for every new session, execute once:
$ honey completion fish > ~/.config/fish/completions/honey.fish

You will need to start a new shell for this setup to take effect.
`

const (
	noDescFlagName = "no-descriptions"
	noDescFlagText = "disable completion descriptions"
)

var disableCompDescriptions bool

func newCompletionCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "generate autocompletion scripts for the specified shell",
		Long:  completionDesc,
		//Args:  require.NoArgs,
	}

	bash := &cobra.Command{
		Use:   "bash",
		Short: "generate autocompletion script for bash",
		Long:  bashCompDesc,
		//Args:                  require.NoArgs,
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     noCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompletionBash(out, cmd)
		},
	}

	zsh := &cobra.Command{
		Use:   "zsh",
		Short: "generate autocompletion script for zsh",
		Long:  zshCompDesc,
		//Args:                  require.NoArgs,
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     noCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompletionZsh(out, cmd)
		},
	}
	zsh.Flags().BoolVar(&disableCompDescriptions, noDescFlagName, false, noDescFlagText)

	fish := &cobra.Command{
		Use:   "fish",
		Short: "generate autocompletion script for fish",
		Long:  fishCompDesc,
		//Args:                  require.NoArgs,
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     noCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompletionFish(out, cmd)
		},
	}
	fish.Flags().BoolVar(&disableCompDescriptions, noDescFlagName, false, noDescFlagText)

	cmd.AddCommand(bash, zsh, fish)

	return cmd
}

func runCompletionBash(out io.Writer, cmd *cobra.Command) error {
	err := cmd.Root().GenBashCompletion(out)

	// In case the user renamed the honey binary (e.g., to be able to run
	// both honey2 and honey3), we hook the new binary name to the completion function
	if binary := filepath.Base(os.Args[0]); binary != "honey" {
		renamedBinaryHook := `
# Hook the command used to generate the completion script
# to the honey completion function to handle the case where
# the user renamed the honey binary
if [[ $(type -t compopt) = "builtin" ]]; then
    complete -o default -F __start_honey %[1]s
else
    complete -o default -o nospace -F __start_honey %[1]s
fi
`
		fmt.Fprintf(out, renamedBinaryHook, binary)
	}

	return err
}

func runCompletionZsh(out io.Writer, cmd *cobra.Command) error {
	var err error
	if disableCompDescriptions {
		err = cmd.Root().GenZshCompletionNoDesc(out)
	} else {
		err = cmd.Root().GenZshCompletion(out)
	}

	// In case the user renamed the honey binary (e.g., to be able to run
	// both honey2 and honey3), we hook the new binary name to the completion function
	if binary := filepath.Base(os.Args[0]); binary != "honey" {
		renamedBinaryHook := `
# Hook the command used to generate the completion script
# to the honey completion function to handle the case where
# the user renamed the honey binary
compdef _honey %[1]s
`
		fmt.Fprintf(out, renamedBinaryHook, binary)
	}

	// Cobra doesn't source zsh completion file, explicitly doing it here
	fmt.Fprintf(out, "compdef _honey honey")

	return err
}

func runCompletionFish(out io.Writer, cmd *cobra.Command) error {
	return cmd.Root().GenFishCompletion(out, !disableCompDescriptions)
}

// Function to disable file completion
func noCompletions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}
