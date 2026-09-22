// Package editor opens a file, or a value in memory, in the user's editor.
//
// It replaces k8s.io/kubectl/pkg/cmd/util/editor, whose NewDefaultEditor took
// the environment variables to consult as an argument and silently fell back
// to vi when that argument was empty. Selection, shell handling and signal
// handling here follow git's launch_editor instead, which is the behaviour
// users already expect:
//
//   - $VISUAL wins over $EDITOR, and both over the vi fallback;
//   - $VISUAL and vi are skipped when $TERM is dumb, since neither can drive a
//     terminal that cannot address the cursor;
//   - an editor of ":" means "do not edit" and succeeds without touching
//     anything;
//   - an editor holding shell metacharacters is run through a shell, with the
//     file passed positionally rather than pasted into the command string;
//   - the terminal is left as the editor found it, and a ^C aimed at the
//     editor does not take the calling program down with it.
//
// There are two entry points, differing in who owns the file. [Editor.Launch]
// edits a file the caller owns:
//
//	err := editor.Launch(ctx, "config.yaml")
//
// [Editor.Open] owns the file itself: it stages a value in a temporary file,
// edits that, reads the result back into the same value, and removes the file.
//
//	b, err := json.MarshalIndent(check, "", "  ")
//	if err != nil {
//		return err
//	}
//	changed, err := editor.Open(ctx, "myapp-check", ".json", &b)
//
// Encoding is left to the caller. This package moves bytes through a file and
// nothing more.
package editor
