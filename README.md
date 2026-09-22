# go-editor

A package for CLI tools to open files, or values in memory, in the user's
editor.

It began as a standalone fork of [`k8s.io/kubectl/pkg/cmd/util/editor`][upstream]
and has since been rewritten: editor selection, shell handling and signal
handling now follow git's `launch_editor`, which is what users expect from a
terminal program that shells out to an editor.

## Install

```sh
go get github.com/DanInProgress/go-editor
```

## Usage

Two entry points, differing in who owns the file.

`Launch` edits a file the caller owns. It neither creates nor removes it:

```go
import editor "github.com/DanInProgress/go-editor"

if err := editor.Launch(ctx, "config.yaml"); err != nil {
    return err
}
```

`Open` owns the file itself. It stages a value in a temporary file, edits it,
reads the result back into that same value, and removes the file:

```go
b, err := json.MarshalIndent(check, "", "  ")
if err != nil {
    return err
}
changed, err := editor.Open(ctx, "myapp-check", ".json", &b)
```

The value can be a `*[]byte`, a `*string`, anything implementing the `encoding`
appender, marshaler or unmarshaler interfaces, or an `io.Reader` that is also an
`io.Writer`. Encoding is left to the caller; the package moves bytes through a
file and nothing more.

Both take a `context.Context`. If it is cancelled the editor is asked to exit,
killed if it ignores that, and the cause is returned.

To consult your own environment variable first, build an `Editor` with it:

```go
e := editor.New("MYAPP_EDITOR")
changed, err := e.Open(ctx, "myapp-check", ".json", &b)
```

## Behaviour

Sources are consulted in order, and the first non-empty value wins: variables
named by the caller, then `$VISUAL`, then `$EDITOR`, then `vi` (`notepad` on
Windows).

- `$VISUAL` and the `vi` fallback are skipped when `$TERM` is `dumb`, since
  neither can drive a terminal that cannot address the cursor. Variables named
  by the caller and `$EDITOR` are honoured whatever `$TERM` says.
- `EDITOR=:` means "do not edit". It succeeds without opening anything and
  without touching the file.
- A value holding shell metacharacters — a space included, so `code --wait`
  counts — is run through `/bin/sh` (`cmd` on Windows), with the file passed
  positionally as `"$@"` rather than pasted into the command string. A value
  without them is a program name, resolved on `PATH` before anything is staged,
  so a typo fails before the user has done any work.
- While the editor is open, a notice on stderr explains the wait, and is erased
  when the editor closes. Set `Quiet` to suppress it.
- The terminal is restored to the state the editor found it in, and `SIGINT`,
  `SIGQUIT` and `SIGTERM` are trapped for the duration, so a `^C` aimed at the
  editor does not take the calling program down with it.

## Status

Pre-1.0; the API may change. `golang.org/x/term` is the only dependency.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Commits follow
[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/), and CI
builds and tests on Linux, macOS and Windows.

## License

Apache 2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE) for attribution of the
upstream code.

[upstream]: https://github.com/kubernetes/kubectl/blob/master/pkg/cmd/util/editor/editor.go
