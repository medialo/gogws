package hookfile

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

var Hooks = []string{
	"pre-init", "post-init",
	"pre-update", "post-update",
	"pre-clone", "post-clone",
	"pre-fetch", "post-fetch",
	"pre-ff", "post-ff",
	"pre-check", "post-check",
}

var OSDirs = []string{"windows", "linux", "darwin"}

var Extensions = []string{"sh", "bash", "ps1", "py", "cmd", "bat", "js"}

type Key struct {
	Target string
	Hook   string
}

func (k Key) String() string {
	if k.Target == "" {
		return k.Hook
	}
	return k.Target + "." + k.Hook
}

type File struct {
	Key
	Path string
	Ext  string
}

type Conflict struct {
	Key
	Dir   string
	Files []string
}

func (c Conflict) String() string {
	names := make([]string, len(c.Files))
	for i, f := range c.Files {
		names[i] = filepath.Base(f)
	}
	return fmt.Sprintf("%s in %s: %s", c.Key, c.Dir, strings.Join(names, ", "))
}

type ConflictError struct {
	Conflict
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflicting hooks for %s — keep only one, then check with 'gogws doctor run HookConflict'", e.Conflict)
}

func Parse(path string) (File, bool) {
	name := filepath.Base(path)
	ext := ""
	if dot := strings.LastIndex(name, "."); dot >= 0 && slices.Contains(Extensions, name[dot+1:]) {
		ext = name[dot+1:]
		name = name[:dot]
	}

	target := ""
	hook := name
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		target = name[:dot]
		hook = name[dot+1:]
	}
	if !slices.Contains(Hooks, hook) {
		return File{}, false
	}
	f := File{Path: path, Ext: ext}
	f.Target, f.Hook = target, hook
	return f, true
}

func Scan(dir string) ([]File, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var files []File
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if f, ok := Parse(filepath.Join(dir, e.Name())); ok {
			files = append(files, f)
		}
	}
	return files, nil
}

func Conflicts(dir string, files []File) []Conflict {
	byKey := make(map[Key][]string)
	for _, f := range files {
		byKey[f.Key] = append(byKey[f.Key], f.Path)
	}
	var conflicts []Conflict
	for key, paths := range byKey {
		if len(paths) > 1 {
			sort.Strings(paths)
			conflicts = append(conflicts, Conflict{Key: key, Dir: dir, Files: paths})
		}
	}
	sort.Slice(conflicts, func(i, j int) bool {
		return conflicts[i].Key.String() < conflicts[j].Key.String()
	})
	return conflicts
}

func SearchDirs(base, goos string) []string {
	if slices.Contains(OSDirs, goos) {
		return []string{filepath.Join(base, goos), base}
	}
	return []string{base}
}

type Scanner func(dir string) ([]File, error)

func CachedScanner() Scanner {
	cache := make(map[string][]File)
	return func(dir string) ([]File, error) {
		if files, ok := cache[dir]; ok {
			return files, nil
		}
		files, err := Scan(dir)
		if err != nil {
			return nil, err
		}
		cache[dir] = files
		return files, nil
	}
}

func Resolve(scan Scanner, dirs []string, target, hook string) (*File, error) {
	key := Key{Target: target, Hook: hook}
	for _, dir := range dirs {
		files, err := scan(dir)
		if err != nil {
			return nil, err
		}
		var matches []File
		for _, f := range files {
			if f.Key == key {
				matches = append(matches, f)
			}
		}
		switch len(matches) {
		case 0:
			continue
		case 1:
			return &matches[0], nil
		default:
			return nil, &ConflictError{Conflicts(dir, matches)[0]}
		}
	}
	return nil, nil
}
