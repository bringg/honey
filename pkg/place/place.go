package place

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/Rican7/conjson"
	"github.com/Rican7/conjson/transform"
	jsoniter "github.com/json-iterator/go"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

var (
	// Registry Backend registry
	Registry []*RegInfo

	log = logrus.WithField("where", "place")
)

type (
	// A configmap.Getter to read from the environment HONEY_option_name
	optionEnvVars struct {
		backendInfo *RegInfo
	}

	// A configmap.Getter to read from the environment HONEY_CONFIG_backend_option_name
	configEnvVars string

	FlattenData struct {
		Len   int
		Bytes []byte
	}
)

// Register backend
func Register(info *RegInfo) {
	info.Options.setValues()

	if info.Prefix == "" {
		info.Prefix = info.Name
	}

	Registry = append(Registry, info)
}

func MustFind(name string) *RegInfo {
	b, err := Find(name)
	if err != nil {
		log.Fatalf("Failed to find backend: %v", err)
	}

	return b
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

func BackendNames() []string {
	names := make([]string, 0)
	for _, info := range Registry {
		names = append(names, info.Name)
	}

	return names
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
		if value, ok = os.LookupEnv(OptionToEnv(key)); ok {
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
	name := strings.ReplaceAll(o.Name, "_", "-") // convert snake_case to kebab-case
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
	log.Debugf("Saving config %q = %q in section %q of the config file", key, value, section)

	err := ConfigFileSet(string(section), key, value)
	if err != nil {
		log.Fatalf("Failed saving config %q = %q in section %q of the config file: %v", key, value, section, err)
	}
}

func instanceFieldNames() []string {
	v := reflect.ValueOf(Model{})
	t := v.Type()

	fields := make([]string, 0)
	for i := 0; i < v.NumField(); i++ {
		tag, hastag := t.Field(i).Tag.Lookup("json")
		if hastag {
			tagParts := strings.Split(tag, ",")
			if tagParts[0] == "-" {
				continue // hidden field
			}
		}

		fields = append(fields, tag)
	}

	return fields
}

func (p Printable) FlattenData() (*FlattenData, error) {
	data := make([]map[string]interface{}, 0)
	for _, i := range p {
		modelData, err := ToMap(i)
		if err != nil {
			return nil, err
		}

		data = append(data, modelData)
	}

	marshaler := conjson.NewMarshaler(data, transform.ConventionalKeys())

	d, err := jsoniter.Marshal(marshaler)
	if err != nil {
		return nil, err
	}

	return &FlattenData{
		Len:   len(data),
		Bytes: d,
	}, nil
}

func (p Printable) Headers() []string {
	return instanceFieldNames()
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

func (d *FlattenData) ToArrayMap() ([]map[string]interface{}, error) {
	data := make([]map[string]interface{}, 0)
	if err := jsoniter.Unmarshal(d.Bytes, &data); err != nil {
		return nil, err
	}

	return data, nil
}

func (d *FlattenData) Filter(keys []string) ([]map[string]interface{}, error) {
	log.Debugf("using filter keys: %v", keys)

	cleanedData := make([]map[string]interface{}, d.Len)
	for i := 0; i < d.Len; i++ {
		if cleanedData[i] == nil {
			cleanedData[i] = map[string]interface{}{}
		}

		for _, key := range keys {
			cleanedData[i][key] = gjson.
				GetBytes(d.Bytes, fmt.Sprintf("%d.%s", i, key)).
				Value()
		}
	}

	return cleanedData, nil
}

func ToMap(m interface{}) (map[string]interface{}, error) {
	data := make(map[string]interface{}, 0)
	if err := mapstructure.Decode(m, &data); err != nil {
		return nil, err
	}

	return data, nil
}
