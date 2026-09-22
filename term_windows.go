//go:build windows

package editor

import (
	"io"
	"os"
	"sync"

	"golang.org/x/term"
)

// terminate ends the editor. Windows has no graceful signal to send a console
// process that we can rely on, so this is the kill that exec.Cmd would
// otherwise issue itself once WaitDelay elapsed.
func terminate(p *os.Process) error {
	return p.Kill()
}

// protectTerminal saves the console state so an editor that leaves it modified
// cannot strand the shell. There are no signals to trap here. The returned
// func is safe to call more than once.
func protectTerminal(in io.Reader) func() {
	undo := func() {}

	if f, ok := file(in); ok {
		fd := int(f.Fd())
		if term.IsTerminal(fd) {
			if state, err := term.GetState(fd); err == nil {
				undo = func() { _ = term.Restore(fd, state) }
			}
		}
	}

	return sync.OnceFunc(undo)
}
