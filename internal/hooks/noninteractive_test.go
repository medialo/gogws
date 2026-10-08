package hooks

import (
	"path/filepath"
	"testing"
)

type failingReader struct{ t *testing.T }

func (r failingReader) Read([]byte) (int, error) {
	r.t.Fatal("stdin must not be read in non-interactive mode")
	return 0, nil
}

func TestApprove_NonInteractiveSkipsWithoutReadingStdin(t *testing.T) {
	isolateHome(t)
	SetTrustMode(TrustModeAsk)
	withPromptInput(t, failingReader{t})
	canPrompt = func() bool { return false }

	dir := t.TempDir()
	path := filepath.Join(dir, "post-ff.sh")
	writeHook(t, path, "echo hi")

	ok, err := approve(&HookInfo{Name: HookPostFF, Path: path, Origin: OriginLocal}, dir)
	if err != nil || ok {
		t.Fatalf("approve = %v, %v; want skipped", ok, err)
	}
	if state, _, _ := HookTrustState(path); state != TrustStateNew {
		t.Fatalf("state = %v; want still new", state)
	}
}
