package config

import (
	"bufio"
	"context"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configstruct"
	"github.com/rclone/rclone/fs/config/obscure"
	"github.com/rclone/rclone/fs/driveletter"
	"github.com/rclone/rclone/fs/fspath"
	"github.com/rclone/rclone/fs/rc"
	"github.com/rclone/rclone/lib/file"
	"github.com/rclone/rclone/lib/random"
	"github.com/rclone/rclone/lib/terminal"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/text/unicode/norm"

	"github.com/bringg/honey/pkg/place"
)

var (
	log = logrus.WithField("where", "config")

	// configFile is the global config data structure. Don't read it directly, use getConfigData()
	configFile *viper.Viper

	// ConfigPath points to the config file
	ConfigPath = makeConfigPath()

	// output of prompt for password
	PasswordPromptOutput = os.Stderr

	// Password can be used to configure the random password generator
	Password = random.Password
)

const (
	configFileName       = "honey.json"
	noConfigFile         = "notfound"
	hiddenConfigFileName = "." + configFileName
)

func init() {
	// Set the function pointers up in place
	place.ConfigFileGet = FileGetFlag
	place.ConfigFileSet = SetValueAndSave
}

// GetConfigPath get config path
func GetConfigPath() string {
	return ConfigPath
}

// SetConfigPath sets new config file path
//
// Checks for empty string, os null device, or special path, all of which indicates in-memory config.
func SetConfigPath(path string) (err error) {
	var cfgPath string
	if path == "" || path == os.DevNull {
		cfgPath = ""
	} else if filepath.Base(path) == noConfigFile {
		cfgPath = ""
	} else if err = file.IsReserved(path); err != nil {
		return err
	} else if cfgPath, err = filepath.Abs(path); err != nil {
		return err
	}

	ConfigPath = cfgPath

	return nil
}

// CreateBackend create a new backend
func CreateBackend(ctx context.Context, name string, provider string, keyValues rc.Params, doObscure, noObscure bool) error {
	getConfigData().Set(fmt.Sprintf("%s.type", name), provider)

	return UpdateBackend(ctx, name, keyValues, doObscure, noObscure)
}

