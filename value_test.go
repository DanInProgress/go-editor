package editor

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// textValue implements the text encoding interfaces but neither the appender
// nor the pointer cases, so it exercises the middle of the priority chain.
type textValue struct{ s string }

func (t textValue) MarshalText() ([]byte, error) { return []byte(t.s), nil }

func (t *textValue) UnmarshalText(b []byte) error {
	t.s = string(b)
	return nil
}

// appenderValue implements the appender interfaces, which outrank the
// marshaler ones. It records which path was taken.
type appenderValue struct {
	s        string
	appended bool
}

func (a *appenderValue) AppendText(b []byte) ([]byte, error) {
	a.appended = true
	return append(b, a.s...), nil
}

func (a *appenderValue) MarshalText() ([]byte, error) { return []byte("marshaler"), nil }

func (a *appenderValue) UnmarshalText(b []byte) error {
	a.s = string(b)
	return nil
}

func TestReadFrom(t *testing.T) {
	t.Run("byte slice pointer", func(t *testing.T) {
		v := []byte("hello")
		read, err := readFrom(&v)
		require.NoError(t, err)

		got, err := read()
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), got)
	})

	t.Run("string pointer", func(t *testing.T) {
		v := "hello"
		read, err := readFrom(&v)
		require.NoError(t, err)

		got, err := read()
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), got)
	})

	t.Run("appender outranks marshaler", func(t *testing.T) {
		v := &appenderValue{s: "appender"}
		read, err := readFrom(v)
		require.NoError(t, err)

		got, err := read()
		require.NoError(t, err)
		assert.Equal(t, []byte("appender"), got)
		assert.True(t, v.appended, "expected AppendText to be preferred over MarshalText")
	})

	t.Run("text marshaler", func(t *testing.T) {
		read, err := readFrom(&textValue{s: "hello"})
		require.NoError(t, err)

		got, err := read()
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), got)
	})

	t.Run("io reader", func(t *testing.T) {
		read, err := readFrom(strings.NewReader("hello"))
		require.NoError(t, err)

		got, err := read()
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), got)
	})

	t.Run("unsupported type names the accepted set", func(t *testing.T) {
		_, err := readFrom(42)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot read bytes from int")
		assert.Contains(t, err.Error(), acceptedTypes)
	})
}

func TestWriteTo(t *testing.T) {
	t.Run("byte slice pointer", func(t *testing.T) {
		v := []byte("before")
		write, err := writeTo(&v)
		require.NoError(t, err)

		require.NoError(t, write([]byte("after")))
		assert.Equal(t, []byte("after"), v)
	})

	t.Run("string pointer", func(t *testing.T) {
		v := "before"
		write, err := writeTo(&v)
		require.NoError(t, err)

		require.NoError(t, write([]byte("after")))
		assert.Equal(t, "after", v)
	})

	t.Run("text unmarshaler", func(t *testing.T) {
		v := &textValue{s: "before"}
		write, err := writeTo(v)
		require.NoError(t, err)

		require.NoError(t, write([]byte("after")))
		assert.Equal(t, "after", v.s)
	})

	t.Run("io writer", func(t *testing.T) {
		var buf bytes.Buffer
		write, err := writeTo(&buf)
		require.NoError(t, err)

		require.NoError(t, write([]byte("after")))
		assert.Equal(t, "after", buf.String())
	})

	t.Run("unsupported type names the accepted set", func(t *testing.T) {
		_, err := writeTo(42)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot write bytes back to int")
		assert.Contains(t, err.Error(), acceptedTypes)
	})

	// Bytes could be taken from a []byte, but there would be no way to put
	// the edited ones back, so it is rejected on the write side.
	t.Run("non-pointer byte slice is not accepted", func(t *testing.T) {
		_, err := writeTo([]byte("hello"))
		require.Error(t, err)
	})
}

// A value that can supply bytes but cannot take them back must be rejected
// before the editor launches, not after the user has done the work.
func TestBindRejectsReadOnlyValueUpFront(t *testing.T) {
	r := strings.NewReader("hello")

	_, readErr := readFrom(r)
	_, writeErr := writeTo(r)

	require.NoError(t, readErr)
	require.Error(t, writeErr)

	assert.Contains(t, errors.Join(readErr, writeErr).Error(), "cannot write bytes back to")
}
