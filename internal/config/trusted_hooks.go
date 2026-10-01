package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const TrustedHooksFile = "trusted-hooks.yaml"

type TrustedHook struct {
	Path   string `yaml:"path"`
	SHA256 string `yaml:"sha256"`
}

type trustedHooksDocument struct {
	Hooks []TrustedHook `yaml:"hooks"`
}

func trustedHooksPath() (string, error) {
	dir, err := GetUserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, TrustedHooksFile), nil
}

func LoadTrustedHooks() ([]TrustedHook, error) {
	path, err := trustedHooksPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var doc trustedHooksDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc.Hooks, nil
}

func FindTrustedHook(path string) (TrustedHook, bool) {
	hooks, err := LoadTrustedHooks()
	if err != nil {
		return TrustedHook{}, false
	}
	for _, h := range hooks {
		if h.Path == path {
			return h, true
		}
	}
	return TrustedHook{}, false
}

func TrustHook(path, sha256 string) error {
	hooks, err := LoadTrustedHooks()
	if err != nil {
		return err
	}
	updated := false
	for i := range hooks {
		if hooks[i].Path == path {
			hooks[i].SHA256 = sha256
			updated = true
		}
	}
	if !updated {
		hooks = append(hooks, TrustedHook{Path: path, SHA256: sha256})
	}

	dir, err := GetUserConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	file, err := trustedHooksPath()
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(trustedHooksDocument{Hooks: hooks})
	if err != nil {
		return err
	}
	return os.WriteFile(file, data, 0644)
}
