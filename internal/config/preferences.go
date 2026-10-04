package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/medialo/gogws/internal/gws2"
	"github.com/spf13/cast"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type ConfigSource string

const (
	SourceDefault ConfigSource = "default"
	SourceFile    ConfigSource = "file"
	SourceEnv     ConfigSource = "env"
	SourceFlag    ConfigSource = "flag"
)

type ConfigValue[T any] struct {
	Value  T
	Source ConfigSource
}

const (
	EnvPrefix = "GOGWS"

	KeyParallel          = "parallel"
	KeyFormat            = "format"
	KeyTheme             = "theme"
	KeyNoColor           = "no-color"
	KeyStopOnError       = "stop-on-error"
	KeyProviderCacheTTL  = "provider-cache-ttl"
	KeyTrustedWorkspaces = "trusted-workspaces"

	DefaultFormat           = "text"
	DefaultProviderCacheTTL = "24h"
)

var OutputFormats = []string{"text", "json", "yaml"}

type Preferences struct {
	Parallel          ConfigValue[int]
	Format            ConfigValue[string]
	Theme             ConfigValue[string]
	NoColor           ConfigValue[bool]
	StopOnError       ConfigValue[bool]
	ProviderCacheTTL  ConfigValue[time.Duration]
	TrustedWorkspaces ConfigValue[[]string]
}

type PreferenceKey struct {
	Name        string
	Type        string
	Default     any
	Description string
	Validate    func(raw string) error
}

var PreferenceKeys = []PreferenceKey{
	{KeyParallel, "integer", gws2.DefaultParallel, "Number of parallel operations", validateParallelText},
	{KeyFormat, "text | json | yaml", DefaultFormat, "Output format", checkFormat},
	{KeyTheme, "path", "", "Theme file", nil},
	{KeyNoColor, "boolean", false, "Disable colored output", validateBoolText},
	{KeyStopOnError, "boolean", false, "Stop execution on first error", validateBoolText},
	{KeyProviderCacheTTL, "duration", DefaultProviderCacheTTL, "How long a discovered provider workspace is kept before --refresh-providers re-reads it", validateDurationText},
	{KeyTrustedWorkspaces, "list of paths", []string{}, "Deprecated, no longer grants trust. Local hooks are trusted per file (path + sha256) in ~/.gws/" + TrustedHooksFile, nil},
}

func LookupPreferenceKey(name string) (PreferenceKey, bool) {
	for _, k := range PreferenceKeys {
		if k.Name == name {
			return k, true
		}
	}
	return PreferenceKey{}, false
}

func IsPreferenceKey(name string) bool {
	_, ok := LookupPreferenceKey(name)
	return ok
}

func EnvVarName(key string) string {
	return EnvPrefix + "_" + strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
}

func checkParallel(n int) error {
	if n < 1 {
		return errors.New("must be an integer greater than or equal to 1")
	}
	return nil
}

func checkFormat(format string) error {
	if !slices.Contains(OutputFormats, format) {
		return errors.New("must be one of " + strings.Join(OutputFormats, ", "))
	}
	return nil
}

func validateParallelText(raw string) error {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return errors.New("must be an integer greater than or equal to 1")
	}
	return checkParallel(n)
}

func validateBoolText(raw string) error {
	if _, err := strconv.ParseBool(raw); err != nil {
		return errors.New("must be true or false")
	}
	return nil
}

func validateDurationText(raw string) error {
	if _, err := time.ParseDuration(raw); err != nil {
		return errors.New("must be a duration (e.g. 30m, 12h)")
	}
	return nil
}

func LoadPreferences(flags *pflag.FlagSet, configFile string) (*Preferences, error) {
	if configFile == "" {
		path, err := GetUserConfigPath()
		if err != nil {
			return nil, err
		}
		configFile = path
	}

	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")
	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	for _, k := range PreferenceKeys {
		v.SetDefault(k.Name, k.Default)
		if flags == nil {
			continue
		}
		if f := flags.Lookup(k.Name); f != nil {
			if err := v.BindPFlag(k.Name, f); err != nil {
				return nil, err
			}
		}
	}

	if err := v.ReadInConfig(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	r := resolver{v: v, flags: flags}
	prefs := &Preferences{
		Theme:             ConfigValue[string]{Value: v.GetString(KeyTheme), Source: r.sourceOf(KeyTheme)},
		TrustedWorkspaces: ConfigValue[[]string]{Value: v.GetStringSlice(KeyTrustedWorkspaces), Source: r.sourceOf(KeyTrustedWorkspaces)},
	}

	parallel, err := cast.ToIntE(v.Get(KeyParallel))
	if err == nil {
		err = checkParallel(parallel)
	}
	if err != nil {
		return nil, r.invalid(KeyParallel, err)
	}
	prefs.Parallel = ConfigValue[int]{Value: parallel, Source: r.sourceOf(KeyParallel)}

	format := v.GetString(KeyFormat)
	if err := checkFormat(format); err != nil {
		return nil, r.invalid(KeyFormat, err)
	}
	prefs.Format = ConfigValue[string]{Value: format, Source: r.sourceOf(KeyFormat)}

	noColor, err := cast.ToBoolE(v.Get(KeyNoColor))
	if err != nil {
		return nil, r.invalid(KeyNoColor, errors.New("must be true or false"))
	}
	prefs.NoColor = ConfigValue[bool]{Value: noColor, Source: r.sourceOf(KeyNoColor)}

	stopOnError, err := cast.ToBoolE(v.Get(KeyStopOnError))
	if err != nil {
		return nil, r.invalid(KeyStopOnError, errors.New("must be true or false"))
	}
	prefs.StopOnError = ConfigValue[bool]{Value: stopOnError, Source: r.sourceOf(KeyStopOnError)}

	ttl, err := time.ParseDuration(v.GetString(KeyProviderCacheTTL))
	if err != nil {
		return nil, r.invalid(KeyProviderCacheTTL, errors.New("must be a duration (e.g. 30m, 12h)"))
	}
	prefs.ProviderCacheTTL = ConfigValue[time.Duration]{Value: ttl, Source: r.sourceOf(KeyProviderCacheTTL)}

	return prefs, nil
}

type resolver struct {
	v     *viper.Viper
	flags *pflag.FlagSet
}

func (r resolver) sourceOf(key string) ConfigSource {
	if r.flags != nil {
		if f := r.flags.Lookup(key); f != nil && f.Changed {
			return SourceFlag
		}
	}
	if val, ok := os.LookupEnv(EnvVarName(key)); ok && val != "" {
		return SourceEnv
	}
	if r.v.InConfig(key) {
		return SourceFile
	}
	return SourceDefault
}

func (r resolver) invalid(key string, reason error) error {
	return fmt.Errorf("invalid %s %q (source: %s): %w", key, fmt.Sprint(r.v.Get(key)), r.sourceOf(key), reason)
}
