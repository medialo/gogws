package hooks

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type interpreter struct {
	candidates []string
	args       []string
}

var interpreters = map[string]interpreter{
	".sh":   {candidates: []string{"bash", "sh"}},
	".bash": {candidates: []string{"bash"}},
	".ps1":  {candidates: []string{"pwsh", "powershell"}, args: []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File"}},
	".py":   {candidates: []string{"python3", "python"}},
	".cmd":  {candidates: []string{"cmd"}, args: []string{"/c"}},
	".bat":  {candidates: []string{"cmd"}, args: []string{"/c"}},
	".js":   {candidates: []string{"node"}},
}

var lookPath = exec.LookPath

func findBinary(name string) (string, error) {
	if runtime.GOOS != "windows" || (name != "bash" && name != "sh") {
		return lookPath(name)
	}
	if bin, err := lookPath(name); err == nil && !strings.Contains(strings.ToLower(bin), `\windows\system32\`) {
		return bin, nil
	}
	if git, err := lookPath("git"); err == nil {
		gitRoot := filepath.Dir(filepath.Dir(git))
		bin := filepath.Join(gitRoot, "bin", name+".exe")
		if isHookFile(bin) {
			return bin, nil
		}
	}
	return "", exec.ErrNotFound
}

func scriptArg(bin, path string) string {
	switch strings.TrimSuffix(strings.ToLower(filepath.Base(bin)), ".exe") {
	case "bash", "sh":
		return filepath.ToSlash(path)
	}
	return path
}

func resolveInterpreter(path string) ([]string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if in, ok := interpreters[ext]; ok {
		for _, c := range in.candidates {
			if bin, err := findBinary(c); err == nil {
				return append(append([]string{bin}, in.args...), scriptArg(bin, path)), nil
			}
		}
		return nil, fmt.Errorf("%s not found to run %s", strings.Join(in.candidates, "/"), filepath.Base(path))
	}

	if runtime.GOOS != "windows" {
		return []string{path}, nil
	}

	shebang, err := readShebang(path)
	if err != nil {
		return nil, err
	}
	if len(shebang) == 0 {
		return nil, fmt.Errorf("cannot run %s on windows: add a file extension (.sh, .ps1, ...) or a shebang", filepath.Base(path))
	}
	name := filepath.Base(shebang[0])
	args := shebang[1:]
	if name == "env" && len(args) > 0 {
		name, args = args[0], args[1:]
	}
	bin, err := findBinary(name)
	if err != nil {
		return nil, fmt.Errorf("%s not found to run %s", name, filepath.Base(path))
	}
	return append(append([]string{bin}, args...), scriptArg(bin, path)), nil
}

func readShebang(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	line, err := bufio.NewReader(f).ReadString('\n')
	if err != nil && line == "" {
		return nil, nil
	}
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "#!") {
		return nil, nil
	}
	return strings.Fields(strings.TrimPrefix(line, "#!")), nil
}

func buildCommand(ctx context.Context, hook *HookInfo, dir string, env []string) (*exec.Cmd, error) {
	argv, err := resolveInterpreter(hook.Path)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	return cmd, nil
}
