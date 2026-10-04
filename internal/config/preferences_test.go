package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/medialo/gogws/internal/gws2"
	"github.com/spf13/pflag"
)

func clearPreferenceEnv(t *testing.T) {
	t.Helper()
	for _, k := range PreferenceKeys {
		t.Setenv(EnvVarName(k.Name), "")
	}
}

func newFlags(t *testing.T, args ...string) *pflag.FlagSet {
	t.Helper()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.Int(KeyParallel, gws2.DefaultParallel, "")
	fs.String(KeyFormat, DefaultFormat, "")
	fs.String(KeyTheme, "", "")
	fs.Bool(KeyNoColor, false, "")
	fs.Bool(KeyStopOnError, false, "")
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	return fs
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if content != "" {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func TestLoadPreferencesPrecedence(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		env        map[string]string
		args       []string
		wantValue  int
		wantSource ConfigSource
	}{
		{name: "default", wantValue: gws2.DefaultParallel, wantSource: SourceDefault},
		{name: "file", file: "parallel: 8\n", wantValue: 8, wantSource: SourceFile},
		{name: "env over file", file: "parallel: 8\n", env: map[string]string{"GOGWS_PARALLEL": "3"}, wantValue: 3, wantSource: SourceEnv},
		{name: "flag over env", file: "parallel: 8\n", env: map[string]string{"GOGWS_PARALLEL": "3"}, args: []string{"--parallel=2"}, wantValue: 2, wantSource: SourceFlag},
		{name: "flag equal to default", file: "parallel: 8\n", args: []string{"--parallel=5"}, wantValue: 5, wantSource: SourceFlag},
		{name: "empty env ignored", file: "parallel: 8\n", env: map[string]string{"GOGWS_PARALLEL": ""}, wantValue: 8, wantSource: SourceFile},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearPreferenceEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			prefs, err := LoadPreferences(newFlags(t, tt.args...), writeConfig(t, tt.file))
			if err != nil {
				t.Fatal(err)
			}
			if prefs.Parallel.Value != tt.wantValue || prefs.Parallel.Source != tt.wantSource {
				t.Errorf("parallel = %d (%s), want %d (%s)", prefs.Parallel.Value, prefs.Parallel.Source, tt.wantValue, tt.wantSource)
			}
		})
	}
}

func TestLoadPreferencesAllKeys(t *testing.T) {
	clearPreferenceEnv(t)
	t.Setenv("GOGWS_PROVIDER_CACHE_TTL", "2h")
	t.Setenv("GOGWS_NO_COLOR", "true")

	file := "format: json\ntheme: my-theme.yaml\nstop-on-error: true\ntrusted-workspaces:\n  - /a\n"
	prefs, err := LoadPreferences(newFlags(t), writeConfig(t, file))
	if err != nil {
		t.Fatal(err)
	}

	if prefs.Format.Value != "json" || prefs.Format.Source != SourceFile {
		t.Errorf("format = %v", prefs.Format)
	}
	if prefs.Theme.Value != "my-theme.yaml" || prefs.Theme.Source != SourceFile {
		t.Errorf("theme = %v", prefs.Theme)
	}
	if !prefs.StopOnError.Value || prefs.StopOnError.Source != SourceFile {
		t.Errorf("stop-on-error = %v", prefs.StopOnError)
	}
	if !prefs.NoColor.Value || prefs.NoColor.Source != SourceEnv {
		t.Errorf("no-color = %v", prefs.NoColor)
	}
	if prefs.ProviderCacheTTL.Value != 2*time.Hour || prefs.ProviderCacheTTL.Source != SourceEnv {
		t.Errorf("provider-cache-ttl = %v", prefs.ProviderCacheTTL)
	}
	if len(prefs.TrustedWorkspaces.Value) != 1 || prefs.TrustedWorkspaces.Source != SourceFile {
		t.Errorf("trusted-workspaces = %v", prefs.TrustedWorkspaces)
	}
}

func TestLoadPreferencesWithoutFlags(t *testing.T) {
	clearPreferenceEnv(t)
	prefs, err := LoadPreferences(nil, writeConfig(t, "parallel: 9\n"))
	if err != nil {
		t.Fatal(err)
	}
	if prefs.Parallel.Value != 9 || prefs.Parallel.Source != SourceFile {
		t.Errorf("parallel = %v", prefs.Parallel)
	}
}

func TestLoadPreferencesErrors(t *testing.T) {
	tests := []struct {
		name string
		file string
		env  map[string]string
		want string
	}{
		{name: "invalid yaml", file: "parallel: [\n", want: "failed to read config file"},
		{name: "invalid parallel", file: "parallel: many\n", want: "invalid parallel"},
		{name: "zero parallel", file: "parallel: 0\n", want: "invalid parallel"},
		{name: "negative parallel from env", env: map[string]string{"GOGWS_PARALLEL": "-1"}, want: "source: env"},
		{name: "invalid format", file: "format: xml\n", want: "invalid format"},
		{name: "invalid bool", file: "no-color: maybe\n", want: "invalid no-color"},
		{name: "invalid ttl", file: "provider-cache-ttl: soon\n", want: "invalid provider-cache-ttl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearPreferenceEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			_, err := LoadPreferences(newFlags(t), writeConfig(t, tt.file))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want containing %q", err, tt.want)
			}
		})
	}
}
