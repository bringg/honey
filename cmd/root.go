package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/shareed2k/honey/pkg/place"
	"github.com/shareed2k/honey/pkg/place/operations"
)

var (
	Version = "development"
	Commit  = "development"
	Date    = time.Now().String()

	cfgFile      string
	filter       string
	force        bool
	outFormat    = "json"
	backends     []string
	backendFlags map[string]struct{}
	// to filter the flags with
	flagsRe *regexp.Regexp

	rootCmd = &cobra.Command{
		Use:     "honey",
		Short:   "DevOps tool to help find an instance in sea of clouds",
		Version: Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				re, err := regexp.Compile(args[0])
				if err != nil {
					log.Fatalf("Failed to compile flags regexp: %v", err)
				}

				flagsRe = re
			}

			if len(backends) == 0 {
				return errors.New("oops you need to select backend")
			}

			return operations.Find(context.TODO(), backends, filter, force, outFormat)
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	setupRootCommand()
	addBackendFlags()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolVarP(&force, "force", "", force, "force will skip lookup in cache")
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.honey.yaml)")
	rootCmd.PersistentFlags().StringVarP(&outFormat, "output", "o", outFormat, "")
	rootCmd.PersistentFlags().StringVarP(&filter, "filter", "f", filter, "")
	rootCmd.PersistentFlags().StringArrayVarP(&backends, "backends", "b", backends, "")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
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
			// Skip if done already (e.g. with Provider options)
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
				fs.Errorf(nil, "Not adding duplicate flag --%s", name)
			}
		}
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
