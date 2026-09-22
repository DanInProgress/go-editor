//go:build !windows

package editor

import (
	"fmt"
	"os/exec"
	"slices"
)

func defaultShell() []string { return []string{"/bin/sh", "-c"} }

func fallbackSource() Source { return Source{Value: "vi", Visual: true} }

// shellCommand wraps an editor value that needs shell parsing:
//
//	/bin/sh -c 'VALUE "$@"' VALUE /path/to/file
//
// The path is passed positionally rather than pasted into the command string,
// so a value ending in a pipe or a redirect still receives the file, and a
// path containing shell syntax is never parsed as part of the command. VALUE
// is repeated as $0 so a diagnostic from the shell names the editor.
func (e *Editor) shellCommand(raw string) (command, error) {
	argv := slices.Clone(e.shellArgv())
	if _, err := exec.LookPath(argv[0]); err != nil {
		return command{}, fmt.Errorf("shell %q not found, needed to run editor %q: %w", argv[0], raw, err)
	}

	argv = append(argv, raw+` "$@"`, raw)
	return command{raw: raw, argv: argv, shell: true}, nil
}

// args returns the full argv for editing path.
func (c command) args(path string) []string {
	return append(slices.Clone(c.argv), path)
}

// configure is where Windows assembles a cmd command line by hand. There is
// nothing to do here.
func configure(*exec.Cmd, command, []string) {}
