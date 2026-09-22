# API design notes

Working notes from the API design conversation that followed the first rewrite
(commit `ea4543f`). Nothing here is implemented yet; the code on the branch is
the *pre-feedback* shape and several decisions below contradict it.

Pick this up by reading "Where we left off" last.

## Two surfaces

The framing that drives everything else:

- **the developer surface / API** -- one set of semantics, should match the API
  a Go developer would expect and want;
- **the user surface** -- should match the prose in
  `Launching_an_Editor_Traced_Behavior.md` accurately.

The central fault in the current code is that the user-facing policy
(`$VISUAL` -> `$EDITOR` -> `vi`, the `$TERM=dumb` gate) was encoded as
developer-facing configuration data (`Sources []Source`).

## Decisions locked in

| Decision | Choice |
| --- | --- |
| Config | Functional options. `editor.New(editor.WithEnv("MYAPP_EDITOR"), ...)`. Opaque struct, no exported fields, no mutable package-level `Default`. |
| Receiver | `Editor` is the receiver, content is the parameter. Interfaces to customize behaviour within the package may end up looking like content constructors, but *using those interfaces directly should not be the norm*. |
| Result | A result struct, for today. Anything added later goes in a functional option instead. Hashing is internal. |
| Decorator | Rejected as the public mechanism -- "too much and overloads and obfuscates our conversion/interface detection". |
| Generics | Rejected: `ed.Edit[editor.Changed](...)` cannot be defaulted. Go has no default type arguments and cannot infer a type parameter from the return position, so every call site would pay the instantiation, including the majority that do not care. |
| Go version | Bump the module to **1.26**. (Frees the `golang.org/x/term` pin at v0.35.0 -- v0.46 requires 1.26 -- and makes `testing/synctest` available.) |
| Tests | Helper-process pattern, as `os/exec` does: the test binary re-execs itself as the editor. Runs on Windows, needs no shell, and the "editor" can assert on its own argv -- which is what actually needs testing about shell wrapping and `"$@"`. |
| Subprocess layer | Exported subpackage in this module. Takes a command string, resolves shell-vs-exec cross-platform, builds the argv, and owns the stream handling. |

### Streaming, not slurping

> if the caller owns the memory then we're in a mode where we shouldn't be
> allocating much and the bytes should be read to and from the files somewhat
> directly without fully reading the memory into contiguous space. Seems to me
> like a ReadWriter style (although marshal/unmarshal bytes is also fine and
> somewhat synonymous for these purposes. To handle a changed flag we'll have to
> make a ReadWriter that accepts the various types and keeps a hash/summary of
> what it reads/writes so we can report on whether or not the file has changed.
> We can use a similar pattern for converting between the types of
> ReadWriter-likes. That's the sort of composable accept interfaces style that go
> typically uses.... For the version where we own the memory choosing to return
> from a different value than the input makes sense since we wouldn't want to
> overwrite memory we're given. They're just different use-cases entirely that's
> why we discussed separate apis.

### The three shapes

> 1) caller owns the file, I just run the editor
> 2) caller owns the memory and provides it, I write its bytes to a file, run the
>    editor, read the file and then truncate the file (which explains why it must
>    fit in memory)
> 3) I own the memory and am provided a way to retrieve bytes. I write them to
>    memory and a file, run the editor, read the file, truncate the file and then
>    return the memory I read from the file.
>
> All of these feel somewhat useful and composed of one another to an extent
> although it may need some massaging. I think we'll likely want to create some
> sort of bytes-like adapter function to turn things into a reader or writerTo as
> an abstraction.

### The subprocess subpackage

> We don't need this level of configurability with the Editor, but I think it
> makes sense to take the same approach as git and split out our helpers that run
> a requested subprocess given a string like we'd extract from the $EDITOR env
> var. Doing that cross platform alone is a useful package worth of
> functionality.

Selected alongside that: `WithStdin`/`WithStdout`/`WithStderr` options, handing
back a pipe end, and "always the real terminal" -- which resolve as *the
subpackage* carrying the stream configurability (prose SS3.1) while the `Editor`
carries none and always gets the real terminal. Same split git has between
`launch_editor` and `run-command`.

## Open questions

### 1. Naming of the three entry points

Rejected so far: `Open`/`Edit`/`EditBytes`, `EditFile`/`Edit`/`EditBytes`,
`Launch`/`Edit`/`EditBytes`, `FromFile`/`FromBuf`/`FromReader`, and the
`Open`/`OpenBuffer`/`OpenReader` family.

> Lets brainstorm some alternatives, not in love with Edit or Open or Launch.
> Open is the closest but it implies opening the file rather than opening the
> editor which is what I think that editor.Open *should* feel like.

> I said different synonyms. not just rehashing the same. We want something else
> like maybe Call or Dispatch or Execute or Spawn or Activate or Start. List at
> least 10 words for this activation or triggering that we're doing before you
> ask again.

**Owed: a list of at least 10 words for the activation/triggering, before asking
again.** Seed list to work from, to be expanded and pruned rather than treated
as an answer:

