package editor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ErrNoEditor is returned when no source yields an editor to run.
var ErrNoEditor = errors.New("no editor configured")

// shellMeta are the characters whose presence in an editor value make it a
// shell command rather than the name of a program. The set is git's, space and
// tab included, which is what makes EDITOR='emacsclient -a ""' work.
const shellMeta = "|&;<>()$`\\\"' \t\n*?[#~=%"

// command is a resolved editor invocation, before the file path is appended.
type command struct {
	raw  string   // as configured, for messages
	argv []string // without the path

	shell bool // argv runs raw through a shell
	noop  bool // raw was NoopEditor; there is nothing to run
}

// resolve determines the editor command line from the sources. It fails when
// no editor is configured or the configured one cannot be found, so that a
// caller can bail out before the user has spent any time editing.
func (e *Editor) resolve() (command, error) {
	dumb := dumbTerminal()

	var raw string
	for _, s := range e.Sources {
		if s.Visual && dumb {
			continue
		}

		v := s.Value
		if s.Env != "" {
			v = os.Getenv(s.Env)
		}
		if v = strings.TrimSpace(v); v != "" {
			raw = v
			break
		}
	}

	switch {
	case raw == "":
		return command{}, e.errNoEditor(dumb)
	case raw == NoopEditor:
		return command{raw: raw, noop: true}, nil
	case strings.ContainsAny(raw, shellMeta):
		return e.shellCommand(raw)
	}

	// Resolve a bare program now, both to fail early and so that the exec is
	// not repeating the lookup against an environment that may have moved on.
	path, err := exec.LookPath(raw)
	if err != nil {
		return command{}, fmt.Errorf("editor %q not found: %w", raw, err)
	}

	return command{raw: raw, argv: []string{path}}, nil
}

// errNoEditor names the variables that would have been consulted, and says so
// when a dumb terminal is why there was nothing left to try.
func (e *Editor) errNoEditor(dumb bool) error {
	var names []string
	var gated bool
	for _, s := range e.Sources {
		switch {
		case s.Visual && dumb:
			gated = true
		case s.Env != "":
			names = append(names, "$"+s.Env)
		}
	}

	switch {
	case len(names) == 0:
		return ErrNoEditor
	case gated:
		return fmt.Errorf("%w: set %s ($TERM is dumb, so full-screen editors were skipped)",
			ErrNoEditor, strings.Join(names, " or "))
	default:
		return fmt.Errorf("%w: set %s", ErrNoEditor, strings.Join(names, " or "))
	}
}

// shellArgv returns the argv prefix that runs a command string through a
// shell, ending in the flag that introduces that string.
func (e *Editor) shellArgv() []string {
	if len(e.Shell) > 0 {
		return e.Shell
	}
	return defaultShell()
}

// dumbTerminal reports whether $TERM says the terminal cannot address the
// cursor. Only the literal value counts: an unset $TERM is common in a
// pipeline and says nothing about what the terminal can do.
func dumbTerminal() bool {
	return os.Getenv("TERM") == "dumb"
}

func (c command) String() string { return c.raw }
