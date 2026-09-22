# Contributing

## Commit messages

This repository uses [Conventional Commits][cc]. Pull request titles and every
commit in a pull request are checked by the `conventional-commits` workflow:

```
<type>[optional scope][!]: <description>
```

| Type | Use for |
| --- | --- |
| `feat` | a new capability in the public API |
| `fix` | a bug fix |
| `docs` | documentation only |
| `refactor` | a change that is neither a fix nor a feature |
| `perf` | a performance change |
| `test` | tests only |
| `build` | the module, its dependencies, or the release |
| `ci` | workflows and CI scripts |
| `chore` | anything else that does not touch `./...` |
| `style` | formatting only |
| `revert` | reverting an earlier commit |

A `!` before the colon, or a `BREAKING CHANGE:` footer, marks a breaking change.
Until 1.0 the API may break in a minor release, but breaks should still be
marked. Keep the description under 100 characters, lowercase, and in the
imperative mood.

Scopes are optional; use the package or area touched, for example
`fix(shell): ...` or `build(deps): ...`.

Dependency updates are opened by Renovate, configured in `renovate.json`.

## Before pushing

CI runs these on Linux, macOS and Windows against the module's minimum Go
version and current stable:

```sh
gofmt -l .
go vet ./...
go test -race ./...
go mod tidy && git diff --exit-code -- go.mod go.sum
```

## Relationship to kubectl and git

The package began as a fork of `k8s.io/kubectl/pkg/cmd/util/editor`; see
[NOTICE](NOTICE). It has since been rewritten, and its observable behaviour --
which variables are consulted, when a value goes to a shell, what `:` means --
follows git's `launch_editor`. Prefer matching git over inventing something new:
these are behaviours users already have expectations about.

[cc]: https://www.conventionalcommits.org/en/v1.0.0/
