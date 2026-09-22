package editor

import (
	"fmt"
	"io"
	"sync"

	"golang.org/x/term"
)

const (
	waitMessage = "waiting for your editor to close the file..."
	eraseLine   = "\r\x1b[K"
)

// isTerminal is a variable so tests can exercise the notice without a pty.
var isTerminal = func(w io.Writer) bool {
	f, ok := file(w)
	return ok && term.IsTerminal(int(f.Fd()))
}

// notice explains why the program appears to have stopped, and returns a func
// that takes the message back off the screen. It goes to stderr so that a
// program whose stdout is captured does not emit it into the capture, and only
// to a terminal, where there is someone to read it.
//
// The message names no file: the caller may not have one yet, and the point is
// to explain the wait rather than its subject.
func (e *Editor) notice() func() {
	w := e.stderr()
	if e.Quiet || !isTerminal(w) {
		return func() {}
	}

	if dumbTerminal() {
		// Nothing can be erased here, so the message gets its own line and
		// stays on screen.
		fmt.Fprintln(w, waitMessage)
		return func() {}
	}

	fmt.Fprint(w, waitMessage)
	return sync.OnceFunc(func() { fmt.Fprint(w, eraseLine) })
}
