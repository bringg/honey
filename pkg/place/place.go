package place

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
)

var (
	// Registry Backend registry
	Registry []*RegInfo
)

type (
	// A configmap.Getter to read from the environment HONEY_option_name
	optionEnvVars struct {
		backendInfo *RegInfo
	}

	// A configmap.Getter to read from the environment HONEY_CONFIG_backend_option_name
	configEnvVars string
)

// Register backend
func Register(info *RegInfo) {
	//info.Options.setValues()
	if info.Prefix == "" {
		info.Prefix = info.Name
	}
	Registry = append(Registry, info)
}

// Find find backend
func Find(name string) (*RegInfo, error) {
	for _, item := range Registry {
		if item.Name == name || item.Prefix == name {
			return item, nil
		}
	}
	return nil, errors.Errorf("didn't find backend called %q", name)
}

// Get a config item from the environment variables if possible
func (configName configEnvVars) Get(key string) (value string, ok bool) {
	return os.LookupEnv(ConfigToEnv(string(configName), key))
}

// Get a config item from the option environment variables if possible
func (oev optionEnvVars) Get(key string) (value string, ok bool) {
	opt := oev.backendInfo.Options.Get(key)
	if opt == nil {
		return "", false
	}
	// For options with NoPrefix set, check without prefix too
	if opt.NoPrefix {
		value, ok = os.LookupEnv(OptionToEnv(key))
		if ok {
			return value, ok
		}
	}
	return os.LookupEnv(OptionToEnv(oev.backendInfo.Prefix + "-" + key))
}

func ConfigMap(backendInfo *RegInfo, configName string) (config *configmap.Map) {
	// Create the config
	config = configmap.New()

	// Read the config, more specific to least specific

	// flag values
	if backendInfo != nil {
		config.AddGetter(&regInfoValues{backendInfo, false})
	}

	// remote specific environment vars
	config.AddGetter(configEnvVars(configName))

	// backend specific environment vars
	if backendInfo != nil {
		config.AddGetter(optionEnvVars{backendInfo: backendInfo})
	}

	// config file
	config.AddGetter(getConfigFile(configName))

	// default values
	if backendInfo != nil {
		config.AddGetter(&regInfoValues{backendInfo, true})
	}

	// Set Config
	config.AddSetter(setConfigFile(configName))
	return config
}

// GetValue gets the current current value which is the default if not set
func (o *Option) GetValue() interface{} {
	val := o.Value
	if val == nil {
		val = o.Default
		if val == nil {
			val = ""
		}
	}
	return val
}

// String turns Option into a string
func (o *Option) String() string {
	return fmt.Sprint(o.GetValue())
}

// Set an Option from a string
func (o *Option) Set(s string) (err error) {
	newValue, err := configstruct.StringToInterface(o.GetValue(), s)
	if err != nil {
		return err
	}
	o.Value = newValue
	return nil
}

// Type of the value
func (o *Option) Type() string {
	return reflect.TypeOf(o.GetValue()).Name()
}

// FlagName for the option
func (o *Option) FlagName(prefix string) string {
	name := strings.Replace(o.Name, "_", "-", -1) // convert snake_case to kebab-case
	if !o.NoPrefix {
		name = prefix + "-" + name
	}
	return name
}

// EnvVarName for the option
func (o *Option) EnvVarName(prefix string) string {
	return OptionToEnv(prefix + "-" + o.Name)
}

// Set the default values for the options
func (os Options) setValues() {
	for i := range os {
		o := &os[i]
		if o.Default == nil {
			o.Default = ""
		}
	}
}

// Get the Option corresponding to name or return nil if not found
func (os Options) Get(name string) *Option {
	for i := range os {
		opt := &os[i]
		if opt.Name == name {
			return opt
		}
	}
	return nil
}

// Len is part of sort.Interface.
func (os OptionExamples) Len() int { return len(os) }

// Swap is part of sort.Interface.
func (os OptionExamples) Swap(i, j int) { os[i], os[j] = os[j], os[i] }

// Less is part of sort.Interface.
func (os OptionExamples) Less(i, j int) bool { return os[i].Help < os[j].Help }

// Sort sorts an OptionExamples
func (os OptionExamples) Sort() { sort.Sort(os) }

// override the values in configMap with the either the flag values or
// the default values
func (r *regInfoValues) Get(key string) (value string, ok bool) {
	opt := r.backendInfo.Options.Get(key)
	if opt != nil && (r.useDefault || opt.Value != nil) {
		return opt.String(), true
	}
	return "", false
}

// Get a config item from the config file
func (section getConfigFile) Get(key string) (value string, ok bool) {
	value, ok = ConfigFileGet(string(section), key)
	// Ignore empty lines in the config file
	if value == "" {
		ok = false
	}
	return value, ok
}

// Set a config item into the config file
func (section setConfigFile) Set(key, value string) {
	//Debugf(nil, "Saving config %q = %q in section %q of the config file", key, value, section)
	err := ConfigFileSet(string(section), key, value)
	if err != nil {
		//Errorf(nil, "Failed saving config %q = %q in section %q of the config file: %v", key, value, section, err)
		log.Fatalf("Failed saving config %q = %q in section %q of the config file: %v", key, value, section, err)
	}
}

func InstanceFieldNames() []string {
	v := reflect.ValueOf(Instance{})
	t := v.Type()

	fields := make([]string, 0)
	for i := 0; i < v.NumField(); i++ {
		fields = append(fields, t.Field(i).Name)
	}

	return fields
}

func (p Printable) Interface() interface{} {
	return p
}

func (p Printable) Headers() []string {
	return InstanceFieldNames()
}

func (p Printable) Rows() [][]string {
	rows := make([][]string, len(p))
	for _, i := range p {
		rows = append(rows, []string{
			i.ID,
			i.BackendName,
			i.Name,
			i.Type,
			i.Status,
			i.PrivateIP,
			i.PublicIP,
		})
	}

	return rows
}
