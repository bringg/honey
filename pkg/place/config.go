package place

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/rclone/rclone/fs"
)

// Global
var (
	// globalConfig for rclone
	globalConfig = NewConfig()

	// Read a value from the config file
	//
	// This is a function pointer to decouple the config
	// implementation from the fs
	ConfigFileGet = func(section, key string) (string, bool) { return "", false }

	// Set a value into the config file and persist it
	//
	// This is a function pointer to decouple the config
	// implementation from the fs
	ConfigFileSet = func(section, key, value string) (err error) {
		return errors.New("no config file set handler")
	}
)

type (
	// ConfigInfo is honey config options
	ConfigInfo struct {
		NoCache        bool
		NoColor        bool
		OutFormat      string
		BackendsString string
		CacheTTL       uint32
	}
)

func NewConfig() *ConfigInfo {
	c := new(ConfigInfo)

	c.OutFormat = "table"
	c.CacheTTL = 600 // Set ttl = 600 , after 600 seconds, cache key will be expired.

	return c
}

type configContextKeyType struct{}

// Context key for config
var configContextKey = configContextKeyType{}

// GetConfig returns the global or context sensitive context
func GetConfig(ctx context.Context) *ConfigInfo {
	if ctx == nil {
		return globalConfig
	}
	c := ctx.Value(configContextKey)
	if c == nil {
		return globalConfig
	}
	return c.(*ConfigInfo)
}

func (c *ConfigInfo) Backends() ([]string, error) {
	backends := fs.CommaSepList{}
	if err := backends.Set(c.BackendsString); err != nil {
		return nil, err
	}

	return backends, nil
}

// AddConfig returns a mutable config structure based on a shallow
// copy of that found in ctx and returns a new context with that added
// to it.
func AddConfig(ctx context.Context) (context.Context, *ConfigInfo) {
	c := GetConfig(ctx)
	cCopy := new(ConfigInfo)
	*cCopy = *c
	newCtx := context.WithValue(ctx, configContextKey, cCopy)
	return newCtx, cCopy
}

// ConfigToEnv converts a config section and name, e.g. ("myremote",
// "ignore-size") into an environment name
// "HONEY_CONFIG_MYREMOTE_IGNORE_SIZE"
func ConfigToEnv(section, name string) string {
	return "HONEY_CONFIG_" + strings.ToUpper(strings.ReplaceAll(section+"_"+name, "-", "_"))
}

// OptionToEnv converts an option name, e.g. "ignore-size" into an
// environment name "HONEY_IGNORE_SIZE"
func OptionToEnv(name string) string {
	return "HONEY_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
}
