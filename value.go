package editor

import (
	"encoding"
	"fmt"
	"io"
)

// acceptedTypes is repeated in the errors below so that a caller who passes
// the wrong thing learns what the right things are without reading the source.
const acceptedTypes = "*[]byte, *string, an encoding Appender/Marshaler/Unmarshaler, or an io.Reader/io.Writer"

// readFrom resolves how v supplies the bytes that go into the editor. The
// cases are ordered by priority: byte slices are the common case and win
// outright, the encoding interfaces serve types that already implement them,
// and io.Reader is the escape hatch.
func readFrom(v any) (func() ([]byte, error), error) {
	switch t := v.(type) {
	case *[]byte:
		return func() ([]byte, error) { return *t, nil }, nil
	case *string:
		return func() ([]byte, error) { return []byte(*t), nil }, nil
	case encoding.BinaryAppender:
		return func() ([]byte, error) { return t.AppendBinary(nil) }, nil
	case encoding.TextAppender:
		return func() ([]byte, error) { return t.AppendText(nil) }, nil
	case encoding.BinaryMarshaler:
		return t.MarshalBinary, nil
	case encoding.TextMarshaler:
		return t.MarshalText, nil
	case io.Reader:
		return func() ([]byte, error) { return io.ReadAll(t) }, nil
	}

	return nil, fmt.Errorf("cannot read bytes from %T: want %s", v, acceptedTypes)
}

// writeTo resolves how v takes the edited bytes back, mirroring readFrom.
func writeTo(v any) (func([]byte) error, error) {
	switch t := v.(type) {
	case *[]byte:
		return func(b []byte) error { *t = b; return nil }, nil
	case *string:
		return func(b []byte) error { *t = string(b); return nil }, nil
	case encoding.BinaryUnmarshaler:
		return t.UnmarshalBinary, nil
	case encoding.TextUnmarshaler:
		return t.UnmarshalText, nil
	case io.Writer:
		return func(b []byte) error { _, err := t.Write(b); return err }, nil
	}

	return nil, fmt.Errorf("cannot write bytes back to %T: want %s", v, acceptedTypes)
}
