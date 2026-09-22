package editor

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubEditor installs an executable script that stands in for the user's
// editor and points $EDITOR at it. The script is handed the file as $1, and
// shares a directory with any files the test wants it to touch.
func stubEditor(t *testing.T, body string) (dir string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("the stub editor is a POSIX shell script")
	}

	clearEnv(t)
	dir = t.TempDir()
	path := filepath.Join(dir, "stub-editor")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755))
	t.Setenv("EDITOR", path)

	return dir
}

// quiet returns an Editor with the notice off and the editor's output out of
// the test log.
func quiet(t *testing.T) *Editor {
	t.Helper()

	e := New()
	e.Quiet = true
	e.Stdin = nil
	e.Stdout = io.Discard
	return e
}

func TestLaunchEditsCallerOwnedFile(t *testing.T) {
	stubEditor(t, `printf 'edited' > "$1"`)

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0o600))

	require.NoError(t, quiet(t).Launch(t.Context(), path))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, []byte("edited"), got, "the file is the caller's and must keep what the editor wrote")
}

func TestLaunchPropagatesEditorFailure(t *testing.T) {
	stubEditor(t, `exit 3`)

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0o600))

	err := quiet(t).Launch(t.Context(), path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "editor")

	var exit *exec.ExitError
	require.ErrorAs(t, err, &exit)
	assert.Equal(t, 3, exit.ExitCode())
}

// EDITOR=: is how a user says "run this flow, but do not open anything".
func TestLaunchDoesNothingForNoopEditor(t *testing.T) {
	clearEnv(t)
	t.Setenv("EDITOR", NoopEditor)

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0o600))

	require.NoError(t, quiet(t).Launch(t.Context(), path))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, []byte("original"), got)
}

func TestLaunchFailsBeforeRunningWhenEditorMissing(t *testing.T) {
	clearEnv(t)
	t.Setenv("EDITOR", "editor-that-does-not-exist")

	err := quiet(t).Launch(t.Context(), filepath.Join(t.TempDir(), "config.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// A value the shell has to parse still receives the file as an argument, even
// when it ends in a pipe, and even when the path needs quoting.
func TestLaunchPassesThePathToAShellValue(t *testing.T) {
	clearEnv(t)
	t.Setenv("EDITOR", "printf 'edited' | tee")

	path := filepath.Join(t.TempDir(), "my config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0o600))

	e := quiet(t)
	require.NoError(t, e.Launch(t.Context(), path))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, []byte("edited"), got)
}

func TestLaunchCancellationReturnsCause(t *testing.T) {
	// exec, so that the signal reaches the sleep rather than a shell that
	// would leave it orphaned.
	stubEditor(t, `exec sleep 30`)

	cause := errors.New("user asked to stop")
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)

	time.AfterFunc(100*time.Millisecond, func() { cancel(cause) })

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0o600))

	start := time.Now()
	err := quiet(t).Launch(ctx, path)

	require.ErrorIs(t, err, cause)
	assert.Less(t, time.Since(start), 10*time.Second, "the editor should have been terminated, not waited out")
}

// terminal makes isTerminal report true for the duration of the test, so the
// notice can be exercised without a pty.
func terminal(t *testing.T) {
	t.Helper()

	original := isTerminal
	t.Cleanup(func() { isTerminal = original })
	isTerminal = func(io.Writer) bool { return true }
}

func TestLaunchPrintsAndErasesTheNotice(t *testing.T) {
	stubEditor(t, `exit 0`)
	terminal(t)

	var stderr bytes.Buffer
	e := New()
	e.Stdout, e.Stderr = io.Discard, &stderr

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, nil, 0o600))
	require.NoError(t, e.Launch(t.Context(), path))

	assert.Contains(t, stderr.String(), waitMessage)
	assert.Contains(t, stderr.String(), eraseLine)
}

func TestLaunchLeavesTheNoticeOnADumbTerminal(t *testing.T) {
	stubEditor(t, `exit 0`)
	terminal(t)
	t.Setenv("TERM", "dumb")

	var stderr bytes.Buffer
	e := New()
	e.Stdout, e.Stderr = io.Discard, &stderr

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, nil, 0o600))
	require.NoError(t, e.Launch(t.Context(), path))

	assert.Equal(t, waitMessage+"\n", stderr.String(), "nothing can be erased on a dumb terminal")
}

func TestLaunchSkipsTheNoticeForANoopEditor(t *testing.T) {
	clearEnv(t)
	t.Setenv("EDITOR", NoopEditor)
	terminal(t)

	var stderr bytes.Buffer
	e := New()
	e.Stderr = &stderr

	require.NoError(t, e.Launch(t.Context(), filepath.Join(t.TempDir(), "config.yaml")))
	assert.Empty(t, stderr.String(), "there is no wait to explain")
}

func TestLaunchQuietSuppressesTheNotice(t *testing.T) {
	stubEditor(t, `exit 0`)
	terminal(t)

	var stderr bytes.Buffer
	e := New()
	e.Quiet = true
	e.Stdout, e.Stderr = io.Discard, &stderr

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, nil, 0o600))
	require.NoError(t, e.Launch(t.Context(), path))

	assert.Empty(t, stderr.String())
}
