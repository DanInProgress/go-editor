package editor

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clearEnv removes the variables the default sources consult, so a test starts
// from a known state whatever the machine running it has set.
func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	t.Setenv("TERM", "xterm")
}

func TestResolvePrefersCallerVariable(t *testing.T) {
	clearEnv(t)
	t.Setenv("MYAPP_EDITOR", "true")
	t.Setenv("VISUAL", "false")
	t.Setenv("EDITOR", "false")

	cmd, err := New("MYAPP_EDITOR").resolve()
	require.NoError(t, err)
	assert.Equal(t, "true", cmd.raw)
	assert.False(t, cmd.shell)
}

func TestResolvePrefersVisualOverEditor(t *testing.T) {
	clearEnv(t)
	t.Setenv("VISUAL", "true")
	t.Setenv("EDITOR", "false")

	cmd, err := New().resolve()
	require.NoError(t, err)
	assert.Equal(t, "true", cmd.raw)
}

func TestResolveFallsThroughEmptyVariables(t *testing.T) {
	clearEnv(t)
	t.Setenv("EDITOR", "true")

	cmd, err := New("MYAPP_EDITOR").resolve()
	require.NoError(t, err)
	assert.Equal(t, "true", cmd.raw)
}

func TestResolveFallsBackToPlatformEditor(t *testing.T) {
	clearEnv(t)

	e := New()
	// vi is not guaranteed to exist on a build agent.
	e.Sources[len(e.Sources)-1] = Source{Value: "true", Visual: true}

	cmd, err := e.resolve()
	require.NoError(t, err)
	assert.Equal(t, "true", cmd.raw)
}

// A dumb terminal cannot run a full-screen editor, so $VISUAL and the fallback
// are passed over while $EDITOR, which promises nothing about the terminal, is
// not.
func TestResolveSkipsVisualSourcesOnDumbTerminal(t *testing.T) {
	clearEnv(t)
	t.Setenv("TERM", "dumb")
	t.Setenv("VISUAL", "false")
	t.Setenv("EDITOR", "true")

	cmd, err := New().resolve()
	require.NoError(t, err)
	assert.Equal(t, "true", cmd.raw)
}

func TestResolveHonoursCallerVariableOnDumbTerminal(t *testing.T) {
	clearEnv(t)
	t.Setenv("TERM", "dumb")
	t.Setenv("MYAPP_EDITOR", "true")

	cmd, err := New("MYAPP_EDITOR").resolve()
	require.NoError(t, err)
	assert.Equal(t, "true", cmd.raw)
}

func TestResolveFailsOnDumbTerminalWithNothingLeft(t *testing.T) {
	clearEnv(t)
	t.Setenv("TERM", "dumb")

	_, err := New().resolve()
	require.ErrorIs(t, err, ErrNoEditor)
	assert.Contains(t, err.Error(), "$EDITOR")
	assert.Contains(t, err.Error(), "$TERM is dumb")
	assert.NotContains(t, err.Error(), "$VISUAL", "a variable that was not consulted must not be suggested")
}

func TestResolveRejectsEmptyConfiguration(t *testing.T) {
	clearEnv(t)

	e := New()
	e.Sources = nil

	_, err := e.resolve()
	require.ErrorIs(t, err, ErrNoEditor)
}

func TestResolveRecognisesNoopEditor(t *testing.T) {
	clearEnv(t)
	t.Setenv("EDITOR", NoopEditor)

	cmd, err := New().resolve()
	require.NoError(t, err)
	assert.True(t, cmd.noop)
	assert.Empty(t, cmd.argv, "nothing should be resolved to run")
}

func TestResolveTrimsSurroundingSpace(t *testing.T) {
	clearEnv(t)
	t.Setenv("EDITOR", "  true  ")

	cmd, err := New().resolve()
	require.NoError(t, err)
	assert.Equal(t, "true", cmd.raw)
	assert.False(t, cmd.shell)
}

func TestResolveResolvesBareProgramOnPath(t *testing.T) {
	clearEnv(t)
	t.Setenv("EDITOR", "true")

	cmd, err := New().resolve()
	require.NoError(t, err)

	want, err := exec.LookPath("true")
	require.NoError(t, err)
	assert.Equal(t, []string{want}, cmd.argv)
	assert.True(t, filepath.IsAbs(cmd.argv[0]))
}

func TestResolveRejectsMissingEditor(t *testing.T) {
	clearEnv(t)
	t.Setenv("EDITOR", "editor-that-does-not-exist")

	_, err := New().resolve()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// Anything the shell would treat as syntax goes to the shell, rather than
// being split into arguments here.
func TestResolveSendsShellSyntaxToShell(t *testing.T) {
	clearEnv(t)

	for _, raw := range []string{
		"code --wait",
		`emacsclient -a ''`,
		"cat > /dev/null",
		"f() { true; }; f",
		"~/bin/edit",
	} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("EDITOR", raw)

			cmd, err := New().resolve()
			require.NoError(t, err)
			assert.True(t, cmd.shell, "%q should have been handed to the shell", raw)
			assert.Equal(t, raw, cmd.raw)
		})
	}
}

func TestResolveReportsAMissingShell(t *testing.T) {
	clearEnv(t)
	t.Setenv("EDITOR", "code --wait")

	e := New()
	e.Shell = []string{"shell-that-does-not-exist", "-c"}

	_, err := e.resolve()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shell")
	assert.Contains(t, err.Error(), "not found")
}

func TestCommandArgsDoesNotMutateReceiver(t *testing.T) {
	c := command{raw: "vim", argv: []string{"vim"}}
	_ = c.args("/tmp/a")
	_ = c.args("/tmp/b")
	assert.Equal(t, []string{"vim"}, c.argv)
}

func TestArgsPassThePathPositionally(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("cmd takes its command line as one string; see shell_windows.go")
	}
	clearEnv(t)
	t.Setenv("EDITOR", "cat > out")

	cmd, err := New().resolve()
	require.NoError(t, err)
	assert.Equal(t,
		[]string{"/bin/sh", "-c", `cat > out "$@"`, "cat > out", "/tmp/my file.json"},
		cmd.args("/tmp/my file.json"))
}
