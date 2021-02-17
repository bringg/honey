package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/rclone/rclone/fs/config/obscure"
	"github.com/rclone/rclone/fs/rc"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/bringg/honey/pkg/place"
)

var (
	log = logrus.WithField("where", "config")

	// configFile is the global config data structure. Don't read it directly, use getConfigData()
	configFile *viper.Viper

	// ConfigPath points to the config file
	ConfigPath = makeConfigPath()
)

const (
	configFileName       = "honey.json"
	hiddenConfigFileName = "." + configFileName
)

func init() {
	// Set the function pointers up in place
	place.ConfigFileGet = FileGetFlag
	place.ConfigFileSet = SetValueAndSave
}

func CreateBackend(ctx context.Context, name string, provider string, keyValues rc.Params, doObscure, noObscure bool) error {
	getConfigData().Set(fmt.Sprintf("%s.type", name), provider)

	return UpdateBackend(ctx, name, keyValues, doObscure, noObscure)
}

// ShowBackend shows the contents of the backend
func ShowBackend(name string) {
	fmt.Printf("--------------------\n")
	fmt.Printf("[%s]\n", name)

	rf := MustFindByName(name)
	for key, _ := range getConfigData().GetStringMap(name) {
		isPassword := false
		for _, option := range rf.Options {
			if option.Name == key && option.IsPassword {
				isPassword = true
				break
			}
		}

		value := FileGet(name, key)
		if isPassword && value != "" {
			fmt.Printf("%s = *** ENCRYPTED ***\n", key)
		} else {
			fmt.Printf("%s = %s\n", key, value)
		}
	}

	fmt.Printf("--------------------\n")
}

// FileGetFlag gets the config key under section returning the
// the value and true if found and or ("", false) otherwise
func FileGetFlag(section, key string) (string, bool) {
	if val := getConfigData().GetString(fmt.Sprintf("%s.%s", section, key)); val != "" {
		return val, true
	}

	return "", false
}

// SetValueAndSave sets the key to the value and saves just that
// value in the config file.  It loads the old config file in from
// disk first and overwrites the given value only.
func SetValueAndSave(name, key, value string) (err error) {
	// Set the value in config in case we fail to reload it
	getConfigData().Set(fmt.Sprintf("%s.%s", name, key), value)

	return configFile.WriteConfig()
}

// UpdateBackend adds the keyValues passed in to the remote of name.
// keyValues should be key, value pairs.
func UpdateBackend(ctx context.Context, name string, keyValues rc.Params, doObscure, noObscure bool) error {
	if doObscure && noObscure {
		return errors.New("can't use --obscure and --no-obscure together")
	}

	// Work out which options need to be obscured
	needsObscure := map[string]struct{}{}
	if !noObscure {
		if bType := FileGet(name, "type"); bType != "" {
			if ri, err := place.Find(bType); err != nil {
				log.Debugf("Couldn't find backend for type %q", bType)
			} else {
				for _, opt := range ri.Options {
					if opt.IsPassword {
						needsObscure[opt.Name] = struct{}{}
					}
				}
			}
		} else {
			log.Debugf("UpdateBackend: Couldn't find backend type")
		}
	}

	// Set the config
	for k, v := range keyValues {
		vStr := fmt.Sprint(v)
		// Obscure parameter if necessary
		if _, ok := needsObscure[k]; ok {
			_, err := obscure.Reveal(vStr)
			if err != nil || doObscure {
				// If error => not already obscured, so obscure it
				// or we are forced to obscure
				vStr, err = obscure.Obscure(vStr)
				if err != nil {
					return errors.Wrap(err, "UpdateBackend: obscure failed")
				}
			}
		}

		getConfigData().Set(fmt.Sprintf("%s.%s", name, k), vStr)
	}

	BackendConfig(ctx, name)

	return getConfigData().WriteConfig()
}

// LoadConfig loads the config file
func LoadConfig() {
	cfgFile := ConfigPath
	// Set HONEY_CONFIG_DIR for backend config
	_ = os.Setenv("HONEY_CONFIG_DIR", filepath.Dir(cfgFile))

	configFile = viper.New()
	// Load configuration file.
	configFile.SetConfigFile(cfgFile)

	configFile.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := configFile.ReadInConfig(); err == nil {
		log.Debugln("Using config file:", configFile.ConfigFileUsed())
	}
}

