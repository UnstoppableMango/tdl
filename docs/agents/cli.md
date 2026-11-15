# CLI and language server

## CLI

The root command silences cobra's error printing; a command returns its error and `cmd/tdl` prints it.

Shell completion is cobra's `completion` command; `complete.go` holds the value completion every command registers, served through `__complete`, and never starts a plugin.

`file.go` is shared by every command that reads files: `loadFile` reads and parses one, and `eachFile` walks the arguments, reporting each failure and continuing.
It prints the error alone, since diagnostics and `os.PathError` already name the file.
With more than one file, output is separated by a `==> path <==` banner; `gen` prints only the paths it wrote.
`play` takes one file, and `gen --watch` rejects more than one.
`fmt --check` lists non-canonical files and writes nothing; `fmt -w` keeps each file's mode.

A file named `-` is standard input, shown as `<stdin>` in positions.
`fmt -w` rejects it, and so does `gen`, because imports resolve relative to the importing file and stdin has no directory.
`ir` and `import` accept it.

`import --from <target>` asks a backend for a model, prints it through `internal/unlower`, and lowers the printed source again, writing nothing when that fails.
`gen --make <target>` writes a make dependency file once the whole run succeeds, naming every file it read through a `sema.Recording` loader, and leaves an existing one alone when the run fails.
`gen` and `import` both drop warnings whose loss code `tdl.toml` or `--allow-lossy` allows, and print a code after the message.
`import` reports read-back loss, what the TDL keeps but regenerating writes differently, only under `--readback`, which sets `ImportRequest.readback`; an importer writes it with `reverse.Reader.Respell` and everything else with `Warn`.

`request --target <name>` writes, as framed messages, a `Handshake` and one `RequestSet`: the requests `gen` would send that backend, and every diagnostic, a file with an error yielding no request.
It runs no backend and writes nothing to disk, for a host that runs the backend itself, such as the C# source generator.
With `--overlay`, standard input holds one framed `Overlay` whose texts `sema.Overlay` reads in place of the disk; the language server's loader is the same type.

`examples/` holds files to experiment with and is outside the conformance corpus.

## Language server

`internal/lsp` serves `tdl lsp` over `go.lsp.dev/protocol`.
An overlay `sema.Loader` serves unsaved text of open documents so imports resolve against the editor's state.

- `position.go` converts between `lex.Position` byte columns and the protocol's UTF-16 code units.
- Go to definition reads the `sema.WithReferences` index.
- Hover prints the declaration with `ast.PrintDecl`, then its deprecation and doc comment; prelude names hover from the embedded source.
  On a declaration, field, or variant where it is written, it also lists the directives each target block resolves onto it, read from the lowered model.
- Formatting returns `ast.Fprint` as one edit and refuses a file that does not parse.
- The outline reads the tree, so a broken file still outlines what was recovered.

See `docs/architecture/lsp.md`.
