# The language server

Design document.

`tdl lsp` is the editor-facing half of the inner loop [workflow.md](workflow.md) describes.
It answers three questions about a file being edited: what is wrong with it, where a name is declared, and what a name means.

[editors.md](editors.md) builds highlighting and says every target there wants this more than it wants colors.
Nothing in that document changes here, and nothing here reads the tree-sitter grammar: a highlighter and a language server answer different questions from the same source text, and the reference implementation is what answers the second.

[lsp-plan.md](lsp-plan.md) tracks which phase has landed.

## What it is built on

The compiler side needed nothing added to it.

`parser.Parse` already reports every syntax error in one pass as a `parser.ErrorList` rather than stopping at the first, which is the same thing an editor wants: a file with three mistakes should underline three, not one.
`sema.Lower` already reports every problem it finds as `sema.Diagnostics`, each an `ast.Position` and a message.
`sema.Loader` is already an interface, so imported sources can come from anywhere.

That last one is what makes an editor's unsaved buffers work.
An editor's copy of a file is not the file on disk, and a server that read the disk would resolve an import against text the user has already changed.
`Loader` was introduced so lowering touches no filesystem, and the overlay below is the payoff.

So the server is a protocol layer and an index, not a second front end.

## The protocol layer

`go.lsp.dev/protocol` at v1, with `go.lsp.dev/jsonrpc2`.

The alternative was hand-rolling JSON-RPC and the message types, which the `plugin` package does for its own wire format.
That reasoning does not carry here.
`plugin` owns its protocol and can define it in one proto file; LSP is someone else's protocol with a large surface, and a hand-written subset is a subset that grows every time a feature is added.

The dependency is not free, and two of its properties are worth stating so they are not rediscovered.

**`protocol.Server` is the whole LSP surface and nothing implements most of it.**
`protocol.UnimplementedServer` is the base the package ships for that: every method answers "method not found", except a notification, which is ignored, because a non-nil error from a notification handler tears the connection down.
`Server` embeds it and overrides what it serves, so adding a feature is one method rather than an edit to a dispatch table.

**The server does not negotiate `positionEncoding`, so positions on the wire are UTF-16 code units.**
LSP 3.17 added the negotiation and the capability defaults to `utf-16` when a server offers nothing, which is what this one does.
`lex.Position` counts bytes, so the conversion is required rather than a nicety, and a server that skipped it would work on ASCII and put the underline in the wrong place the first time a model used a non-ASCII character in a comment or a string.

`internal/lsp/position.go` is the one place that knows this.
It builds a line index over a document's text and converts in both directions, and every other file in the package works in `lex.Position` or in byte offsets.

## Text synchronization

Full, not incremental.

A `.tdl` file is a description of a domain model rather than a program, the parser is one pass over it, and lowering is a handful of passes over the tree.
Incremental sync would save work that is not worth measuring and would add a second way for the server's copy of the text to drift from the editor's.

If a model ever appears that is slow to reparse on every keystroke, the answer is to debounce, and only after that to sync incrementally.

## The overlay

`internal/lsp/loader.go` implements `sema.Loader` over the document store, falling back to `sema.FSLoader` for a file nobody has open.

This is the whole reason lowering takes a loader.
An open document is served from the editor's text at the editor's version, and a file on disk is read from disk, and lowering cannot tell the difference.

## Diagnostics

A snapshot is computed per document: the text, the `*ast.File`, the parse errors, the `*ir.Model`, and the lowering diagnostics.

Parse errors and lowering diagnostics are not both published.
A file that does not parse produces an `*ast.File` with holes in it, and lowering that tree reports names it cannot resolve because the declaration they name failed to parse.
Those are noise, and `sema.Diagnostics` already carries the same rule internally: a non-empty list means no later pass should run.
So a file with syntax errors publishes syntax errors only, and lowering diagnostics appear once it parses.

Diagnostics are grouped by the filename in each position rather than published against the request's URI.
An import that fails to resolve is a problem in the file that wrote the import, and lowering already records that; a problem lowering found in an imported file belongs on that file.

A position is a point and a diagnostic wants a range, so the range covers the word at that position when there is one, and is empty when there is not.
An empty range is rendered by every editor as a caret at the position, which is the honest answer when the compiler only knows where something started.

## Go to definition

Lowering resolves every name and then throws the answer away.
`sema` interns a type reference into the type table, and the table is keyed by structure, so the second mention of a name is not a second entry, and a cursor on it has nothing to find.

`internal/sema/refs.go` records the answers instead, behind `WithReferences`.
Each record is the position and length of the name as written and what it bound to: a declaration, a type parameter, or an extern.
Recording is off by default, so `tdl check` and `tdl gen` pay nothing.

The hooks are the places that already resolve a name, so the server and `tdl check` cannot disagree about what a name means.
There is no second resolver, and shadowing, the prelude, and imports are not restated anywhere in `internal/lsp`.

A `_` import binds each of the dependency's exported names at the position the dependency declares it, so a reference to one jumps into that file.
That is the one place a binding's position is in a file other than the one being edited, and it is why the server reads a file it has no document for: a jump has to land on the name, and a column is a byte offset into text the server would otherwise not have.
An aliased import is the other answer: the dependency is parsed for the names it exports rather than for the declaration a qualified reference names, so `common.Money` records the question and no target.
The record still earns its place, because it is what stops a cursor on a qualified name from finding whatever reference sits next to it.

## The prelude has no file

The prelude is embedded with `go:embed` and lowered under `prelude.Name`, which names no path an editor can open.

A reference into the prelude therefore has no definition to jump to, and the server returns nothing rather than a location that fails to open.
Hover is the answer instead, and it is usually the one that was wanted: `[T]` resolving to `List` is a fact about the prelude, and what a user wants is to be told, not to be moved into a file they cannot edit.

Serving it under a `tdl:` URI would need a content provider in every client, which buys one jump per prelude name.
It is not worth a client-side feature in four editors.

## Not in this document

- **Completion.** It needs the bindings in scope at a cursor, and what the index records is the binding a resolved name reached. That is a different question, and it is worth asking after the first three features have been used.
- **Rename and references.** Both fall out of the index once it records every occurrence rather than every resolution, which is a smaller step than it sounds and still a separate one.
- **Editor clients.** A client changes nothing about the server. [lsp-editors-plan.md](lsp-editors-plan.md) plans one for each editor, and the features after them.
- **The MCP server.** [backlog.md](../backlog.md) has it, and it reads a resolved model rather than a file being edited. The two share `sema` and nothing else.
