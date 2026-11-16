# Architecture

TDL is a language for describing domain models: entities, values, enums, newtypes, classes, and collections.
It has no expressions, control flow, or runtime.
This repository holds the specification and the reference implementation.

The parser reads the whole grammar, and the conformance corpus lowers to `ir` with no diagnostic.
A dependency's block-scope target directives reach its `ir.Import`, and its declaration-level directives reach the `ir.Extern` naming the declaration; a transitive dependency's do not, and two dependencies are never compared.
The plugin protocol in `docs/architecture/plugins.md` is complete.

## Front end

- `lex`: hand-written lexer.
  Positions start here and flow through the AST as `ast.Position`.
  Regex literals are scanned only on request via `RescanRegexAt`, because `/` is also unit division.
  `table.go` exports `Keywords`, `Punctuation`, `Lookup`, and the `*Pattern` constants so tools derived from the grammar read what the lexer accepts.
- `parser`: recursive descent producing `*ast.File`.
  Errors accumulate in an `ErrorList`; `syncTop` resynchronizes at the next declaration.
- `ast`: parse tree mirroring the source, names unresolved.
  `ast.Fprint` is the canonical formatting used by `tdl fmt`.
  Every node is an `ast.Node`: `Pos` is where its first token starts and `End` is just past its last, from the lexer's `Token.End`; a block's closing brace is `Rbrace`.
  A doc comment and a `deprecated` modifier come before `Pos`.
  `ast.Inspect` walks a tree in source order as `go/ast.Inspect` does, and `TestChildrenCoverEveryField` keeps `ast.Children` in step with the node types.
- `internal/ebnf`: linter for `docs/grammar.ebnf` and `docs/notation.ebnf`, built on `golang.org/x/exp/ebnf`.
  It also checks every quoted terminal against `lex`, and reports unterminated comments and strings itself because the library's scanner prints them to stderr.
- `internal/treesitter`: emits `tree-sitter/grammar.js` from the grammar.
  Rule names are snake_case, a hidden production gets a leading underscore, an inline one is substituted into its callers, and rules keep source order.
  `tools/treesitter` runs it.
  `tree-sitter/src/` is committed output of `tree-sitter generate`, except `scanner.c`, which is hand-written and produces `regex_lit` from `valid_symbols` the way the parser calls `lex.RescanRegexAt`.
- `internal/textmate`: emits `editors/vscode/syntaxes/tdl.tmLanguage.json` from the grammar, since VS Code highlighting is TextMate, and the same rules as `editors/bat/tdl.sublime-syntax` for bat, which reads only sublime-syntax.
  See [TextMate grammar](grammar.md#textmate-grammar).
  `tools/textmate` runs it.

## Middle

- `internal/sema`: ast to ir.
  Declaration table, interned type and unit tables, sugar lowering, scopes, recursion rules, and the import graph.
  `units.go` runs first, because a unit may be written after one deriving from it and a type argument naming a unit needs its reduction.
  It touches no filesystem: a `Loader` supplies imports, `FSLoader` for real files and `MapLoader` in tests.
  `refs.go` records every name resolution when `WithReferences` is set, for the language server, by hooking the places lowering already resolves names.
  Private.
- `internal/unlower`: ir to ast, the reverse of `sema`, for reverse backends.
  It writes the model's own declarations and leaves to lowering what lowering computes: inherited constraints, mixin fields, struct kinds, and a class directive's expansion.
  `TestCorpusRoundTrips` holds it to lowering the printed corpus back to an equal model.
- `ir`: the resolved model backends consume.
  `ir.pb.go` is generated from `proto/tdl/ir/v1/ir.proto`; `model.go` holds hand-written lookups, and `WithoutPositions` clears positions so `proto.Equal` compares two models.
  Three interned tables, each its own ID space: `Decls`, `Types`, `Units`.
  A unit is interned on its base dimensions, so `decimal<N>` and `decimal<kg*m/s^2>` are one entry in `Types`; `UnitDef` is the declaration and `Unit` what it measures.
  `proto/` and `ir/` are the public compatibility surface.
- `prelude`: the standard prelude, in TDL, embedded with `go:embed`.
  `sema` loads it beneath every file and merges its declarations untagged.
  Lowering knows the sugar's spellings (`List`, `Option`, ...) but not their meaning, so the prelude is replaceable.
- `plugin`: the backend wire protocol, generated from `proto/tdl/plugin/v1/plugin.proto`, plus the framing codec.
  Public.
- `internal/analysis`: what every backend serving one target would otherwise compute, sent in `plugin.Request.Analysis`: each type table entry resolved to the prelude's shapes with aliases expanded, or refused with a reason; and per declaration, whether it is the prelude's, what it names, its enum discriminant, its newtype chain's end, and the words of its names.
  `emit` reads it, and derives it when a request carries none.
  `Scope` cuts a model down to one target's block and directives, which is all a request carries.
  It also reads each constraint (`constraints.go`) and, under a backend's declared `Numbering`, numbers every field and variant (`numbers.go`); `emit.Length`, `Pattern`, `ConstraintText`, and `Session.Numbers` share that code, and `Session.NumbersIn` reads the numbers a request carries.
  `Request` builds every request a host sends, returning a length that can never hold or a refused pin as errors to report before the backend runs.
  Under a `roundtrip` directive it also carries the model as TDL (`source.go`), which `emit`'s round-trip helpers read; `Imported` lowers an import response that returns TDL source instead of a model.
- `internal/gen`: the compiler side of the plugin protocol: the backend registry, request building, and writing returned files.
  `keep.go` keeps a nondeterministic backend's output while its request hashes the same, recording the hashes in `.tdl-output` (ADR-0002).
  `post.go` runs the commands a target block's `post` directive names, through `tdl.toml`'s `[post]` table, over each returned file before anything is compared or written.
  `internal/gen/echo` is a test backend that writes a model as JSON and imports it back; it is not compiled into `tdl`.
- `internal/host`: what a host does around a backend, importing none, so a backend hosting itself can use it.
  `loss.go` declares the loss codes, which `emit` re-exports, so `tdl`'s completion lists them too.
  `check.go` holds every directive to its backend's `DirectiveSpec` (arity, kinds, placement, allowed values, repeats) before the backend runs, so a backend need not check them itself; `Directives` holds the specs of those `tdl` reads itself for every target, `post` so far, and `out` is never checked.
  `Silence` drops warnings with an allowed loss code, and `Allowed` reads which codes `tdl.toml` allows.
- `internal/config`: reads the `[lossy]` and `[post]` tables of the nearest `tdl.toml` above a file.
- `internal/makefn`: what the make functions in `cmd/tdl-gmk` return, `Files` and `Imports`, kept out of the plugin so they test without make.
  `cmd/tdl-gmk` is behind the `makeplugin` build tag, since a `main` package built with `-buildmode=c-shared` links against symbols only make defines; `go build ./...` and the linters never see it.
- `internal/cli`: cobra commands (`ast`, `check`, `fmt`, `gen`, `import`, `ir`, `lsp`, `play`, `request`, `tokens`, `version`). See [CLI](cli.md#cli).
- `internal/lsp`: the language server. See [Language server](cli.md#language-server).
- `cmd/tdl`: main.
