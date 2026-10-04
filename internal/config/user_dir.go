package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

const (
	UserConfigDir  = ".gws"
	UserConfigFile = "config.yaml"
)

func GetUserConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, UserConfigDir), nil
}

func GetUserConfigPath() (string, error) {
	configDir, err := GetUserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, UserConfigFile), nil
}

func GetUserHooksDir() (string, error) {
	configDir, err := GetUserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "hooks"), nil
}

func getEnvVarName(key string) string {
	switch key {
	case "github-token":
		return "GITHUB_TOKEN"
	case "gitlab-token":
		return "GITLAB_TOKEN"
	default:
		return ""
	}
}

func GetProviderToken(provider string) string {
	envVar := getEnvVarName(provider + "-token")
	if envVar == "" {
		return ""
	}
	return os.Getenv(envVar)
}

type ConfigFile struct {
	Parallel          *int     `yaml:"parallel,omitempty"`
	Format            string   `yaml:"format,omitempty"`
	Theme             string   `yaml:"theme,omitempty"`
	NoColor           *bool    `yaml:"no-color,omitempty"`
	StopOnError       *bool    `yaml:"stop-on-error,omitempty"`
	ProviderCacheTTL  string   `yaml:"provider-cache-ttl,omitempty"`
	TrustedWorkspaces []string `yaml:"trusted-workspaces,omitempty"`
}

func SetPreference(key, raw string) error {
	pk, ok := LookupPreferenceKey(key)
	if !ok || key == KeyTrustedWorkspaces {
		return fmt.Errorf("%s cannot be set with this command", key)
	}
	if pk.Validate != nil {
		if err := pk.Validate(raw); err != nil {
			return fmt.Errorf("invalid %s %q: %w", key, raw, err)
		}
	}

	file, err := loadConfigFile()
	if err != nil {
		return err
	}

	switch key {
	case KeyParallel:
		n, _ := strconv.Atoi(raw)
		file.Parallel = &n
	case KeyFormat:
		file.Format = raw
	case KeyTheme:
		file.Theme = raw
	case KeyNoColor:
		b, _ := strconv.ParseBool(raw)
		file.NoColor = &b
	case KeyStopOnError:
		b, _ := strconv.ParseBool(raw)
		file.StopOnError = &b
	case KeyProviderCacheTTL:
		file.ProviderCacheTTL = raw
	}

	return saveConfigFile(file)
}

func AddTrustedWorkspace(pattern string) error {
	file, err := loadConfigFile()
	if err != nil {
		return err
	}

	for _, existing := range file.TrustedWorkspaces {
		if existing == pattern {
			return nil
		}
	}

	file.TrustedWorkspaces = append(file.TrustedWorkspaces, pattern)
	return saveConfigFile(file)
}

func loadConfigFile() (*ConfigFile, error) {
	path, err := GetUserConfigPath()
	if err != nil {
		return nil, err
	}

	file := &ConfigFile{}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return file, nil
	}
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, file); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return file, nil
}

func saveConfigFile(file *ConfigFile) error {
	dir, err := GetUserConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path, err := GetUserConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(file)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
