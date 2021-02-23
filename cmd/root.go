package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/rclone/rclone/fs/config/flags"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/bringg/honey/pkg/config/configflags"
	"github.com/bringg/honey/pkg/place"
	"github.com/bringg/honey/pkg/place/operations"
	"github.com/bringg/honey/pkg/place/printers"
)

const bannerTmp = `
88 Build by: %s
88
88 Where’s my instance,
88,dPPYba,   ,adPPYba,  8b,dPPYba,   ,adPPYba, 8b       d8
88P'    "8a a8"     "8a 88P'   '"8a a8P_____88 '8b     d8'
88       88 8b       d8 88       88 8PP"""""""  '8b   d8'
88       88 "8a,   ,a8" 88       88 "8b,   ,aa   '8b,d8'
88       88  '"YbbdP"'  88       88  '"Ybbd8"'     Y88'
                                                   d8'
                                                  d8' ?
Version: %s
Commit: %s
Date: %s
`

var (
	version      = "development"
	commit       = "development"
	builtBy      = "shareed2k"
	filter       = ""
	date         = time.Now().String()
	banner       = fmt.Sprintf(color.GreenString(bannerTmp)+"\n", builtBy, version, commit, date)
	backendFlags map[string]struct{}
	// to filter the flags with
	flagsRe *regexp.Regexp

	Root = &cobra.Command{
		Use:           "honey",
		SilenceUsage:  true,
		SilenceErrors: true,
		Short:         "DevOps tool to help find an instance in sea of clouds",
		Version:       version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Setup shell completion for the k8s-namespace flag
			if err := cmd.RegisterFlagCompletionFunc("k8s-namespace", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				getter := genericclioptions.NewConfigFlags(true)
				flag := cmd.Flag("k8s-context").Value.String()
				if flag != "" {
					getter.Context = &flag
				}

				factory := cmdutil.NewFactory(getter)
				if client, err := factory.KubernetesClientSet(); err == nil {
					// Choose a long enough timeout that the user notices somethings is not working
					// but short enough that the user is not made to wait very long
					to := int64(3)
					cobra.CompDebugln(fmt.Sprintf("About to call kube client for namespaces with timeout of: %d", to), true)

					nsNames := []string{}
					if namespaces, err := client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{TimeoutSeconds: &to}); err == nil {
						for _, ns := range namespaces.Items {
							if strings.HasPrefix(ns.Name, toComplete) {
								nsNames = append(nsNames, ns.Name)
							}
						}
						return nsNames, cobra.ShellCompDirectiveNoFileComp
					}
				}
				return nil, cobra.ShellCompDirectiveDefault
			}); err != nil {
				return err
			}

			// Setup shell completion for the kube-context flag
			return cmd.RegisterFlagCompletionFunc("k8s-context", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				cobra.CompDebugln("About to get the different kube-contexts", true)

				loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
				/* if len("") > 0 {
					loadingRules = &clientcmd.ClientConfigLoadingRules{ExplicitPath: ""}
				} */
				if config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
					loadingRules,
					&clientcmd.ConfigOverrides{}).RawConfig(); err == nil {
					ctxs := []string{}
					for name := range config.Contexts {
						if strings.HasPrefix(name, toComplete) {
							ctxs = append(ctxs, name)
						}
					}
					return ctxs, cobra.ShellCompDirectiveNoFileComp
				}
				return nil, cobra.ShellCompDirectiveNoFileComp
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				re, err := regexp.Compile(args[0])
				if err != nil {
					return errors.Wrap(err, "Failed to compile flags regexp")
				}

				flagsRe = re
				filter = args[0]
			}

			ctx := context.TODO()
			ci := place.GetConfig(ctx)

			backends, err := ci.Backends()
			if err != nil {
				return err
			}

			if len(backends) == 0 {
				return errors.New("oops you must specify at least one backend")
			}

			instances, err := operations.Find(context.TODO(), backends, filter)
			if err != nil {
				return err
			}

			return printers.Print(&printers.PrintInput{
				Data:    instances,
				Format:  ci.OutFormat,
				NoColor: ci.NoColor,
			})
		},
	}

	// Show the backends
	helpBackends = &cobra.Command{
		Use:   "backends",
		Short: "List the backends available",
		Run: func(command *cobra.Command, args []string) {
			showBackends()
		},
	}

	// root help command
	helpCommand = &cobra.Command{
		Use:   "help",
		Short: Root.Short,
		Long:  Root.Long,
		Run: func(command *cobra.Command, args []string) {
			Root.SetOutput(os.Stdout)
			_ = Root.Usage()
		},
	}

	// Show a single backend
	helpBackend = &cobra.Command{
		Use:   "backend <name>",
		Short: "List full info about a backend",
		Run: func(command *cobra.Command, args []string) {
			if len(args) == 0 {
				Root.SetOutput(os.Stdout)
				_ = command.Usage()
				return
			}

			showBackend(args[0])
		},
	}

	// Show the flags
	helpFlags = &cobra.Command{
		Use:   "flags [<regexp to match>]",
		Short: "Show the global flags for honey",
		Run: func(command *cobra.Command, args []string) {
			if len(args) > 0 {
				re, err := regexp.Compile(args[0])
				if err != nil {
					log.Fatalf("Failed to compile flags regexp: %v", err)
				}
				flagsRe = re
			}

			Root.SetOutput(os.Stdout)

			_ = command.Usage()
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the Root.
func Execute() {
	setupRootCommand()
	addBackendFlags()

	if err := Root.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	Root.SetUsageTemplate(banner + usageTemplate)
	Root.SetHelpCommand(helpCommand)

	Root.AddCommand(newCompletionCmd(os.Stdout))
	Root.AddCommand(helpCommand)
	Root.AddCommand(configCommand)
	Root.AddCommand(obscureCmd)
	Root.AddCommand(serveCmd)

	helpCommand.AddCommand(helpFlags)
	helpCommand.AddCommand(helpBackends)
	helpCommand.AddCommand(helpBackend)
	configCommand.AddCommand(configCreateCommand)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Finish parsing any command line flags
	configflags.SetFlags()
}

// addBackendFlags creates flags for all the backend options
func addBackendFlags() {
	backendFlags = map[string]struct{}{}
	for _, backendInfo := range place.Registry {
		done := map[string]struct{}{}
		for i := range backendInfo.Options {
			opt := &backendInfo.Options[i]
			// Skip if done already (e.g. with Backend options)
			if _, doneAlready := done[opt.Name]; doneAlready {
				continue
			}
			done[opt.Name] = struct{}{}
			// Make a flag from each option
			name := opt.FlagName(backendInfo.Prefix)
			found := pflag.CommandLine.Lookup(name) != nil
			if !found {
				// Take first line of help only
				help := strings.TrimSpace(opt.Help)
				if nl := strings.IndexRune(help, '\n'); nl >= 0 {
					help = help[:nl]
				}
				help = strings.TrimSpace(help)
				flag := flags.VarPF(pflag.CommandLine, opt, name, opt.ShortOpt, help)
				if _, isBool := opt.Default.(bool); isBool {
					flag.NoOptDefVal = "true"
				}
				// Hide on the command line if requested
				if opt.Hide&place.OptionHideCommandLine != 0 {
					flag.Hidden = true
				}
				backendFlags[name] = struct{}{}
			} else {
				log.Errorf("Not adding duplicate flag --%s", name)
			}
		}
	}
}

// show all the backends
func showBackends() {
	fmt.Printf("All honey backends:\n\n")

	for _, backend := range place.Registry {
		fmt.Printf("  %-12s %s\n", backend.Prefix, backend.Description)
	}

	fmt.Printf("\nTo see more info about a particular backend use:\n")
	fmt.Printf("  honey help backend <name>\n")
}

// show a single backend
func showBackend(name string) {
	backend, err := place.Find(name)
	if err != nil {
		log.Fatal(err)
	}
	var standardOptions, advancedOptions place.Options
	done := map[string]struct{}{}
	for _, opt := range backend.Options {
		// Skip if done already (e.g. with Provider options)
		if _, doneAlready := done[opt.Name]; doneAlready {
			continue
		}
		if opt.Advanced {
			advancedOptions = append(advancedOptions, opt)
		} else {
			standardOptions = append(standardOptions, opt)
		}
	}
	optionsType := "standard"
	for _, opts := range []place.Options{standardOptions, advancedOptions} {
		if len(opts) == 0 {
			optionsType = "advanced"
			continue
		}
		fmt.Printf("### %s Options\n\n", strings.Title(optionsType))
		fmt.Printf("Here are the %s options specific to %s (%s).\n\n", optionsType, backend.Name, backend.Description)
		optionsType = "advanced"
		for _, opt := range opts {
			done[opt.Name] = struct{}{}
			shortOpt := ""
			if opt.ShortOpt != "" {
				shortOpt = fmt.Sprintf(" / -%s", opt.ShortOpt)
			}
			fmt.Printf("#### --%s%s\n\n", opt.FlagName(backend.Prefix), shortOpt)
			fmt.Printf("%s\n\n", opt.Help)
			fmt.Printf("- Config:      %s\n", opt.Name)
			fmt.Printf("- Env Var:     %s\n", opt.EnvVarName(backend.Prefix))
			fmt.Printf("- Type:        %s\n", opt.Type())
			fmt.Printf("- Default:     %s\n", quoteString(opt.GetValue()))
			if len(opt.Examples) > 0 {
				fmt.Printf("- Examples:\n")
				for _, ex := range opt.Examples {
					fmt.Printf("    - %s\n", quoteString(ex.Value))
					for _, line := range strings.Split(ex.Help, "\n") {
						fmt.Printf("        - %s\n", line)
					}
				}
			}
			fmt.Printf("\n")
		}
	}
}

// nolint
func quoteString(v interface{}) string {
	switch v.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	}

	return fmt.Sprint(v)
}

func setupRootCommand() {
	ci := place.GetConfig(context.Background())
	configflags.AddFlags(ci, pflag.CommandLine)

	flags.StringVarP(pflag.CommandLine, &filter, "filter", "f", "", "instance name filter")

	cobra.AddTemplateFunc("showLocalFlags", func(cmd *cobra.Command) bool {
		// Don't show local flags (which are the global ones on the root) on "honey" and
		// "honey help" (which shows the global help)
		return cmd.CalledAs() != "honey" && cmd.CalledAs() != ""
	})

	cobra.AddTemplateFunc("backendFlags", func(cmd *cobra.Command, include bool) *pflag.FlagSet {
		backendFlagSet := pflag.NewFlagSet("Backend Flags", pflag.ExitOnError)
		cmd.InheritedFlags().VisitAll(func(flag *pflag.Flag) {
			matched := flagsRe == nil || flagsRe.MatchString(flag.Name)
			if _, ok := backendFlags[flag.Name]; matched && ok == include {
				backendFlagSet.AddFlag(flag)
			}
		})

		return backendFlagSet
	})

	cobra.AddTemplateFunc("showGlobalFlags", func(cmd *cobra.Command) bool {
		return cmd.CalledAs() == "flags"
	})
	cobra.AddTemplateFunc("showCommands", func(cmd *cobra.Command) bool {
		return cmd.CalledAs() != "flags"
	})
}

var usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if and (showCommands .) .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if and (showLocalFlags .) .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if and (showGlobalFlags .) .HasAvailableInheritedFlags}}

Global Flags:
{{(backendFlags . false).FlagUsages | trimTrailingWhitespaces}}

Backend Flags:
{{(backendFlags . true).FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}

Use "honey [command] --help" for more information about a command.
Use "honey help flags" for to see the global flags.
Use "honey help backends" for a list of supported services.
`

// CheckArgs checks there are enough arguments and prints a message if not
func CheckArgs(MinArgs, MaxArgs int, cmd *cobra.Command, args []string) {
	if len(args) < MinArgs {
		_ = cmd.Usage()
		log.Fatalf("Command %s needs %d arguments minimum: you provided %d non flag arguments: %q\n", cmd.Name(), MinArgs, len(args), args)
	} else if len(args) > MaxArgs {
		_ = cmd.Usage()
		log.Fatalf("Command %s needs %d arguments maximum: you provided %d non flag arguments: %q\n", cmd.Name(), MaxArgs, len(args), args)
	}
}
