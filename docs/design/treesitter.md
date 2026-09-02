# Deriving the tree-sitter grammar

Design document.
Of what it describes, the lexical tables in `lex/table.go`, the annotations, the generator, and the generated parser are built; the external scanner is a stub and the wiring is not.
[treesitter-plan.md](treesitter-plan.md) tracks which phase has landed.

A tree-sitter grammar gives syntax highlighting, structural selection, and folding to every editor that speaks it, and is the dependency for the editor work in [backlog.md](../backlog.md).
It is also a second parser, and a second parser is a second thing to keep in step with [grammar.ebnf](../grammar.ebnf).

This document says how to have the second parser without maintaining it by hand.

## Why derive it

The obvious way to hold two parsers together is the conformance corpus: a grammar that parses `testdata/conformance/*/source.tdl` and rejects `testdata/invalid/*/source.tdl` agrees with the reference implementation on those files.

That is a real check and it stays.
What it cannot catch is a production that reaches `grammar.ebnf` and never reaches the second parser, because no corpus file exercises it yet.
The corpus is the written-down target rather than a record of what works, so the gap is not hypothetical.

Deriving closes it from the other side.
A rule that exists in one file and not the other is a diff, and a diff fails a build.

## What is already single-sourced

`lex` states the lexical facts for a program in `table.go`.

`Keywords` and `Punctuation` list every fixed spelling, `Lookup` maps one back to the kind the lexer produces, and `Pattern` gives a regular expression for each of the six classes scanned by shape.
The first three read the tables `Lexer.Next` dispatches on, so they cannot drift.
The patterns are declared, because `scanIdent`, `scanNumber`, `scanString`, and `RescanRegexAt` are loops rather than regexes, and a test holds them to the lexer over the corpus.

The grammar's quoted terminals are therefore checkable rather than merely copyable.
Every `"package"` and `"->"` in the EBNF must be a spelling `lex.Lookup` knows, and the `reserved_word` production is checked against `lex.Keywords` in both directions, so neither side can gain a word the other lacks.

## What the EBNF does not say

The grammar is written for a reader, and four things it leaves to prose are things a generator needs.

- The six terminals it names but never defines. `identifier`, `string_lit`, `int_lit`, `float_lit`, `bool_lit`, and `regex_lit` are the lexer's business, which is why they are absent.
- Whitespace and comments. The header says doc comments are lexical and omits them.
- The ambiguities it resolves in prose. Its own comments name them: `key` as a bare requirement against `key` as a field modifier, a `<...>` argument that is a type or a unit, `{ }` as both a declaration body and a set or map type, and `/` as both unit division and a regex delimiter.
- Which productions are structure worth a node and which are plumbing. `CoreType`, `Member`, and `FieldMod` exist to make the notation readable and would only clutter a tree.

Prose is the right form for a reader, and the file explains each of these where it happens.
The annotations below make the same statements machine-readable without moving them somewhere else.

## Annotations

Annotations live in comments, opened `/*@` instead of `/*`.
A reader who ignores them loses nothing, and the file keeps a formatter-free plain-text shape.
`golang.org/x/exp/ebnf` drops comments rather than reporting them, so the annotations are scanned separately and attached by position.

File-level directives come first, before any production.

```ebnf
/*@ word identifier */
/*@ extra doc_comment line_comment */
/*@ token doc_comment = DocPattern */
/*@ conflict FieldMod KeyRequirement */
```

`word` names the token tree-sitter extracts keywords from, which it needs to keep a keyword from matching a prefix of an identifier.
`extra` lists what may appear between any two tokens.
`token` binds a name to the `lex` symbol that defines it, which is what turns a pattern into a tree-sitter rule. It is file-level only for a name with no production, as `doc_comment` and `line_comment` are: the lexer skips ordinary comments and the grammar never mentions them.
`conflict` emits an entry in the grammar's `conflicts` array, one per set of productions the GLR parser cannot decide locally.
A set of one is a production that cannot be decided against its own other readings, which is what a field followed by `where` is.

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

`hidden` emits the rule with a leading underscore, so it structures the grammar without appearing in the tree.
`inline` substitutes the rule into its callers instead.
The generator does the substitution rather than emitting tree-sitter's `inline` array, which refuses a rule that is a single token, as `FieldRel` is.
`prec`, `prec.left`, and `prec.right` carry the associativity the notation cannot state.
`external` marks a terminal the scanner in `scanner.c` produces, which `regex_lit` needs because its `/` is also an operator.
`token` on a lexical production names the `lex` symbol that defines it, so the production's own name does not have to be repeated.

Some things are better spelled out than annotated.
`bool_lit` is `"true" | "false"`, and `reserved_word` lists the keywords one by one, because both are grammar rather than lexical shape.
`internal/ebnf` checks that list against `lex.Keywords`, so it cannot drift.

The set is deliberately small.
An annotation earns its place by describing something `grammar.ebnf` already explains in prose to a reader.

## The generator

A Go program in `tools/treesitter`, in this module, so it imports `lex` directly.
It reads `docs/grammar.ebnf` and writes `tree-sitter/grammar.js`.

Two packages, split at the seam that matters.
`internal/ebnf` reads the notation and the annotations and knows nothing about tree-sitter.
`internal/treesitter` turns what it produces into `grammar.js`.

There is no third parser in this repository.
`golang.org/x/exp/ebnf` documents this dialect exactly, upper-case nonterminals and lower-case lexical names included, and returns a walkable tree with positions.
`internal/ebnf` reads `grammar.ebnf` with it today, adds the checks it does not make, and grows the annotation scanner rather than a parser.

The library is experimental and carries no compatibility promise.
It is 456 lines under a BSD licence and unchanged since 2009, so vendoring it is a real fallback rather than a hypothetical one.

Errors are the point of the exercise, so they are specific.
A quoted terminal the lexer never produces, an undefined nonterminal, a terminal with no `token` binding, and an annotation naming a production that does not exist are each reported with the line that caused them.

## What it does not emit

`queries/highlights.scm` is a judgment about what should be colored, not a fact about the grammar, and is written and maintained by hand.
The same holds for `folds.scm`, `indents.scm`, and injections.

`scanner.c` is hand-written too.
The generator emits the `externals` array naming its tokens and nothing else.

## Where it lives

`tree-sitter/` in this repository, rather than a separate `tree-sitter-tdl`.

One repository is what makes the regeneration check a single CI job, and the grammar has no reason to version independently of the language it describes.
Editors that expect a repository per grammar can be pointed at a subdirectory, and extracting one later is a move, not a rewrite.

## How it is held together

Three checks, each catching what the others cannot.

`command make generate-treesitter` regenerates `grammar.js`, and CI runs it followed by `git diff --exit-code`.
A production added to the EBNF without a regeneration fails the build.
This is the check the corpus cannot make.

The corpus stays.
`tree-sitter parse` over `testdata/conformance/*/source.tdl` must produce no ERROR node, and over `testdata/invalid/*/source.tdl` must produce one.
This catches the generator being wrong, which a clean diff would not.

`lex`'s own tests hold the patterns to the lexer.
This catches the terminals, which neither of the others touches.

## Deferred

- Supertypes. Tagging `Decl` and `TypeRef` as tree-sitter supertypes would give editors a coarser handle on the tree, and it is additive once the grammar exists.
- Comment attachment. `tdl fmt` drops `//` comments and the tree-sitter grammar keeps them as extras, which is a difference the corpus check does not see and does not need to.
- Emitting anything but `grammar.js`. The annotated grammar could drive a railroad diagram or an LSP's semantic token legend, and neither is worth a second output format before the first one works.
