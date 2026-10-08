package updatecheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func fakeReleases(t *testing.T, tag string, delay time.Duration, status int) *atomic.Int32 {
	t.Helper()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if delay > 0 {
			time.Sleep(delay)
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		w.Write([]byte(`{"tag_name":"` + tag + `"}`))
	}))
	previous := latestReleaseURL
	latestReleaseURL = server.URL
	t.Cleanup(func() {
		latestReleaseURL = previous
		server.Close()
	})
	return &calls
}

func withExecutable(t *testing.T, path string) {
	t.Helper()
	previous := executablePath
	executablePath = func() string { return path }
	t.Cleanup(func() { executablePath = previous })
}

func writeTestState(t *testing.T, dir string, s state) {
	t.Helper()
	if err := writeState(filepath.Join(dir, stateFileName), s); err != nil {
		t.Fatal(err)
	}
}

func TestStart_FreshCacheSkipsNetwork(t *testing.T) {
	calls := fakeReleases(t, "v9.9.9", 0, http.StatusOK)
	withExecutable(t, "/opt/gogws")
	dir := t.TempDir()
	writeTestState(t, dir, state{CheckedAt: time.Now().Add(-time.Hour), Latest: "1.3.0"})

	notice := Start(context.Background(), Options{Current: "1.2.3", StateDir: dir}).Notice(time.Now().Add(time.Second))

	if calls.Load() != 0 {
		t.Fatalf("network calls = %d, want 0", calls.Load())
	}
	if !strings.Contains(notice, "1.2.3 → 1.3.0") {
		t.Fatalf("notice = %q", notice)
	}
}

func TestStart_StaleCacheFetchesAndSaves(t *testing.T) {
	calls := fakeReleases(t, "v1.4.0", 0, http.StatusOK)
	withExecutable(t, "/opt/gogws")
	dir := t.TempDir()
	writeTestState(t, dir, state{CheckedAt: time.Now().Add(-48 * time.Hour), Latest: "1.3.0"})

	notice := Start(context.Background(), Options{Current: "1.2.3", StateDir: dir}).Notice(time.Now().Add(2 * time.Second))

	if calls.Load() != 1 || !strings.Contains(notice, "1.2.3 → 1.4.0") {
		t.Fatalf("calls = %d, notice = %q", calls.Load(), notice)
	}
	saved, err := readState(filepath.Join(dir, stateFileName))
	if err != nil || saved.Latest != "1.4.0" || time.Since(saved.CheckedAt) > time.Minute {
		t.Fatalf("saved state = %+v, %v", saved, err)
	}
}

func TestStart_NoNoticeWhenUpToDate(t *testing.T) {
	fakeReleases(t, "v1.2.3", 0, http.StatusOK)
	notice := Start(context.Background(), Options{Current: "1.2.3", StateDir: t.TempDir()}).Notice(time.Now().Add(2 * time.Second))
	if notice != "" {
		t.Fatalf("notice = %q, want none", notice)
	}
}

func TestStart_FailureKeepsPreviousLatestAndDelaysRetry(t *testing.T) {
	fakeReleases(t, "", 0, http.StatusInternalServerError)
	withExecutable(t, "/opt/gogws")
	dir := t.TempDir()
	writeTestState(t, dir, state{CheckedAt: time.Now().Add(-48 * time.Hour), Latest: "1.3.0"})

	notice := Start(context.Background(), Options{Current: "1.2.3", StateDir: dir}).Notice(time.Now().Add(2 * time.Second))
	if !strings.Contains(notice, "1.3.0") {
		t.Fatalf("notice = %q, want previous latest", notice)
	}
	saved, _ := readState(filepath.Join(dir, stateFileName))
	if saved.Latest != "1.3.0" || time.Since(saved.CheckedAt) > time.Minute {
		t.Fatalf("saved state = %+v", saved)
	}
}

func TestNotice_ReturnsAtDeadline(t *testing.T) {
	fakeReleases(t, "v1.4.0", 2*time.Second, http.StatusOK)
	check := Start(context.Background(), Options{Current: "1.2.3", StateDir: t.TempDir()})

	start := time.Now()
	notice := check.Notice(time.Now().Add(200 * time.Millisecond))
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("Notice waited %s, want about 200ms", elapsed)
	}
	if notice != "" {
		t.Fatalf("notice = %q, want none before the answer", notice)
	}
}

func TestStart_DevVersionDoesNothing(t *testing.T) {
	calls := fakeReleases(t, "v1.4.0", 0, http.StatusOK)
	dir := t.TempDir()
	if check := Start(context.Background(), Options{Current: "dev", StateDir: dir}); check != nil {
		t.Fatal("expected no check for dev version")
	}
	if calls.Load() != 0 {
		t.Fatal("no network call expected")
	}
	if _, err := os.Stat(filepath.Join(dir, stateFileName)); !os.IsNotExist(err) {
		t.Fatal("no state expected")
	}
	var nilCheck *Check
	if nilCheck.Notice(time.Now()) != "" {
		t.Fatal("nil check must not produce a notice")
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v1.3.0", "1.2.3", true},
		{"1.3.0", "v1.2.3", true},
		{"1.2.3", "1.2.3", false},
		{"1.2.0", "1.2.3", false},
		{"v1.4.0-rc.1", "1.3.0", false},
		{"v1.4.0", "1.4.0-rc.1", true},
		{"", "1.2.3", false},
		{"1.3.0", "dev", false},
	}
	for _, c := range cases {
		if got := IsNewer(c.latest, c.current); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestUpdateInstruction(t *testing.T) {
	gopath := t.TempDir()
	t.Setenv("GOPATH", gopath)
	t.Setenv("GOBIN", "")

	cases := map[string]string{
		filepath.Join("C:", "Users", "me", "scoop", "apps", "gogws", "current", "gogws.exe"): scoopUpdateCmd,
		filepath.Join(gopath, "bin", "gogws"):                                                goInstallCmd,
		filepath.Join(t.TempDir(), "gogws"):                                                  releasesURL,
		"":                                                                                   releasesURL,
	}
	for path, want := range cases {
		withExecutable(t, path)
		if got := UpdateInstruction(); got != want {
			t.Errorf("UpdateInstruction(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestStart_ForceIgnoresFreshCache(t *testing.T) {
	calls := fakeReleases(t, "v1.5.0", 0, http.StatusOK)
	withExecutable(t, "/opt/gogws")
	dir := t.TempDir()
	writeTestState(t, dir, state{CheckedAt: time.Now().Add(-time.Minute), Latest: "1.3.0"})

	notice := Start(context.Background(), Options{Current: "1.2.3", StateDir: dir, Force: true}).Notice(time.Now().Add(2 * time.Second))

	if calls.Load() != 1 || !strings.Contains(notice, "1.2.3 → 1.5.0") {
		t.Fatalf("calls = %d, notice = %q", calls.Load(), notice)
	}
}
