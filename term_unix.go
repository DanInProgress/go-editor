//go:build !windows

package editor

import (
	"io"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"

	"golang.org/x/term"
)

// terminate asks the editor to exit. exec.Cmd escalates to a kill on its own
// once WaitDelay elapses, so this only needs to be the polite half.
func terminate(p *os.Process) error {
	return p.Signal(syscall.SIGTERM)
}

// protectTerminal saves the terminal state and shields this process from the
// signals an editor is apt to generate, so that a ^C in vim neither kills the
// calling program nor leaves the shell in raw mode. The returned func undoes
// both and is safe to call more than once.
func protectTerminal(in io.Reader) func() {
	var undo []func()

	if f, ok := file(in); ok {
		fd := int(f.Fd())
		if term.IsTerminal(fd) {
			if state, err := term.GetState(fd); err == nil {
				undo = append(undo, func() { _ = term.Restore(fd, state) })
			}
		}
	}

	// Trap rather than ignore: signal.Stop restores whatever disposition was
	// in place before, where signal.Reset would clobber a handler the rest of
	// the program installed. Nothing drains the channel -- delivery to a full
	// buffer is dropped, which is the point.
	trapped := make(chan os.Signal, 1)
	signal.Notify(trapped, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	undo = append(undo, func() { signal.Stop(trapped) })

	return sync.OnceFunc(func() {
		for _, f := range slices.Backward(undo) {
			f()
		}
	})
}