// ShowBackend shows the contents of the backend
func ShowBackend(name string) {
	fmt.Printf("--------------------\n")
	fmt.Printf("[%s]\n", name)

	rf := MustFindByName(name)
	for key := range getConfigData().GetStringMap(name) {
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

// UpdateBackend adds the keyValues passed in to the backend of name.
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

	// configFile.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := configFile.ReadInConfig(); err != nil {
		if errors.Is(err, iofs.ErrNotExist) || errors.As(err, &viper.ConfigFileNotFoundError{}) {
			if ConfigPath == "" {
				log.Debug("Config is memory-only - using defaults")
			} else {
				log.Debugf("Config file %q not found - using defaults", ConfigPath)
			}

			return
		}

		log.WithError(err).Fatalf("Failed to load config file %q", ConfigPath)
	}

	log.Debugln("Using config file:", configFile.ConfigFileUsed())
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

// MustFindByName finds the RegInfo for the backend name passed in or
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

// editOptions edits the options.  If new is true then it just allows
// entry and doesn't show any old values.
func editOptions(ri *place.RegInfo, name string, isNew bool) {
	hasAdvanced := false
	for _, advanced := range []bool{false, true} {
		if advanced {
			if !hasAdvanced {
				break
			}
			fmt.Printf("Edit advanced config? (y/n)\n")
			if !Confirm(false) {
				break
			}
		}

		for _, option := range ri.Options {
			isVisible := option.Hide&place.OptionHideConfigurator == 0
			hasAdvanced = hasAdvanced || (option.Advanced && isVisible)
			if option.Advanced != advanced {
				continue
			}

			subProvider := getConfigData().GetString(fmt.Sprintf("%s.%s", name, fs.ConfigProvider))
			if matchProvider(option.Provider, subProvider) && isVisible {
				if !isNew {
					fmt.Printf("Value %q = %q\n", option.Name, FileGet(name, option.Name))
					fmt.Printf("Edit? (y/n)>\n")
					if !Confirm(false) {
						continue
					}
				}

				FileSet(name, option.Name, ChooseOption(option, name))
			}
		}
	}
}

// GetPassword asks the user for a password with the prompt given.
func GetPassword(prompt string) string {
	_, _ = fmt.Fprintln(PasswordPromptOutput, prompt)
	for {
		_, _ = fmt.Fprint(PasswordPromptOutput, "password:")
		password := ReadPassword()
		password, err := checkPassword(password)
		if err == nil {
			return password
		}
		_, _ = fmt.Fprintf(os.Stderr, "Bad password: %v\n", err)
	}
}

// checkPassword normalises and validates the password
func checkPassword(password string) (string, error) {
	if !utf8.ValidString(password) {
		return "", errors.New("password contains invalid utf8 characters")
	}
	// Check for leading/trailing whitespace
	trimmedPassword := strings.TrimSpace(password)
	// Warn user if password has leading+trailing whitespace
	if len(password) != len(trimmedPassword) {
		_, _ = fmt.Fprintln(os.Stderr, "Your password contains leading/trailing whitespace - in previous versions of rclone this was stripped")
	}
	// Normalize to reduce weird variations.
	password = norm.NFKC.String(password)
	if len(password) == 0 || len(trimmedPassword) == 0 {
		return "", errors.New("no characters in password")
	}
	return password, nil
}

// ChangePassword will query the user twice for the named password. If
// the same password is entered it is returned.
func ChangePassword(name string) string {
	for {
		a := GetPassword(fmt.Sprintf("Enter %s password:", name))
		b := GetPassword(fmt.Sprintf("Confirm %s password:", name))
		if a == b {
			return a
		}
		fmt.Println("Passwords do not match!")
	}
}

// ChooseOption asks the user to choose an option
func ChooseOption(o place.Option, name string) string {
	var subProvider = getConfigData().GetString(fmt.Sprintf("%s.%s", name, fs.ConfigProvider))
	fmt.Println(o.Help)

	if o.IsPassword {
		actions := []string{"yYes type in my own password", "gGenerate random password"}
		defaultAction := -1
		if !o.Required {
			defaultAction = len(actions)
			actions = append(actions, "nNo leave this optional password blank")
		}

		var password string
		var err error
		switch i := CommandDefault(actions, defaultAction); i {
		case 'y':
			password = ChangePassword("the")
		case 'g':
			for {
				fmt.Printf("Password strength in bits.\n64 is just about memorable\n128 is secure\n1024 is the maximum\n")
				bits := ChooseNumber("Bits", 64, 1024)
				password, err = Password(bits)
				if err != nil {
					log.Fatalf("Failed to make password: %v", err)
				}
				fmt.Printf("Your password is: %s\n", password)
				fmt.Printf("Use this password? Please note that an obscured version of this \npassword (and not the " +
					"password itself) will be stored under your \nconfiguration file, so keep this generated password " +
					"in a safe place.\n")
				if Confirm(true) {
					break
				}
			}
		case 'n':
			return ""
		default:
			fs.Errorf(nil, "Bad choice %c", i)
		}

		return obscure.MustObscure(password)
	}

	what := fmt.Sprintf("%T value", o.Default)
	switch o.Default.(type) {
	case bool:
		what = "boolean value (true or false)"
	case fs.SizeSuffix:
		what = "size with suffix k,M,G,T"
	case fs.Duration:
		what = "duration s,m,h,d,w,M,y"
	case int, int8, int16, int32, int64:
		what = "signed integer"
	case uint, byte, uint16, uint32, uint64:
		what = "unsigned integer"
	}

	var in string
	for {
		fmt.Printf("Enter a %s. Press Enter for the default (%q).\n", what, fmt.Sprint(o.Default))
		if len(o.Examples) > 0 {
			var values []string
			var help []string
			for _, example := range o.Examples {
				if matchProvider(example.Provider, subProvider) {
					values = append(values, example.Value)
					help = append(help, example.Help)
				}
			}
			in = Choose(o.Name, values, help, true)
		} else {
			fmt.Printf("%s> ", o.Name)
			in = ReadLine()
		}

		if in == "" {
			if o.Required && fmt.Sprint(o.Default) == "" {
				fmt.Printf("This value is required and it has no default.\n")
				continue
			}
			break
		}

		newIn, err := configstruct.StringToInterface(o.Default, in)
		if err != nil {
			fmt.Printf("Failed to parse %q: %v\n", in, err)
			continue
		}

		in = fmt.Sprint(newIn) // canonicalise
		break
	}

	return in
}

// ChooseNumber asks the user to enter a number between min and max
// inclusive prompting them with what.
func ChooseNumber(what string, min, max int) int {
	for {
		fmt.Printf("%s> ", what)
		result := ReadLine()

		i, err := strconv.Atoi(result)
		if err != nil {
			fmt.Printf("Bad number: %v\n", err)
			continue
		}

		if i < min || i > max {
			fmt.Printf("Out of range - %d to %d inclusive\n", min, max)
			continue
		}

		return i
	}
}

// Choose one of the defaults or type a new string if newOk is set
func Choose(what string, defaults, help []string, newOk bool) string {
	valueDescription := "an existing"
	if newOk {
		valueDescription = "your own"
	}

	fmt.Printf("Choose a number from below, or type in %s value\n", valueDescription)
	attributes := []string{terminal.HiRedFg, terminal.HiGreenFg}
	for i, text := range defaults {
		var lines []string
		if help != nil {
			parts := strings.Split(help[i], "\n")
			lines = append(lines, parts...)
		}

		lines = append(lines, fmt.Sprintf("%q", text))
		pos := i + 1
		terminal.WriteString(attributes[i%len(attributes)])

		if len(lines) == 1 {
			fmt.Printf("%2d > %s\n", pos, text)
		} else {
			mid := (len(lines) - 1) / 2
			for i, line := range lines {
				var sep rune
				switch i {
				case 0:
					sep = '/'
				case len(lines) - 1:
					sep = '\\'
				default:
					sep = '|'
				}

				number := "  "
				if i == mid {
					number = fmt.Sprintf("%2d", pos)
				}

				fmt.Printf("%s %c %s\n", number, sep, line)
			}
		}

		terminal.WriteString(terminal.Reset)
	}

	for {
		fmt.Printf("%s> ", what)
		result := ReadLine()
		i, err := strconv.Atoi(result)
		if err != nil {
			if newOk {
				return result
			}

			for _, v := range defaults {
				if result == v {
					return result
				}
			}

			continue
		}

		if i >= 1 && i <= len(defaults) {
			return defaults[i-1]
		}
	}
}

// ReadLine reads some input
var ReadLine = func() string {
	buf := bufio.NewReader(os.Stdin)
	line, err := buf.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read line: %v", err)
	}

	return strings.TrimSpace(line)
}

