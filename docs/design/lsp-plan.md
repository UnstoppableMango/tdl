# Implementing the language server

An implementation plan for [lsp.md](lsp.md).
Phases are ordered by dependency, and each states what makes it done.
This is not a task list and does not estimate anything.

## Scope

This plan builds `tdl lsp`: the protocol layer, the document store, and the three features [backlog.md](../backlog.md) names as the first ones worth having.

It touches the reference implementation in one place.
`internal/sema` gains an opt-in reference index, because go to definition needs the answer lowering already computes and currently discards.
Nothing else in `lex`, `parser`, `ast`, `ir`, or any backend changes.

## Layout

```text
internal/lsp/          # the server, private
internal/cli/lsp.go    # the `tdl lsp` command
internal/sema/refs.go  # the reference index
```

`internal/lsp` is private for the reason `internal/sema` is: `ir` and `proto` are the compatibility surface, and an editor integration is not.

The server is a subcommand rather than a second binary.
[workflow.md](workflow.md) already writes `tdl lsp` in the inner loop, `nix/cmd.nix` already builds `cmd/tdl`, and an editor that has to find a second executable on `PATH` is a second thing to package and a second thing to misconfigure.

## Testing

The server is driven over `net.Pipe` with `protocol.NewClient` on the far side, so a test speaks real JSON-RPC to the real handler rather than calling the methods directly.
This follows `internal/gen/subprocess_test.go`, where the same backend value is exercised in process and over a connection.

Both corpora are reused rather than restated:

- Every `testdata/conformance/*/source.tdl` publishes no diagnostics, which is what `internal/sema/corpus_test.go` asserts about lowering, asked through the protocol.
- Every `testdata/invalid/*/source.tdl` publishes at least one, containing the text in the sibling `error.golden`.

Adding a conformance case therefore adds a server test, the same way it adds a parser test.

The encoding conversion gets its own table test, over ASCII, a two-byte rune, and a rune outside the basic multilingual plane.
It is the one piece of this that is wrong silently rather than loudly.

## Phase 1: protocol, lifecycle, and diagnostics

The dependency, the base implementation, the position mapping, the document store, the overlay loader, and a server that publishes diagnostics.

Serves `initialize`, `initialized`, `shutdown`, `exit`, and the four text document notifications.
`initialize` advertises full text synchronization and nothing else, so a client asks for nothing this phase does not answer.

`internal/cli/lsp.go` builds the server over stdin and stdout and waits on the connection.
The logger is `zap.NewNop`: anything written to stdout on a stdio server corrupts the stream, and a library that logs by default is a library that will.

Done when opening a file with a syntax error underlines the right span, when opening one with an undefined type underlines that, and when fixing either clears it.

Done.
The two corpora run through the protocol, and `TestDiagnosticsUseUTF16Columns` covers the encoding.

## Phase 2: the reference index and go to definition

`WithReferences` in `internal/sema`, recorded at the places that already resolve a name, and `textDocument/definition` over a binary search by offset.

Type references, class references, and target paths all record.
A target path is the one worth naming: `User.email => tag("json:email")` names a field in a namespace that belongs to the backend, and a path that names nothing is the mistake a model author actually makes.

Done when the cursor on a type in a field jumps to its declaration, in the same file and across an import.

Done.
`TestDefinitionJumpsToADeclaration` and `TestDefinitionCrossesAnImport` are the two jumps, and `TestDefinitionReturnsNothing` covers the three answers that are not a location.

## Phase 3: hover

Rendered from the declaration the reference reached: what it is, what it is called, its doc comment from `Meta.doc`, and its deprecation when it carries one.

A reference into the prelude hovers and does not jump, which [lsp.md](lsp.md) argues is the right answer rather than a limitation.

Done when hovering a field's type shows the declaration and its doc comment.

## Phase 4: formatting and document symbols

`textDocument/formatting` is `ast.Fprint` over the snapshot as one whole-file edit.
This is `tdl fmt` reused rather than reimplemented, and the corpus already holds `ast.Fprint` to canonical output and idempotence, so the feature arrives tested.

`textDocument/documentSymbol` walks `ast.File.Decls` with fields as children.

Both are last because neither needs the index and neither is interesting, not because either is hard.

## What is not here

**Completion.** It needs the scope at a cursor rather than the binding a name reached, which is a different question of lowering.

**Rename and references.** Both want every occurrence of a name, and the index records every resolution, which is not the same set.

**A VS Code client.** Its own work, in `editors/vscode/`, changing nothing here.
Phase 1 is verified in an editor that needs no packaging: Neovim's built-in client, pointed at the built binary.
