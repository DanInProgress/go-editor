package editor

import (
	"io"
	"os"
	"time"
)

const (
	// DefaultWaitDelay is how long an editor is given to exit after being
	// asked to terminate, before it is killed outright.
	DefaultWaitDelay = 5 * time.Second

	// NoopEditor is the editor value meaning "do not edit". It is the
	// shell's null command, so EDITOR=: is the conventional way to run an
	// editing flow without opening anything.
	NoopEditor = ":"
)

// Source is one place an editor command line can come from.
type Source struct {
	// Env is an environment variable to read. If it is empty, Value is used
	// instead.
	Env string

	// Value is a literal command line. A fallback is just a source that
	// does not depend on the environment.
	Value string

	// Visual marks a source naming a full-screen editor, which is skipped
	// when $TERM is dumb. $VISUAL means precisely this, which is why it
	// exists alongside $EDITOR.
	Visual bool
}

// Editor holds the settings used to locate and run an editor. The zero value
// has no sources and so cannot resolve one; call [New].
type Editor struct {
	// Sources are consulted in order, and the first non-empty value wins.
	Sources []Source

	// Shell is the argv prefix used for an editor value that needs a shell,
	// ending in the flag that introduces the command string. Nil means
	// {"/bin/sh", "-c"}, or {"cmd", "/C"} on Windows.
	//
	// It is deliberately not $SHELL: editor values are written in POSIX
	// shell syntax by convention, and an interactive shell that parses them
	// differently -- fish, say -- would break values that work everywhere
	// else.
	Shell []string

	// WaitDelay bounds how long Launch waits for the editor to exit after
	// the context is cancelled, before killing it.
	WaitDelay time.Duration

	// Quiet suppresses the notice printed while the editor is open.
	Quiet bool

	// Stdin, Stdout and Stderr are handed to the editor. Nil means the
	// corresponding os file.
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// New returns an Editor that consults the given environment variables first,
// then $VISUAL, $EDITOR and the platform fallback.
//
// Variables named by the caller are honoured whatever $TERM says. A program
// offering its own MYAPP_EDITOR cannot know whether the editor named there
// needs a capable terminal, and guessing wrong would ignore an explicit
// instruction.
func New(envVars ...string) *Editor {
	sources := make([]Source, 0, len(envVars)+3)
	for _, name := range envVars {
		sources = append(sources, Source{Env: name})
	}
	sources = append(sources,
		Source{Env: "VISUAL", Visual: true},
		Source{Env: "EDITOR"},
		fallbackSource(),
	)

	return &Editor{
		Sources:   sources,
		WaitDelay: DefaultWaitDelay,
	}
}

// Default is the Editor used by the package-level [Launch] and [Open].
var Default = New()

func (e *Editor) stdin() io.Reader {
	if e.Stdin != nil {
		return e.Stdin
	}
	return os.Stdin
}

func (e *Editor) stdout() io.Writer {
	if e.Stdout != nil {
		return e.Stdout
	}
	return os.Stdout
}

func (e *Editor) stderr() io.Writer {
	if e.Stderr != nil {
		return e.Stderr
	}
	return os.Stderr
}

// file returns v as an *os.File, for the terminal queries that need a file
// descriptor. Anything else is not a terminal, which is the answer those
// queries want anyway.
func file(v any) (*os.File, bool) {
	f, ok := v.(*os.File)
	return f, ok
}