Call, Dispatch, Execute, Spawn, Activate, Start, Invoke, Summon, Trigger, Raise,
Fire, Hand off, Present, Prompt, Defer to, Yield to, Engage, Enter, Attend,
Consult, Session, Compose.

Constraints to weigh when picking: `Start` and `Run` already carry
`exec.Cmd`'s blocking/non-blocking contract in a Go reader's head (`Start`
returns immediately, `Run` waits), so using `Start` for a blocking call is
actively misleading. `Spawn` and `Fire` say process, not editing. `Invoke`,
`Summon` and `Consult` say "hand control to the editor and wait", which is the
actual semantic.

### 2. Does the target/output belong on the Result?

> I think it could make sense for target/output to always be present on the
> Result even if it isn't usable as a retrieval location. Is there any prior art
> for this in the go std lib or popular usage?

Not yet answered. Candidates to go verify before answering -- none of these have
been checked against the source, they are recall and may be wrong:

- `httptest.ResponseRecorder` -- a result struct that *is* the target
  (`Body *bytes.Buffer` alongside `Code`). Closest match to the idea.
- `os/exec.Cmd` -- destination fields (`Stdout`, `Stderr`) live on the command,
  outcome lands in `ProcessState` after `Run`; `Cmd.Output()` returns the bytes
  instead and requires `Stdout` be nil. Prior art for *both* answers, and for
  the rule that you may not have it both ways at once.
- `http.Response` -- output (`Body`) hangs off the result.
- `sql.Result` -- outcome only, no target. The counter-example.
- `text/template.Template.Execute(w, data)` -- target passed in, nothing
  returned but error.

The question to settle: whether `Result` naming the file it staged (and, for
shape 3, the bytes) makes the type mean one thing or two.

### 3. Shape 3's signature

Depends on (2). Options were `([]byte, Result, error)`, `(Result, error)` with
the bytes on `Result`, or `([]byte, error)` with no `Result`.

## Feedback on the code as written

Verbatim, for the cleanup that still applies whatever the API becomes:

> You're going to have to do a review pass to clean up your AI slop comments.
> This is some pretty bad go code. Look to the existing files or the patch for
> better style

> Yeah, no that's not very go-like for your method of handling the fallbacks.

> hmmm, I think my sense is that [...] Your typical java, python or C++ shapes
> don't always make sense in go.

On the stream fields, correcting the "test seam" reading of them:

> We can keep it testable, but reminder that the stdin/stdout/stderr behaviors of
> the example you were provided had nothing to do with testing and everything to
> do with piping/handling stdin/stdout of the child process. Totally different
> use-case. I think for testing you should ideate a bit more on how you do that
> injection/mocking... I don't think the current shape looks good.

On the receiver question, before it was settled:

> I don't think that looks right either... Maybe more like FromFile or FromBuf? I
> think we may still have tweaking to do on this overall api shape. Lets consider
> the call-sites and then think about which receivers make sense. Should the
> editor be a struct that you open edit or launch? should the reader types be the
> thing that gets a receiver? like buf := editor.Buf(new(bytes.Buffer));
> buf.Open(ctx, launcher) or should it be something else entirely.

On the original behaviour port, for the record:

> The prose come from a very reliable and well understood implementation. We want
> to mirror that but put a golang spin on it. For example, our handling of pipes
> will be very different, our handling of files would obviously use defer instead
> of having multiple clean-up blocks, etc. There should be some fairly obvious
> translation points, but we want the outward functional behaviors to for the end
> user (setting an EDITOR if it contains a shell-like character then it will be
> used with shell otherwise as an executable, etc.) These will draw on very
> well-understood user expectations around these behaviors.

> I think our example prose also has the right idea about splitting the API
> surface into two where the file or reader or buffer ownership is either owned
> by the caller or owned by our library. Having two funcs makes sense in this way.

> yes use stretchr/testify but it's only allowed in tests so it won't be bundled
> by those that import us.

## Review findings still outstanding

From the review pass, independent of the API decisions:

1. `Sources []Source` -- policy as data. Zero value broken, policy can be made
   incoherent, and replacing just the fallback needs
   `e.Sources[len(e.Sources)-1] = ...`, which is what `resolve_test.go` does.
2. `New()` plus a mutable package-level `Default` -- `Default` is a data race by
   construction; a test mutates it.
3. `Open(ctx, name, ext string, v any)` -- 7-arm runtime type switch, two
   returned closures, `errors.Join` of two binding errors, and an
   `acceptedTypes` string constant explaining the type system at runtime.
4. Test seams in the production API: `isTerminal` as a mutable package var,
   `file(v any)`. (The stream fields are *not* in this category -- see above.)
5. Platform seam drawn in the wrong place: `configure()` is an empty function on
   Unix and `command.args` is duplicated across both build tags. Only two things
   are genuinely platform-specific: the shell prefix, and Windows needing
   `SysProcAttr.CmdLine`.
6. `Shell []string` -- "argv prefix ending in the flag that introduces the
   command string" is an under-specified contract for an exported field.

## Where we left off

Next action: produce the list of at least 10 activation verbs (SS Open question 1),
then confirm naming, then answer the `Result` prior-art question before writing
any code.
