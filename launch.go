package editor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
)

// Launch opens path in the user's editor and waits for it to close. It uses
// Default; see [Editor.Launch].
func Launch(ctx context.Context, path string) error {
	return Default.Launch(ctx, path)
}

// Launch opens path in the user's editor and waits for it to close. The file
// is the caller's: Launch neither creates nor removes it, and leaves whatever
// the editor wrote in place.
//
// If ctx is cancelled the editor is asked to exit, killed if it ignores that,
// and Launch returns the context's cause.
func (e *Editor) Launch(ctx context.Context, path string) error {
	cmd, err := e.resolve()
	if err != nil {
		return err
	}
	return e.launch(ctx, cmd, path)
}

func (e *Editor) launch(ctx context.Context, c command, path string) error {
	if c.noop {
		return nil
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	argv := c.args(abs)
	proc := exec.CommandContext(ctx, argv[0], argv[1:]...)
	proc.Stdin, proc.Stdout, proc.Stderr = e.stdin(), e.stdout(), e.stderr()
	configure(proc, c, argv)

	// Ask the editor to exit rather than killing it outright, so it can put
	// the terminal back and clean up its own swap files. WaitDelay is the
	// escalation to a kill for an editor that ignores the request.
	proc.Cancel = func() error { return terminate(proc.Process) }
	proc.WaitDelay = e.WaitDelay

	// Registered first so the notice is erased last, once the terminal is
	// ours again.
	erase := e.notice()
	defer erase()

	restore := protectTerminal(e.stdin())
	defer restore()

	if err := proc.Run(); err != nil {
		// A cancelled context surfaces as a kill signal from Wait, which says
		// nothing useful; report why we cancelled instead.
		if cause := context.Cause(ctx); cause != nil {
			return cause
		}
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("editor %q not found: %w", c, err)
		}
		return fmt.Errorf("editor %q: %w", c, err)
	}

	return nil
}
