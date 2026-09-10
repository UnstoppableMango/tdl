# Specification, grammar, and tree-sitter

## They move together

`docs/spec.md` is canonical and `docs/grammar.ebnf` holds the formal grammar.
Both must be updated alongside any grammar or lexer change.
A change to one that should have changed the other is the thing to flag.

## The notation is Wirth, not ISO

The dialect is the one the Go and Oberon reports use, not ISO 14977.
A production ends with `.` rather than `;`, and items in a sequence are juxtaposed rather than separated by `,`.
Comments are Go's `/* */` and `//`, not the ISO family's `(* *)`.
Never suggest an ISO form; it would not lint and could not be parsed.

`docs/notation.ebnf` describes that notation in itself, and `internal/ebnf` lints both files.
`TestDocsAreClean` fails the build when either stops linting clean.

A lexical name the lexer owns is declared as a production with no expression, which is how this notation says the name is defined elsewhere.

## Annotations are machine-readable

A `/*@ ... */` comment is not a note.
`internal/ebnf` reads them and `internal/treesitter` turns them into `tree-sitter/grammar.js`.

A production with no expression needs a `token` binding, every name an annotation mentions has to exist, and a production added or renamed without regenerating fails CI.

## The grammar is held to the lexer

A quoted terminal must be a spelling `lex.Lookup` knows, so a spelling the grammar invents is a rule that can never match.
`reserved_word` is checked against `lex.Keywords` in both directions, so a keyword added to one and not the other fails the build.

## tree-sitter is derived

`tree-sitter/grammar.js` and `tree-sitter/src/` come from `docs/grammar.ebnf`, and `make treesitter` regenerates both.
A change to the derived files that is not a consequence of a change to the grammar or the emitter is the wrong place to make it.

`tree-sitter/queries/highlights.scm` is hand-written, because what should be colored is a judgment rather than a fact about the grammar.
