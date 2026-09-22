# go-editor

A package for CLI tools to open files in their preferred editor.

This is a standalone fork of [`k8s.io/kubectl/pkg/cmd/util/editor`][upstream],
extracted so it can be used without depending on Kubernetes.

## Install

```sh
go get github.com/DanInProgress/go-editor
```

## Usage

```go
import editor "github.com/DanInProgress/go-editor"

e := editor.NewDefaultEditor([]string{"MYAPP_EDITOR", "EDITOR"})

// Open an existing file, restoring terminal state on exit or interrupt.
if err := e.Launch("config.yaml"); err != nil {
    return err
}

// Or edit a temporary file seeded from a reader; the caller owns cleanup.
contents, path, err := e.LaunchTempFile("myapp-", ".yaml", bytes.NewBufferString(original))
if err != nil {
    return err
}
defer os.Remove(path)
```

`NewDefaultEditor` takes the environment variables to consult, in order. If none
are set it falls back to `vi` (`notepad` on Windows). An editor value containing
spaces is split into arguments; one containing quotes or backslashes is handed to
`$SHELL` (`cmd` on Windows) instead.

## Status

Pre-1.0. This initial import is a near-verbatim copy of the upstream code; the
API is expected to change before 1.0.

Deliberate changes from upstream:

- the package moved to the module root;
- `k8s.io/kubectl/pkg/util/term` and `k8s.io/kubectl/pkg/util/interrupt` were
  vendored into `internal/` (`term` trimmed to the TTY handling used here, which
  drops the `k8s.io/cli-runtime` and `k8s.io/apimachinery` dependencies);
- the `k8s.io/klog/v2` verbose log line in `Launch` became a `log/slog` debug
  record on the default logger, so nothing is emitted unless the embedding
  program enables debug logging.

`github.com/moby/term` is the only remaining direct dependency.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Commits follow
[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/), and CI
builds and tests on Linux, macOS and Windows.

## License

Apache 2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE) for attribution of the
upstream code.

[upstream]: https://github.com/kubernetes/kubectl/blob/master/pkg/cmd/util/editor/editor.go
