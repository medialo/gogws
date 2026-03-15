package config

import (
	"log/slog"
	"sync"

	"gogws/internal/gws"
)

type Config struct {
	WorkspaceRoot string
	ThemeFile     string
	Parallel      int
	Format        string
	NoColor       bool
	OnlyChanges   bool
	StopOnError   bool
}

var (
	globalConfig *Config
	configMu     sync.RWMutex
	initialized  bool
)

func Initialize() error {
	slog.Debug("Initializing configuration manager...")

	configMu.Lock()
	defer configMu.Unlock()

	if initialized {
		slog.Debug("Configuration manager already initialized")
		return nil
	}

	cfg, err := load()
	if err != nil {
		return err
	}

	globalConfig = cfg
	initialized = true

	slog.Debug("Configuration loaded", "workspaceRoot", cfg.WorkspaceRoot, "parallel", cfg.Parallel)

	return nil
}

func GetConfig() *Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return globalConfig
}

func MustGetConfig() *Config {
	cfg := GetConfig()
	if cfg == nil {
		panic("config not initialized - call Initialize() first")
	}
	return cfg
}

func IsInitialized() bool {
	configMu.RLock()
	defer configMu.RUnlock()
	return initialized
}

func Reload() error {
	configMu.Lock()
	initialized = false
	configMu.Unlock()
	return Initialize()
}

func ApplyFlags(themeFile string, parallel int, format string, noColor, onlyChanges, stopOnError bool, workingDir string) {
	configMu.Lock()
	defer configMu.Unlock()

	if globalConfig == nil {
		return
	}

	if themeFile != "" {
		globalConfig.ThemeFile = themeFile
	}
	if parallel > 0 {
		globalConfig.Parallel = parallel
	}
	if format != "" {
		globalConfig.Format = format
	}
	if workingDir != "" {
		globalConfig.WorkspaceRoot = workingDir
	}
	globalConfig.NoColor = noColor
	globalConfig.OnlyChanges = onlyChanges
	globalConfig.StopOnError = stopOnError
}

func load() (*Config, error) {
	cfg := &Config{
		Parallel: gws.DefaultParallel,
		Format:   "text",
	}

	wsInfo, err := gws.FindRoot()
	if err != nil {
		return nil, err
	}
	cfg.WorkspaceRoot = wsInfo.Root

	return cfg, nil
}
