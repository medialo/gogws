package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func useTempHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func TestSetPreference(t *testing.T) {
	useTempHome(t)
	clearPreferenceEnv(t)

	for key, value := range map[string]string{
		KeyNoColor:          "false",
		KeyParallel:         "8",
		KeyFormat:           "yaml",
		KeyProviderCacheTTL: "12h",
	} {
		if err := SetPreference(key, value); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}

	path, _ := GetUserConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "no-color: false") {
		t.Errorf("no-color: false not persisted:\n%s", data)
	}

	prefs, err := LoadPreferences(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if prefs.Parallel.Value != 8 || prefs.Format.Value != "yaml" || prefs.ProviderCacheTTL.Value != 12*time.Hour {
		t.Errorf("prefs = %+v", prefs)
	}
	if prefs.NoColor.Source != SourceFile {
		t.Errorf("no-color source = %s, want file", prefs.NoColor.Source)
	}
}

func TestSetPreferenceRejectsInvalidValues(t *testing.T) {
	useTempHome(t)

	for key, value := range map[string]string{
		KeyParallel:          "0",
		KeyFormat:            "xml",
		KeyNoColor:           "maybe",
		KeyProviderCacheTTL:  "soon",
		KeyTrustedWorkspaces: "/a",
		"unknown":            "x",
	} {
		if err := SetPreference(key, value); err == nil {
			t.Errorf("set %s=%s: expected error", key, value)
		}
	}

	path, _ := GetUserConfigPath()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("config file written despite invalid values")
	}
}

func TestAddTrustedWorkspaceIsIdempotent(t *testing.T) {
	useTempHome(t)

	for range 2 {
		if err := AddTrustedWorkspace("/a"); err != nil {
			t.Fatal(err)
		}
	}

	file, err := loadConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(file.TrustedWorkspaces) != 1 {
		t.Errorf("trusted-workspaces = %v", file.TrustedWorkspaces)
	}
}

func TestUpdateCheckPreference(t *testing.T) {
	useTempHome(t)
	clearPreferenceEnv(t)

	prefs, err := LoadPreferences(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if !prefs.UpdateCheck.Value || prefs.UpdateCheck.Source != SourceDefault {
		t.Fatalf("default update-check = %v (%s), want true (default)", prefs.UpdateCheck.Value, prefs.UpdateCheck.Source)
	}

	if err := SetPreference(KeyUpdateCheck, "false"); err != nil {
		t.Fatal(err)
	}
	prefs, err = LoadPreferences(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if prefs.UpdateCheck.Value || prefs.UpdateCheck.Source != SourceFile {
		t.Fatalf("file update-check = %v (%s), want false (file)", prefs.UpdateCheck.Value, prefs.UpdateCheck.Source)
	}

	t.Setenv(EnvVarName(KeyUpdateCheck), "true")
	prefs, err = LoadPreferences(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if !prefs.UpdateCheck.Value || prefs.UpdateCheck.Source != SourceEnv {
		t.Fatalf("env update-check = %v (%s), want true (env)", prefs.UpdateCheck.Value, prefs.UpdateCheck.Source)
	}
}
