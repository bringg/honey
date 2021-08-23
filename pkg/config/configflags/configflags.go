package configflags

import (
	"github.com/rclone/rclone/fs/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/bringg/honey/pkg/config/flags"
	"github.com/bringg/honey/pkg/place"
)

var (
	// these will get interpreted into place.Config via SetFlags() below
	verbose    int
	quiet      bool
	configPath string
)

// AddFlags adds the non filing system specific flags to the command
func AddFlags(ci *place.ConfigInfo, flagSet *pflag.FlagSet) {
	flags.CountVarP(flagSet, &verbose, "verbose", "v", "Print lots more stuff (repeat for more)")
	flags.BoolVarP(flagSet, &ci.NoColor, "no-color", "", ci.NoColor, "disable colorize the json for outputing to the screen")
	flags.BoolVarP(flagSet, &quiet, "quiet", "q", quiet, "Print as little stuff as possible")
	flags.BoolVarP(flagSet, &ci.NoCache, "no-cache", "", ci.NoCache, "no-cache will skip lookup in cache")
	flags.DurationVarP(flagSet, &ci.CacheTTL, "cache-ttl", "", ci.CacheTTL, "cache-ttl cache duration in seconds")
	flags.StringVarP(flagSet, &configPath, "config", "c", config.GetConfigPath(), "config file")
	flags.StringVarP(flagSet, &ci.OutFormat, "output", "o", ci.OutFormat, "")
	flags.StringVarP(flagSet, &ci.BackendsString, "backends", "b", ci.BackendsString, "")
}

// SetFlags converts any flags into config which weren't straight forward
func SetFlags() {
	if verbose >= 2 {
		logrus.SetLevel(logrus.DebugLevel)
	} else if verbose >= 1 {
		logrus.SetLevel(logrus.InfoLevel)
	}

	if quiet {
		if verbose > 0 {
			logrus.Fatalf("Can't set -v and -q")
		}

		logrus.SetLevel(logrus.ErrorLevel)
	}

	// Set path to configuration file
	if err := config.SetConfigPath(configPath); err != nil {
		logrus.Fatalf("--config: Failed to set %q as config path: %v", configPath, err)
	}
}
