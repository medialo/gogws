package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestParseProgress(t *testing.T) {
	cases := []struct {
		line string
		want Progress
		ok   bool
	}{
		{"Receiving objects:  45% (450/1000), 1.20 MiB | 2.00 MiB/s", Progress{"Receiving objects", 45, 450, 1000}, true},
		{"remote: Compressing objects: 100% (3/3), done.", Progress{"Compressing objects", 100, 3, 3}, true},
		{"Resolving deltas:   0% (0/12)", Progress{"Resolving deltas", 0, 0, 12}, true},
		{"Cloning into 'api'...", Progress{}, false},
		{"remote: Enumerating objects: 1000, done.", Progress{}, false},
	}
	for _, c := range cases {
		got, ok := parseProgress(c.line)
		if ok != c.ok || got != c.want {
			t.Errorf("parseProgress(%q) = %+v, %v; want %+v, %v", c.line, got, ok, c.want, c.ok)
		}
	}
}

func TestLineNotifyWriter_ProgressDeduplicated(t *testing.T) {
	var logs, progress []string
	w := &lineNotifyWriter{
		notify:   func(s string) { logs = append(logs, s) },
		progress: func(s string) { progress = append(progress, s) },
		now:      func() time.Time { return time.Unix(0, 0) },
	}

	fmt.Fprint(w, "Cloning into 'api'...\n")
	for i := range 1000 {
		fmt.Fprintf(w, "Receiving objects: %3d%% (%d/1000)\r", i/10, i)
	}
	fmt.Fprint(w, "Receiving objects: 100% (1000/1000), done.\n")
	w.Close()

	if len(progress) > 101 {
		t.Fatalf("progress events = %d, want <= 101", len(progress))
	}
	if last, _ := parseProgress(progress[len(progress)-1]); last.Percent != 100 {
		t.Fatalf("last progress = %q, want 100%%", progress[len(progress)-1])
	}
	if len(logs) != 1 || logs[0] != "Cloning into 'api'..." {
		t.Fatalf("logs = %v, want only the non-progress line", logs)
	}
}

func longRunningCommand() *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("powershell", "-NoProfile", "-Command", "Start-Sleep -Seconds 30")
	}
	return exec.Command("sleep", "30")
}

func TestNotifiableCmd_KilledOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := longRunningCommand()

	done := make(chan error, 1)
	go func() { done <- Wrap(cmd).Run(ctx, func(EventType, string) {}) }()

	time.Sleep(300 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "cancelled") {
			t.Fatalf("err = %v, want cancelled", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("command not killed after cancel")
	}
}

func TestNotifiableCmd_UnattendedEnv(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=TestHelperPrintEnv")
	cmd.Env = append(os.Environ(), "GOGWS_HELPER_PRINT_ENV=1")

	var logs []string
	if err := Wrap(cmd).Run(context.Background(), func(_ EventType, s string) { logs = append(logs, s) }); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(logs, "\n")
	for _, want := range []string{"GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=never"} {
		if !strings.Contains(joined, want) {
			t.Errorf("job environment missing %s", want)
		}
	}
}

func TestHelperPrintEnv(t *testing.T) {
	if os.Getenv("GOGWS_HELPER_PRINT_ENV") != "1" {
		t.Skip("helper process")
	}
	fmt.Println("GIT_TERMINAL_PROMPT=" + os.Getenv("GIT_TERMINAL_PROMPT"))
	fmt.Println("GCM_INTERACTIVE=" + os.Getenv("GCM_INTERACTIVE"))
}

func TestConsumeVerbose_Milestones(t *testing.T) {
	m := progressMilestones{}
	var logged []string
	for _, p := range []Progress{
		{"Receiving objects", 0, 0, 100}, {"Receiving objects", 10, 10, 100}, {"Receiving objects", 24, 24, 100},
		{"Receiving objects", 25, 25, 100}, {"Receiving objects", 60, 60, 100}, {"Receiving objects", 100, 100, 100},
		{"Resolving deltas", 3, 3, 100},
	} {
		if m.reached("job", &p) {
			logged = append(logged, fmt.Sprintf("%s %d", p.Phase, p.Percent))
		}
	}
	want := "Receiving objects 0|Receiving objects 25|Receiving objects 60|Receiving objects 100|Resolving deltas 3"
	if strings.Join(logged, "|") != want {
		t.Fatalf("logged = %v, want %s", logged, want)
	}
}
