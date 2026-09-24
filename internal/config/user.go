package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	UserConfigDir  = ".gws"
	UserConfigFile = "config.yaml"
)

type ConfigSource string

const (
	SourceDefault ConfigSource = "default"
	SourceFile    ConfigSource = "file"
	SourceEnv     ConfigSource = "env"
)

type ConfigValue[T any] struct {
	Value  T
	Source ConfigSource
}

type UserConfig struct {
	TrustedWorkspaces []string `yaml:"trusted-workspaces,omitempty"`
	ProviderCacheTTL  string   `yaml:"provider-cache-ttl,omitempty"`
}

type UserConfigResolved struct {
	TrustedWorkspaces ConfigValue[[]string]
	ProviderCacheTTL  ConfigValue[string]
}

// DefaultProviderCacheTTL is used whenever provider-cache-ttl is unset.
const DefaultProviderCacheTTL = "24h"

func GetUserConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, UserConfigDir, UserConfigFile), nil
}

func GetUserConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, UserConfigDir), nil
}

func LoadUserConfigResolved() (*UserConfigResolved, error) {
	resolved := &UserConfigResolved{
		TrustedWorkspaces: ConfigValue[[]string]{Value: []string{}, Source: SourceDefault},
		ProviderCacheTTL:  ConfigValue[string]{Value: DefaultProviderCacheTTL, Source: SourceDefault},
	}

	configPath, err := GetUserConfigPath()
	if err != nil {
		return resolved, nil
	}

	data, err := os.ReadFile(configPath)
	if err == nil {
		var fileCfg UserConfig
		if yaml.Unmarshal(data, &fileCfg) == nil {
			if fileCfg.TrustedWorkspaces != nil {
				resolved.TrustedWorkspaces = ConfigValue[[]string]{Value: fileCfg.TrustedWorkspaces, Source: SourceFile}
			}
			if fileCfg.ProviderCacheTTL != "" {
				resolved.ProviderCacheTTL = ConfigValue[string]{Value: fileCfg.ProviderCacheTTL, Source: SourceFile}
			}
		}
	}

	return resolved, nil
}

func LoadUserConfig() (*UserConfig, error) {
	resolved, err := LoadUserConfigResolved()
	if err != nil {
		return nil, err
	}
	cfg := &UserConfig{
		TrustedWorkspaces: resolved.TrustedWorkspaces.Value,
	}
	// Only carry the TTL over if it actually came from the file: unlike
	// TrustedWorkspaces' empty-slice default (naturally omitted by
	// omitempty), the TTL default is a non-empty string, so resolving it
	// here would round-trip the default into the saved file on any
	// unrelated SetUserConfigValue/AddTrustedWorkspace call.
	if resolved.ProviderCacheTTL.Source == SourceFile {
		cfg.ProviderCacheTTL = resolved.ProviderCacheTTL.Value
	}
	return cfg, nil
}

func SaveUserConfig(cfg *UserConfig) error {
	configDir, err := GetUserConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath, err := GetUserConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func SetUserConfigValue(key string, value interface{}) error {
	cfg, err := LoadUserConfig()
	if err != nil {
		return err
	}

	switch key {
	case "trusted-workspaces":
		if v, ok := value.([]string); ok {
			cfg.TrustedWorkspaces = v
		}
	case "provider-cache-ttl":
		if v, ok := value.(string); ok {
			cfg.ProviderCacheTTL = v
		}
	}

	return SaveUserConfig(cfg)
}

func AddTrustedWorkspace(pattern string) error {
	cfg, err := LoadUserConfig()
	if err != nil {
		return err
	}

	for _, existing := range cfg.TrustedWorkspaces {
		if existing == pattern {
			return nil
		}
	}

	cfg.TrustedWorkspaces = append(cfg.TrustedWorkspaces, pattern)
	return SaveUserConfig(cfg)
}

func GetUserConfigValue(key string) (interface{}, error) {
	cfg, err := LoadUserConfig()
	if err != nil {
		return nil, err
	}

	switch key {
	case "trusted-workspaces":
		return cfg.TrustedWorkspaces, nil
	case "provider-cache-ttl":
		return cfg.ProviderCacheTTL, nil
	default:
		return nil, nil
	}
}

func GetAvailableConfigKeys() []string {
	return []string{"trusted-workspaces", "provider-cache-ttl"}
}

func GetEnvVarName(key string) string {
	switch key {
	case "github-token":
		return "GITHUB_TOKEN"
	case "gitlab-token":
		return "GITLAB_TOKEN"
	default:
		return ""
	}
}

// GetProviderToken returns the API token for the given provider name
// ("github", "gitlab"), read from its conventional environment variable
// (matching the gh/glab CLI tools). Tokens are never persisted to
// UserConfig/~/.gws/config.yaml — env var only. Returns "" if unset,
// which callers treat as "make unauthenticated requests".
func GetProviderToken(provider string) string {
	envVar := GetEnvVarName(provider + "-token")
	if envVar == "" {
		return ""
	}
	return os.Getenv(envVar)
}

func GetUserHooksDir() (string, error) {
	configDir, err := GetUserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "hooks"), nil
}