// CommandDefault - choose one.  If return is pressed then it will
// chose the defaultIndex if it is >= 0
func CommandDefault(commands []string, defaultIndex int) byte {
	opts := []string{}
	for i, text := range commands {
		def := ""
		if i == defaultIndex {
			def = " (default)"
		}
		fmt.Printf("%c) %s%s\n", text[0], text[1:], def)
		opts = append(opts, text[:1])
	}

	optString := strings.Join(opts, "")
	optHelp := strings.Join(opts, "/")

	for {
		fmt.Printf("%s> ", optHelp)
		result := strings.ToLower(ReadLine())
		if len(result) == 0 && defaultIndex >= 0 {
			return optString[defaultIndex]
		}
		if len(result) != 1 {
			continue
		}
		i := strings.Index(optString, string(result[0]))
		if i >= 0 {
			return result[0]
		}
	}
}

// FileSet sets the key in section to value.  It doesn't save
// the config file.
func FileSet(section, key, value string) {
	if value != "" {
		getConfigData().Set(fmt.Sprintf("%s.%s", section, key), value)
	}
}

func getSectionList() (backends []string) {
	keys := new(map[string]interface{})
	if err := getConfigData().Unmarshal(keys); err != nil {
		return
	}

	for key := range *keys {
		backends = append(backends, key)
	}

	return
}

func BackendListUnmarshal(out interface{}) error {
	return getConfigData().Unmarshal(out)
}

// ShowBackends shows an overview of the config file
func ShowBackends() {
	backends := getSectionList()

	if len(backends) == 0 {
		return
	}

	sort.Strings(backends)
	fmt.Printf("%-20s %s\n", "Name", "Type")
	fmt.Printf("%-20s %s\n", "====", "====")

	for _, backend := range backends {
		fmt.Printf("%-20s %s\n", backend, FileGet(backend, "type"))
	}
}

// Command - choose one
func Command(commands []string) byte {
	return CommandDefault(commands, -1)
}

