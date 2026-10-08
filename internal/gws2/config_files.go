package gws2

import (
	"os"
	"path/filepath"
)

type ConfigFiles struct {
	Dir       string
	root      map[string]string
	configDir map[string]string
}

func ScanConfigFiles(dir string) ConfigFiles {
	files := ConfigFiles{Dir: dir, root: listNames(dir)}
	if name, ok := files.root[PathKey(ConfigDirName)]; ok {
		files.configDir = listNames(filepath.Join(dir, name))
	}
	return files
}

func NewConfigFiles(dir string, rootNames, configDirNames []string) ConfigFiles {
	return ConfigFiles{Dir: dir, root: indexNames(rootNames), configDir: indexNames(configDirNames)}
}

func (f ConfigFiles) Lookup(inConfigDir bool, name string) (string, bool) {
	names := f.root
	base := f.Dir
	if inConfigDir {
		names = f.configDir
		base = filepath.Join(f.Dir, ConfigDirName)
	}
	actual, ok := names[PathKey(name)]
	if !ok {
		return "", false
	}
	return filepath.Join(base, actual), true
}

func listNames(dir string) map[string]string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	names := make(map[string]string, len(entries))
	for _, entry := range entries {
		names[PathKey(entry.Name())] = entry.Name()
	}
	return names
}

func indexNames(names []string) map[string]string {
	index := make(map[string]string, len(names))
	for _, name := range names {
		index[PathKey(name)] = name
	}
	return index
}
