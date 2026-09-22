//go:build windows

package editor

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
)

func defaultShell() []string { return []string{"cmd", "/C"} }

// notepad is not a terminal editor, so unlike vi it survives $TERM=dumb.
func fallbackSource() Source { return Source{Value: "notepad"} }

func isCmd(name string) bool {
	base := strings.ToLower(filepath.Base(name))
	return base == "cmd" || base == "cmd.exe"
}

// shellCommand wraps an editor value that needs shell parsing. cmd takes its
// whole invocation as one string, so the path is quoted into the command
// rather than passed positionally as it is on Unix; point [Editor.Shell] at a
// POSIX shell to get the positional form.
func (e *Editor) shellCommand(raw string) (command, error) {
	argv := slices.Clone(e.shellArgv())
	if _, err := exec.LookPath(argv[0]); err != nil {
		return command{}, fmt.Errorf("shell %q not found, needed to run editor %q: %w", argv[0], raw, err)
	}

	if isCmd(argv[0]) {
		return command{raw: raw, argv: append(argv, raw), shell: true}, nil
	}

	argv = append(argv, raw+` "$@"`, raw)
	return command{raw: raw, argv: argv, shell: true}, nil
}

// args returns the full argv for editing path.
func (c command) args(path string) []string {
	argv := slices.Clone(c.argv)
	if !c.shell || !isCmd(argv[0]) {
		return append(argv, path)
	}

	// Quote the command and the path together, as cmd requires.
	// See https://stackoverflow.com/a/6378038.
	last := len(argv) - 1
	argv[last] = fmt.Sprintf(`"%s %s"`, argv[last], cmdQuoteArg(path))
	return argv
}

// configure hands cmd its command line verbatim. exec.Cmd would otherwise
// quote each argument the way the C runtime parses them, and cmd does not.
func configure(proc *exec.Cmd, c command, argv []string) {
	if c.shell && isCmd(argv[0]) {
		proc.SysProcAttr = &syscall.SysProcAttr{CmdLine: strings.Join(argv, " ")}
	}
}

// cmdQuoteArg encloses arg in double quotes, doubling any it contains.
func cmdQuoteArg(arg string) string {
	return `"` + strings.ReplaceAll(arg, `"`, `""`) + `"`
}