// FileGet gets the config key under section returning the
// default or empty string if not set.
//
// It looks up defaults in the environment if they are present
func FileGet(section, key string, defaultVal ...string) string {
	envKey := place.ConfigToEnv(section, key)
	newValue, found := os.LookupEnv(envKey)
	if found {
		defaultVal = []string{newValue}
	}

	if val := getConfigData().GetString(fmt.Sprintf("%s.%s", section, key)); val != "" {
		return val
	}

	if len(defaultVal) > 0 {
		return defaultVal[0]
	}

	return ""
}

// MustFindByName finds the RegInfo for the remote name passed in or
// exits with a fatal error.
func MustFindByName(name string) *place.RegInfo {
	bType := FileGet(name, "type")
	if bType == "" {
		log.Fatalf("Couldn't find type of backend for %q", name)
	}

	return place.MustFind(bType)
}

// BackendConfig runs the config helper for the backend if needed
func BackendConfig(ctx context.Context, name string) {
	log.Printf("Backend config %s\n", name)

	f := MustFindByName(name)
	if f.Config != nil {
		m := place.ConfigMap(f, name)
		f.Config(ctx, name, m)
	}
}

func getConfigData() *viper.Viper {
	if configFile == nil {
		LoadConfig()
	}
	return configFile
}

// Return the path to the configuration file
func makeConfigPath() string {
	// Use honey.json from honey executable directory if already existing
	exe, err := os.Executable()
	if err == nil {
		exedir := filepath.Dir(exe)
		cfgpath := filepath.Join(exedir, configFileName)
		_, err := os.Stat(cfgpath)
		if err == nil {
			return cfgpath
		}
	}

	// Find user's home directory
	homeDir, err := homedir.Dir()

	// Find user's configuration directory.
	// Prefer XDG config path, with fallback to $HOME/.config.
	// See XDG Base Directory specification
	// https://specifications.freedesktop.org/basedir-spec/latest/),
	xdgdir := os.Getenv("XDG_CONFIG_HOME")
	var cfgdir string
	if xdgdir != "" {
		// User's configuration directory for honey is $XDG_CONFIG_HOME/honey
		cfgdir = filepath.Join(xdgdir, "honey")
	} else if homeDir != "" {
		// User's configuration directory for honey is $HOME/.config/honey
		cfgdir = filepath.Join(homeDir, ".config", "honey")
	}

	// Use honey.json from user's configuration directory if already existing
	var cfgpath string
	if cfgdir != "" {
		cfgpath = filepath.Join(cfgdir, configFileName)
		_, err := os.Stat(cfgpath)
		if err == nil {
			return cfgpath
		}
	}

	// Use .honey.json from user's home directory if already existing
	var homeconf string
	if homeDir != "" {
		homeconf = filepath.Join(homeDir, hiddenConfigFileName)
		_, err := os.Stat(homeconf)
		if err == nil {
			return homeconf
		}
	}

	// Check to see if user supplied a --config variable or environment
	// variable.  We can't use pflag for this because it isn't initialised
	// yet so we search the command line manually.
	_, configSupplied := os.LookupEnv("HONEY_CONFIG")
	if !configSupplied {
		for _, item := range os.Args {
			if item == "--config" || strings.HasPrefix(item, "--config=") {
				configSupplied = true
				break
			}
		}
	}

	// If user's configuration directory was found, then try to create it
	// and assume honey.json can be written there. If user supplied config
	// then skip creating the directory since it will not be used.
	if cfgpath != "" {
		// cfgpath != "" implies cfgdir != ""
		if configSupplied {
			return cfgpath
		}

		err := os.MkdirAll(cfgdir, os.ModePerm)
		if err == nil {
			return cfgpath
		}
	}

	// Assume .honey.json can be written to user's home directory.
	if homeconf != "" {
		return homeconf
	}

	// Default to ./.honey.json (current working directory) if everything else fails.
	if !configSupplied {
		log.Errorf("Couldn't find home directory or read HOME or XDG_CONFIG_HOME environment variables.")
		log.Errorf("Defaulting to storing config in current directory.")
		log.Errorf("Use --config flag to workaround.")
		log.Errorf("Error was: %v", err)
	}

	return hiddenConfigFileName
}