// EditConfig edits the config file interactively
func EditConfig(ctx context.Context) {
	for {
		haveBackends := len(getSectionList()) != 0
		what := []string{"eEdit existing backend", "nNew backend", "qQuit config"}

		if haveBackends {
			fmt.Printf("Current backends:\n\n")
			ShowBackends()
			fmt.Printf("\n")
		} else {
			fmt.Printf("No backends found - make a new one\n")
			// take 2nd item and last 2 items of menu list
			what = append(what[1:2], what[len(what)-2:]...)
		}

		switch i := Command(what); i {
		case 'e':
			name := ChooseBackend()
			b := MustFindByName(name)
			EditBackend(ctx, b, name)
		case 'n':
			NewBackend(ctx, NewBackendName())
		case 'q':
			return

		}
	}
}

// NewBackendName asks the user for a name for a new backend
func NewBackendName() (name string) {
	for {
		fmt.Printf("name> ")
		name = ReadLine()

		if getConfigData().IsSet(name) {
			fmt.Printf("Backend %q already exists.\n", name)
			continue
		}

		err := fspath.CheckConfigName(name)
		switch {
		case name == "":
			fmt.Printf("Can't use empty name.\n")
		case driveletter.IsDriveLetter(name):
			fmt.Printf("Can't use %q as it can be confused with a drive letter.\n", name)
		case err != nil:
			fmt.Printf("Can't use %q as %v.\n", name, err)
		default:
			return name
		}
	}
}

// NewBackend make a new backend from its name
func NewBackend(ctx context.Context, name string) {
	var (
		newType string
		ri      *place.RegInfo
		err     error
	)

	// Set the type first
	for {
		newType = ChooseOption(bOption(), name)
		ri, err = place.Find(newType)
		if err != nil {
			fmt.Printf("Bad backend %q: %v\n", newType, err)
			continue
		}

		break
	}

	getConfigData().Set(fmt.Sprintf("%s.type", name), newType)

	editOptions(ri, name, true)
	BackendConfig(ctx, name)

	if OkBackend(name) {
		getConfigData().WriteConfig()
		return
	}

	EditBackend(ctx, ri, name)
}

// bOption returns an Option describing the possible backends
func bOption() place.Option {
	o := place.Option{
		Name:    "Backend",
		Help:    "Type of backend to configure.",
		Default: "",
	}

	for _, item := range place.Registry {
		example := place.OptionExample{
			Value: item.Name,
			Help:  item.Description,
		}

		o.Examples = append(o.Examples, example)
	}

	o.Examples.Sort()

	return o
}

// OkBackend prints the contents of the backend and ask if it is OK
func OkBackend(name string) bool {
	ShowBackend(name)

	switch i := CommandDefault([]string{"yYes this is OK", "eEdit this backend"}, 0); i {
	case 'y':
		return true
	case 'e':
		return false
	default:
		fs.Errorf(nil, "Bad choice %c", i)
	}

	return false
}

// EditBackend gets the user to edit a backend
func EditBackend(ctx context.Context, ri *place.RegInfo, name string) {
	ShowBackend(name)
	fmt.Printf("Edit backend\n")

	for {
		editOptions(ri, name, false)
		if OkBackend(name) {
			break
		}
	}

	getConfigData().WriteConfig()
	BackendConfig(ctx, name)
}

// ChooseBackend chooses a backend name
func ChooseBackend() string {
	backends := getSectionList()
	sort.Strings(backends)

	return Choose("backend", backends, nil, false)
}

// matchProvider returns true if provider matches the providerConfig string.
//
// The providerConfig string can either be a list of providers to
// match, or if it starts with "!" it will be a list of providers not
// to match.
//
// If either providerConfig or provider is blank then it will return true
func matchProvider(providerConfig, provider string) bool {
	if providerConfig == "" || provider == "" {
		return true
	}

	negate := false

	if strings.HasPrefix(providerConfig, "!") {
		providerConfig = providerConfig[1:]
		negate = true
	}
	providers := strings.Split(providerConfig, ",")
	matched := false

	for _, p := range providers {
		if p == provider {
			matched = true
			break
		}
	}

	if negate {
		return !matched
	}

	return matched
}

// Confirm asks the user for Yes or No and returns true or false
//
// If the user presses enter then the isDefault will be used
func Confirm(isDefault bool) bool {
	defaultIndex := 1
	if isDefault {
		defaultIndex = 0
	}

	return config.CommandDefault([]string{"yYes", "nNo"}, defaultIndex) == 'y'
}
