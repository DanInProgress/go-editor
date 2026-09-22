package editor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Open edits v in a temporary file. It uses Default; see [Editor.Open].
func Open(ctx context.Context, name, ext string, v any) (changed bool, err error) {
	return Default.Open(ctx, name, ext, v)
}

// Open writes v to a temporary file, opens that file in the user's editor, and
// reads the result back into v. The file is removed before Open returns.
//
// name identifies the caller and becomes the file's prefix, so that files left
// behind by a crash can be traced back to the command that made them. ext is
// the file extension, which is what drives syntax highlighting in the editor;
// a leading dot is added if missing.
//
// v supplies the initial bytes and receives the edited ones. It is written
// back whether or not anything changed; changed reports only whether the bytes
// differ. Deciding what an unchanged -- or empty -- buffer means is left to the
// caller, because only the caller knows whether that is a no-op, an abort, or a
// legitimate empty value.
//
// If the editor fails or ctx is cancelled, v is left alone.
func (e *Editor) Open(ctx context.Context, name, ext string, v any) (changed bool, err error) {
	// Resolve everything that can fail before the user spends time editing.
	// A failure discovered on the way back would throw away their work.
	cmd, err := e.resolve()
	if err != nil {
		return false, err
	}

	read, readErr := readFrom(v)
	write, writeErr := writeTo(v)
	if err := errors.Join(readErr, writeErr); err != nil {
		return false, err
	}

	before, err := read()
	if err != nil {
		return false, fmt.Errorf("reading value for editing: %w", err)
	}

	path, err := writeTempFile(name, ext, before)
	if err != nil {
		return false, err
	}
	defer os.Remove(path)

	if err := e.launch(ctx, cmd, path); err != nil {
		return false, err
	}

	after, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("reading edited file: %w", err)
	}

	if err := write(after); err != nil {
		return false, fmt.Errorf("writing edited value back: %w", err)
	}

	return !bytes.Equal(before, after), nil
}

// tempPattern builds the os.CreateTemp pattern for a staged file. os.CreateTemp
// rejects patterns containing separators, and a caller should not be able to
// steer the file out of the temp directory anyway.
func tempPattern(name, ext string) string {
	name = filepath.Base(name)
	if name == "." || name == string(filepath.Separator) {
		name = "edit"
	}

	if ext != "" {
		if ext[0] != '.' {
			ext = "." + ext
		}
		ext = filepath.Base(ext)
	}

	return name + "*" + ext
}

// writeTempFile stages content in a temporary file named for the caller.
func writeTempFile(name, ext string, content []byte) (_ string, err error) {
	f, err := os.CreateTemp("", tempPattern(name, ext))
	if err != nil {
		return "", fmt.Errorf("creating temp file for editing: %w", err)
	}
	path := f.Name()
	defer func() {
		if err != nil {
			os.Remove(path)
		}
	}()

	_, err = f.Write(content)

	// Close before the editor runs so it can claim the file itself, keeping
	// the write error in preference to anything close has to say.
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", fmt.Errorf("staging file for editing: %w", err)
	}

	return path, nil
}
