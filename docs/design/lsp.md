# The language server

Design document.
[lsp-plan.md](lsp-plan.md) and [lsp-editors-plan.md](lsp-editors-plan.md) track what has landed.

`tdl lsp` is the editor-facing half of the inner loop in [workflow.md](workflow.md).
It answers three questions about a file being edited: what is wrong with it, where a name is declared, and what a name means.
It does not read the tree-sitter grammar; highlighting is [editors.md](editors.md).

## What it is built on

The server is a protocol layer and an index over the existing front end.

- `parser.Parse` reports every syntax error in one pass as a `parser.ErrorList`.
- `sema.Lower` reports every problem as `sema.Diagnostics`, each an `ast.Position` and a message.
- `sema.Loader` is an interface, so imported sources can come from an editor's unsaved buffers.

## The protocol layer

`go.lsp.dev/protocol` at v1, with `go.lsp.dev/jsonrpc2`.
LSP is a large protocol owned by someone else, so a hand-written subset would grow with every feature.

`protocol.Server` is the whole LSP surface.
`Server` embeds `protocol.UnimplementedServer`, which answers "method not found" for a request and ignores a notification (a non-nil error from a notification handler tears the connection down), so adding a feature is one method.

The server does not negotiate `positionEncoding`, so positions on the wire are UTF-16 code units, the LSP 3.17 default.
`lex.Position` counts bytes, so skipping the conversion would misplace every underline after a non-ASCII character.
`internal/lsp/position.go` is the one place that converts; every other file works in `lex.Position` or byte offsets.

## Text synchronization

Full, not incremental.
Parsing and lowering a model is cheap, and incremental sync adds a way for the server's copy of the text to drift from the editor's.
If reparsing on every keystroke becomes slow, debounce first.

## The overlay

`internal/lsp/loader.go` implements `sema.Loader` over the document store, falling back to `sema.FSLoader` for a file nobody has open.
An open document is served from the editor's text, so an import resolves against what the editor has rather than what is on disk.

## Diagnostics

A snapshot per document holds the text, the `*ast.File`, the parse errors, the `*ir.Model`, and the lowering diagnostics.

A file that does not parse publishes syntax errors only.
Lowering a tree with holes reports names as undefined only because their declaration failed to parse.

Diagnostics are grouped by the filename in each position rather than published against the request's URI, so a problem lowering found in an imported file lands on that file.

A diagnostic's range covers the word at its position, or is empty when there is none, which editors render as a caret.

## Go to definition

Lowering interns a type reference by structure, so a second mention of a name is not a second entry and a cursor on it would have nothing to find.
`internal/sema/refs.go` records each resolution behind `WithReferences`: the position and length of the name as written, and what it bound to (a declaration, a type parameter, or an extern).
Recording is off by default, so `tdl check` and `tdl gen` pay nothing.

The hooks are the places that already resolve a name, so the server and `tdl check` cannot disagree about what a name means.

A `_` import binds each exported name at the position the dependency declares it, so a reference jumps into that file.
This is the one case where the server reads a file it has no document for, because a column is a byte offset into that file's text.
A qualified reference such as `common.Money` records the question and no target, which keeps a cursor on it from finding a neighboring reference.

## The prelude has no file

The prelude is embedded and lowered under `prelude.Name`, which names no path an editor can open.
A reference into it has no definition to jump to, and the server returns nothing.
Hover answers instead: `[T]` resolving to `List` is a fact a user wants to be told.
Serving the prelude under a `tdl:` URI would need a content provider in every client.

## Not in this document

- **Completion, rename, and references.** Completion needs the bindings in scope at a cursor; rename and references need every occurrence rather than every resolution. [lsp-editors-plan.md](lsp-editors-plan.md) plans both.
- **Editor clients.** [lsp-editors-plan.md](lsp-editors-plan.md).
- **The MCP server.** [backlog.md](../backlog.md). It reads a resolved model rather than a file being edited.
