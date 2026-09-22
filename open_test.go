package editor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenRoundTrip(t *testing.T) {
	stubEditor(t, `printf 'edited' > "$1"`)

	v := []byte("original")
	changed, err := quiet(t).Open(t.Context(), "test", ".json", &v)

	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, []byte("edited"), v)
}

func TestOpenReportsUnchanged(t *testing.T) {
	stubEditor(t, `exit 0`) // opened, saved nothing

	v := []byte("original")
	changed, err := quiet(t).Open(t.Context(), "test", ".json", &v)

	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, []byte("original"), v)
}

// An empty buffer is a legitimate value, not an error. Whether it means
// "abort" or "clear this field" is the caller's call, so Open just reports it.
func TestOpenAllowsEmptyBuffer(t *testing.T) {
	stubEditor(t, `: > "$1"`)

	v := []byte("original")
	changed, err := quiet(t).Open(t.Context(), "test", ".json", &v)

	require.NoError(t, err)
	assert.True(t, changed)
	assert.Empty(t, v)
}

func TestOpenWritesStringPointer(t *testing.T) {
	stubEditor(t, `printf 'edited' > "$1"`)

	v := "original"
	changed, err := quiet(t).Open(t.Context(), "test", ".txt", &v)

	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, "edited", v)
}

func TestOpenReadsAndWritesIOBuffer(t *testing.T) {
	stubEditor(t, `printf 'edited' > "$1"`)

	buf := bytes.NewBufferString("original")
	changed, err := quiet(t).Open(t.Context(), "test", ".txt", buf)

	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, "edited", buf.String())
}

// recordPath makes the stub editor report the path it was given.
func recordPath(t *testing.T) func() string {
	t.Helper()

	dir := stubEditor(t, `printf '%s' "$1" > "$(dirname "$0")/record"`)
	return func() string {
		b, err := os.ReadFile(filepath.Join(dir, "record"))
		require.NoError(t, err)
		return string(b)
	}
}

func TestOpenNamesAndExtendsTempFile(t *testing.T) {
	path := recordPath(t)

	v := []byte("x")
	_, err := quiet(t).Open(t.Context(), "myapp-check", ".json", &v)
	require.NoError(t, err)

	base := filepath.Base(path())
	assert.True(t, strings.HasPrefix(base, "myapp-check"), "got %q", base)
	assert.True(t, strings.HasSuffix(base, ".json"), "got %q", base)
}

func TestOpenAddsLeadingDotToExtension(t *testing.T) {
	path := recordPath(t)

	v := []byte("x")
	_, err := quiet(t).Open(t.Context(), "test", "yaml", &v)
	require.NoError(t, err)

	assert.True(t, strings.HasSuffix(path(), ".yaml"), "got %q", path())
}

// A name is a filename, not a path: a caller must not be able to steer the
// temporary file out of the temp directory.
func TestOpenSanitizesName(t *testing.T) {
	path := recordPath(t)

	v := []byte("x")
	_, err := quiet(t).Open(t.Context(), "../../escape", ".txt", &v)
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(os.TempDir(), filepath.Base(path())), path())
	assert.True(t, strings.HasPrefix(filepath.Base(path()), "escape"), "got %q", path())
}

// The file is ours, so unlike Launch we take it away again.
func TestOpenRemovesTempFile(t *testing.T) {
	path := recordPath(t)

	v := []byte("x")
	_, err := quiet(t).Open(t.Context(), "test", ".txt", &v)
	require.NoError(t, err)

	_, err = os.Stat(path())
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestOpenPropagatesEditorFailure(t *testing.T) {
	stubEditor(t, `exit 3`)

	v := []byte("original")
	changed, err := quiet(t).Open(t.Context(), "test", ".txt", &v)

	require.Error(t, err)
	assert.False(t, changed)
	assert.Equal(t, []byte("original"), v, "the value must not be touched when the editor fails")
}

// Resolution failures must land before the user has done any work.
func TestOpenFailsBeforeEditingWhenEditorMissing(t *testing.T) {
	clearEnv(t)
	t.Setenv("EDITOR", "editor-that-does-not-exist")

	v := []byte("original")
	changed, err := quiet(t).Open(t.Context(), "test", ".txt", &v)

	require.Error(t, err)
	assert.False(t, changed)
	assert.Contains(t, err.Error(), "not found")
	assert.Equal(t, []byte("original"), v)
}

func TestOpenRejectsUnsupportedValue(t *testing.T) {
	stubEditor(t, `printf 'edited' > "$1"`)

	changed, err := quiet(t).Open(t.Context(), "test", ".txt", 42)

	require.Error(t, err)
	assert.False(t, changed)
	assert.Contains(t, err.Error(), acceptedTypes)
}

// A value that can be read but not written back is rejected up front, rather
// than after the edit.
func TestOpenRejectsValueItCannotWriteBack(t *testing.T) {
	stubEditor(t, `printf 'edited' > "$1"`)

	_, err := quiet(t).Open(t.Context(), "test", ".txt", strings.NewReader("original"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot write bytes back to")
}

// EDITOR=: leaves the staged value exactly as it was.
func TestOpenDoesNothingForNoopEditor(t *testing.T) {
	clearEnv(t)
	t.Setenv("EDITOR", NoopEditor)

	v := []byte("original")
	changed, err := quiet(t).Open(t.Context(), "test", ".txt", &v)

	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, []byte("original"), v)
}

func TestDefaultIsUsedByPackageLevelOpen(t *testing.T) {
	clearEnv(t)

	original := Default
	t.Cleanup(func() { Default = original })

	Default = New()
	Default.Sources = []Source{{Value: "editor-that-does-not-exist"}}

	v := []byte("original")
	_, err := Open(t.Context(), "test", ".txt", &v)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// The bug this package exists to fix: the kubectl editor was always given an
// empty environment list, so it fell through to vi and $EDITOR was ignored,
// while every command's help text claimed otherwise.
func TestOpenHonoursEDITOR(t *testing.T) {
	stubEditor(t, `printf 'edited by EDITOR' > "$1"`)

	v := []byte("original")
	changed, err := quiet(t).Open(t.Context(), "test", ".txt", &v)

	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, []byte("edited by EDITOR"), v)
}
