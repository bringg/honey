package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/flags"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/shareed2k/honey/pkg/place"
	"github.com/shareed2k/honey/pkg/place/operations"
)

var (
	Version = "development"
	Commit  = "development"
	Date    = time.Now().String()

	verbose        int
	quiet          bool
	cfgFile        string
	filter         string
	noCache        bool
	outFormat      = "table"
	backendsString string
	backendFlags   map[string]struct{}
	// to filter the flags with
	flagsRe *regexp.Regexp

	Root = &cobra.Command{
		Use:     "honey",
		Short:   "DevOps tool to help find an instance in sea of clouds",
		Version: Version,
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
			}

			backends := fs.CommaSepList{}
			if err := backends.Set(backendsString); err != nil {
				return err
			}

			if len(backends) == 0 {
				return errors.New("oops you must specify at least one backend")
			}

			return operations.Find(context.TODO(), backends, filter, noCache, outFormat)
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

	//util.SetNamingStrategy(util.LowerCaseWithUnderscores)

	Root.PersistentFlags().CountVarP(&verbose, "verbose", "v", "Print lots more stuff (repeat for more)")
	Root.PersistentFlags().BoolVarP(&quiet, "quiet", "q", quiet, "Print as little stuff as possible")
	Root.PersistentFlags().BoolVarP(&noCache, "no-cache", "", noCache, "no-cache will skip lookup in cache")
	Root.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.honey.yaml)")
	Root.PersistentFlags().StringVarP(&outFormat, "output", "o", outFormat, "")
	Root.PersistentFlags().StringVarP(&filter, "filter", "f", filter, "")
	Root.PersistentFlags().StringVarP(&backendsString, "backends", "b", backendsString, "")

	Root.AddCommand(newCompletionCmd(os.Stdout))
	Root.AddCommand(helpBackends)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	setVerboseLogFlags()

	//ctx := context.Background()
	//ci := place.GetConfig(ctx)

	/* if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".honey" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".honey")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} */
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
}

func setupRootCommand() {
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
}

func setVerboseLogFlags() {
	if verbose >= 2 {
		log.SetLevel(log.DebugLevel)
	} else if verbose >= 1 {
		log.SetLevel(log.InfoLevel)
	}
}
