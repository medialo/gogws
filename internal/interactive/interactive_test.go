package interactive

import (
	"os"
	"testing"
)

func withTTY(t *testing.T, tty bool) {
	t.Helper()
	previous := isTTY
	isTTY = func(*os.File) bool { return tty }
	reset()
	t.Cleanup(func() {
		isTTY = previous
		reset()
	})
}

func TestEnabled(t *testing.T) {
	cases := []struct {
		name    string
		tty     bool
		ci      string
		noInter string
		disable bool
		want    bool
	}{
		{"terminal", true, "", "", false, true},
		{"no terminal", false, "", "", false, false},
		{"ci true", true, "true", "", false, false},
		{"ci 1", true, "1", "", false, false},
		{"ci false", true, "false", "", false, true},
		{"env no interactive", true, "", "yes", false, false},
		{"flag", true, "", "", true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withTTY(t, c.tty)
			t.Setenv("CI", c.ci)
			t.Setenv(EnvNoInteractive, c.noInter)
			if c.disable {
				Disable()
			}
			if got := Enabled(); got != c.want {
				t.Fatalf("Enabled() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestEnabled_ComputedOnce(t *testing.T) {
	withTTY(t, true)
	t.Setenv("CI", "")
	t.Setenv(EnvNoInteractive, "")

	calls := 0
	isTTY = func(*os.File) bool {
		calls++
		return true
	}
	Enabled()
	Enabled()
	if calls != 2 {
		t.Fatalf("terminal checked %d times, want 2 (stdout + stdin, once)", calls)
	}
}
