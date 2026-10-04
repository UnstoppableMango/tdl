# Specification, grammar, and tree-sitter

## They move together

`docs/spec.md` is canonical and `docs/grammar.ebnf` is the formal grammar.
A grammar or lexer change updates both; flag a change to one that should have changed the other.

## The notation is Wirth, not ISO

The dialect is the Go and Oberon reports', not ISO 14977: productions end with `.`, sequences are juxtaposed, and comments are `/* */` and `//`.
Never suggest an ISO form; it would not parse.

`docs/notation.ebnf` defines the notation, and `TestDocsAreClean` lints both files.
A production with no expression is a lexical name the lexer defines.

## Annotations are machine-readable

A `/*@ ... */` comment is input to `internal/treesitter`, not a note.
A production with no expression needs a `token` binding, and every name an annotation mentions must exist.

## The grammar is held to the lexer

Every quoted terminal must be a spelling `lex.Lookup` knows.
`reserved_word` must match `lex.Keywords` exactly.

## tree-sitter is derived

`tree-sitter/grammar.js` and `tree-sitter/src/` come from `docs/grammar.ebnf` through `make treesitter`.
A hand edit there is in the wrong place.
`tree-sitter/queries/highlights.scm` is hand-written.
