# Implementing the language server

An implementation plan for [lsp.md](lsp.md), continued by [lsp-editors-plan.md](lsp-editors-plan.md).
Phases are ordered by dependency, and each states what makes it done.
All four are done.

## Scope

`tdl lsp`: the protocol layer, the document store, diagnostics, go to definition, hover, formatting, and document symbols.
The one change outside `internal/lsp` is an opt-in reference index in `internal/sema`.

## Layout

```text
internal/lsp/          # the server, private
internal/cli/lsp.go    # the `tdl lsp` command
internal/sema/refs.go  # the reference index
```

The server is a subcommand of `tdl` rather than a second binary, so an editor has one executable to find.

## Testing

Tests drive the server over `net.Pipe` with `protocol.NewClient` on the far side, so they speak real JSON-RPC to the real handler.

Both corpora run through the protocol:

- Every `testdata/conformance/*/source.tdl` publishes no diagnostics.
- Every `testdata/invalid/*/source.tdl` publishes at least one, containing the text in the sibling `error.golden`.

The encoding conversion has a table test over ASCII, a two-byte rune, and a rune outside the basic multilingual plane.

## Phase 1: protocol, lifecycle, and diagnostics (done)

Serves `initialize`, `initialized`, `shutdown`, `exit`, and the four text document notifications, and publishes diagnostics.
`internal/cli/lsp.go` runs the server over stdin and stdout and logs nothing, since anything written to stdout corrupts the stream.

Done when opening a file with a syntax error or an undefined type underlines the right span, and fixing it clears the underline.
`TestDiagnosticsUseUTF16Columns` covers the encoding.

## Phase 2: the reference index and go to definition (done)

`WithReferences` in `internal/sema`, recorded where a name is already resolved, and `textDocument/definition` by binary search over offsets.
Type references, class references, and target paths all record.

Done when the cursor on a field's type jumps to its declaration, in the same file and across an import.
`TestDefinitionJumpsToADeclaration`, `TestDefinitionCrossesAnImport`, and `TestDefinitionReturnsNothing` cover it.

## Phase 3: hover (done)

Renders the declaration a reference reached, its doc comment, and its deprecation.
A reference into the prelude hovers and does not jump.

Done when hovering a field's type shows the declaration and its doc comment.

## Phase 4: formatting and document symbols (done)

`textDocument/formatting` is `ast.Fprint` over the snapshot as one whole-file edit, so it inherits the corpus checks on canonical output and idempotence.
`textDocument/documentSymbol` walks `ast.File.Decls` with fields as children.

Done when formatting rewrites a messy file and the outline lists declarations and their fields.

## Not here

- **Completion, rename, and references.** [lsp-editors-plan.md](lsp-editors-plan.md) phases 7 and 8.
- **Editor clients.** [lsp-editors-plan.md](lsp-editors-plan.md).
