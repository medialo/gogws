package updatecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	Interval       = 24 * time.Hour
	stateFileName  = "update-check.json"
	RequestTimeout = 3 * time.Second
	releasesURL    = "https://github.com/medialo/gogws/releases/latest"
	goInstallCmd   = "go install github.com/medialo/gogws/cmd/gogws@latest"
	scoopUpdateCmd = "scoop update gogws"
)

var (
	latestReleaseURL = "https://api.github.com/repos/medialo/gogws/releases/latest"
	httpClient       = &http.Client{Timeout: RequestTimeout}
	now              = time.Now
	executablePath   = currentExecutable
)

type state struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest,omitempty"`
}

type Options struct {
	Current  string
	StateDir string
	Token    string
	Force    bool
}

type Check struct {
	current string
	done    chan struct{}
	latest  string
}

func Start(ctx context.Context, opts Options) *Check {
	if !semver.IsValid(canonical(opts.Current)) || opts.StateDir == "" {
		return nil
	}

	statePath := filepath.Join(opts.StateDir, stateFileName)
	c := &Check{current: opts.Current, done: make(chan struct{})}

	previous, _ := readState(statePath)
	if !opts.Force && now().Sub(previous.CheckedAt) < Interval {
		c.latest = previous.Latest
		close(c.done)
		return c
	}

	go func() {
		defer close(c.done)
		latest, err := fetchLatest(ctx, opts.Current, opts.Token)
		if err != nil {
			latest = previous.Latest
		}
		c.latest = latest
		_ = writeState(statePath, state{CheckedAt: now(), Latest: latest})
	}()
	return c
}

func (c *Check) Notice(deadline time.Time) string {
	if c == nil {
		return ""
	}
	wait := time.Until(deadline)
	if wait < 0 {
		wait = 0
	}
	select {
	case <-c.done:
	case <-time.After(wait):
		return ""
	}
	if !IsNewer(c.latest, c.current) {
		return ""
	}
	return fmt.Sprintf("A new version of gogws is available: %s → %s\nUpdate with: %s",
		strings.TrimPrefix(c.current, "v"), strings.TrimPrefix(c.latest, "v"), UpdateInstruction())
}

func IsNewer(latest, current string) bool {
	l, c := canonical(latest), canonical(current)
	if !semver.IsValid(l) || !semver.IsValid(c) {
		return false
	}
	if semver.Prerelease(l) != "" && semver.Prerelease(c) == "" {
		return false
	}
	return semver.Compare(l, c) > 0
}

func UpdateInstruction() string {
	path := executablePath()
	if path == "" {
		return releasesURL
	}
	lower := strings.ToLower(filepath.ToSlash(path))
	if strings.Contains(lower, "/scoop/") {
		return scoopUpdateCmd
	}
	for _, dir := range goBinDirs() {
		if dir != "" && strings.EqualFold(filepath.Clean(filepath.Dir(path)), filepath.Clean(dir)) {
			return goInstallCmd
		}
	}
	return releasesURL
}

func canonical(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func fetchLatest(ctx context.Context, current, token string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gogws/"+strings.TrimPrefix(current, "v"))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	if !semver.IsValid(canonical(release.TagName)) {
		return "", errors.New("invalid release tag " + release.TagName)
	}
	return strings.TrimPrefix(release.TagName, "v"), nil
}

func readState(path string) (state, error) {
	var s state
	data, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(data, &s)
	return s, err
}

func writeState(path string, s state) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func currentExecutable() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return path
}

func goBinDirs() []string {
	dirs := []string{os.Getenv("GOBIN")}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		for _, p := range filepath.SplitList(gopath) {
			dirs = append(dirs, filepath.Join(p, "bin"))
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "go", "bin"))
	}
	return dirs
}
