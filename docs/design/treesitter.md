# Deriving the tree-sitter grammar

Design document.
All of it is built: the lexical tables in `lex/table.go`, the annotations, the generator, the generated parser, the external scanner, and the checks.

A tree-sitter grammar gives highlighting, structural selection, and folding to editors that speak it, and is what [editors.md](editors.md) builds on.
It is a second parser, so it is derived from [grammar.ebnf](../grammar.ebnf) rather than maintained by hand.

## Why derive it

The conformance corpus checks that two parsers agree on the files it contains.
It cannot catch a production that reaches `grammar.ebnf` and never reaches the second parser, because no corpus file exercises it yet.
Deriving turns that gap into a diff, and a diff fails the build.

## What is already single-sourced

`lex/table.go` states the lexical facts for a program.
`Keywords`, `Punctuation`, and `Lookup` read the tables `Lexer.Next` dispatches on, so they cannot drift.
The `*Pattern` constants are declared, because the scanners are loops rather than regexes, and a test holds them to the lexer over the corpus.

Every quoted terminal in the EBNF must be a spelling `lex.Lookup` knows, and `reserved_word` is checked against `lex.Keywords` in both directions.

## What the EBNF does not say

Four things the grammar leaves to prose, which a generator needs:

- The terminals it names but never defines: `identifier`, `string_lit`, `int_lit`, `float_lit`, and `regex_lit` are the lexer's.
- Whitespace and comments.
- The ambiguities its comments resolve in prose: the name after `type X:` as a class or a newtype's base, a `<...>` argument as a type or a unit, `{ }` as a body or a set or map type, and `/` as unit division or a regex delimiter.
- Which productions deserve a node. `CoreType`, `Member`, and `ClassMember` exist for readability and would clutter a tree.

The annotations make these machine-readable beside the prose that explains them.

## Annotations

Annotations are comments opened `/*@` instead of `/*`, so a reader can ignore them.
`golang.org/x/exp/ebnf` drops comments, so the annotations are scanned separately and attached by position.

File-level directives come first, before any production.

```ebnf
/*@ word identifier */
/*@ extra doc_comment line_comment */
/*@ token doc_comment = DocPattern */
/*@ conflict ClassRef NamedType */
```

- `word` names the token tree-sitter extracts keywords from, which keeps a keyword from matching a prefix of an identifier.
- `extra` lists what may appear between any two tokens.
- `token` binds a name to the `lex` symbol that defines it. At file level it is only for a name with no production, such as `doc_comment` and `line_comment`.
- `conflict` emits an entry in the grammar's `conflicts` array, one per set of productions the GLR parser cannot decide locally. A set of one, such as `Field`, is a production that conflicts with its own other readings.

Production annotations precede the production they describe.

```ebnf
/*@ hidden */
CoreType = ListType | SetOrMapType | NamedType .

/*@ prec.left 1 */
UnitExpr = UnitTerm { ( "*" | "/" ) UnitTerm } .

/*@ token RegexPattern */
/*@ external */
regex_lit = .
```

- `hidden` emits the rule with a leading underscore, so it is absent from the tree.
- `inline` substitutes the rule into its callers. The generator substitutes itself because tree-sitter's `inline` array refuses a rule that is a single token, as `FieldRel` is.
- `prec`, `prec.left`, and `prec.right` carry associativity.
- `external` marks a terminal `scanner.c` produces; `regex_lit` needs it because `/` is also an operator.
- `token` on a lexical production names the `lex` symbol that defines it.

`bool_lit` is spelled `"true" | "false"` and `reserved_word` lists the keywords, because both are grammar rather than lexical shape.
An annotation is added only for something `grammar.ebnf` already explains in prose.

## The generator

`tools/treesitter` reads `docs/grammar.ebnf` and writes `tree-sitter/grammar.js`.
`internal/ebnf` reads the notation and annotations and knows nothing of tree-sitter; `internal/treesitter` writes `grammar.js` from its output.

`golang.org/x/exp/ebnf` documents this dialect exactly and returns a walkable tree with positions.
It is experimental, but it is 456 BSD-licensed lines unchanged since 2009, so vendoring it is a practical fallback.

Each error names the line that caused it: a quoted terminal the lexer never produces, an undefined nonterminal, a terminal with no `token` binding, and an annotation naming a missing production.

## What it does not emit

`queries/highlights.scm` is a judgment about what to color and is hand-written, as are any other query files.
`scanner.c` is hand-written too; the generator emits only the `externals` array naming its tokens.

## Where it lives

`tree-sitter/` in this repository, so the regeneration check is one CI job and the grammar versions with the language.
Zed needs a repository holding only the grammar; [editors.md](editors.md) describes generating a mirror on release.

## How it is held together

- `make check-treesitter`, which CI runs, regenerates and runs `git diff --exit-code` over `tree-sitter/`, so a production added to the EBNF without a regeneration fails.
- `tree-sitter/corpus.sh` requires `testdata/conformance/*/source.tdl` to parse with no ERROR node and `testdata/invalid/*/source.tdl` to produce one, which catches a wrong generator.
- `lex`'s tests hold the patterns to the lexer.

## Deferred

- Supertypes for `Decl` and `TypeRef`, which would give editors a coarser handle on the tree.
- Outputs other than `grammar.js`, such as a railroad diagram or a semantic token legend.
