# TDL in graphify

Design document.
Nothing here is built; [graphify-plan.md](graphify-plan.md) orders the work.

graphify builds the code graph an agent queries instead of grepping (see `AGENTS.md`).
It parses each language with a tree-sitter grammar and a Python extractor, and it knows no TDL: a `.tdl` file lands in its unclassified list and is never read.
The `.tdl` files are the specification's corpus, the prelude, the examples, and the smoke models, and none of them is in the graph.

## Goal

An agent asking graphify about this repository sees the TDL declarations as nodes, linked to one another the way lowering links them.
Other repositories holding `.tdl` files get the same graph once graphify reads the language, but this repository is the driver.

## Graph

A `.tdl` file contributes three kinds of node:

- the file;
- its package, from the `package` declaration;
- one node per named declaration: type, enum, alias, class, mixin, primitive, and unit.

Fields and enum variants are not nodes.
They would multiply the graph past the 4,000 nodes `.graphifyignore` keeps it below, and each is reachable from its declaration's source location.

Edges use graphify's existing relations, so its queries and its ranking of hub nodes treat TDL as any other language:

| TDL | Relation |
| --- | --- |
| `import "x.tdl" as x` | `imports`, file to file |
| a package holding a declaration | `contains` |
| a type reference to a declaration | `references` |
| `include` of a mixin | `mixes_in` |
| conformance to a class, and `instance` | `implements` |
| a `requires` constraint | `references` |

Target blocks and directives add no edges.
They configure backends, and an edge per directive would tie every declaration to every target.
Nothing links a declaration to the Go, protobuf, or other code generated from it; that needs the compiler, not a parser.

### Names

graphify reads one file at a time and does not lower, so the extractor resolves names the way `internal/sema` scopes them, closely enough without reimplementing it:

1. A bare name resolves to a declaration in the same file, then to one in `prelude/prelude.tdl`.
1. A qualified `x.Name` resolves through the import aliased `x`.
1. Both are marked `EXTRACTED`.
1. A name neither places, such as one brought in by a `_` import, is matched by name across the graph and marked `INFERRED`; one matching more than one declaration is left out.

A prelude name links to the prelude's own declaration, which is in the repository, so `Entity` becomes the hub it is.

## Extractor

The extractor is a bespoke `graphify/extractors/tdl.py`, not a `LanguageConfig` for graphify's generic extractor, which is built around the classes, functions, and calls TDL does not have.
`.tdl` is registered wherever graphify lists extensions.
It imports `tree_sitter_tdl` only when that is installed, the way graphify's optional extras import their grammars, so graphify builds and runs without it.

## Packaging

graphify has no plugin mechanism, so a new language is a change to graphify itself.
Nothing goes upstream for now: no graphify issue or pull request, and no `tree-sitter-tdl` on PyPI.

- **The grammar.** `tree-sitter/` gains tree-sitter's generated Python binding, compiling the hand-written `scanner.c` with the parser.
  tdl's flake builds it as a `tree-sitter-tdl` Python package from the same commit as the `.tdl` files it reads.
- **The extractor.** [unmango/pkgs](https://github.com/unmango/pkgs) carries it as a patch on its graphify derivation, with a fixture and cases in graphify's `tests/test_languages.py`, run in `checkPhase` with the grammar supplied only to the tests.
  The patch is written as the upstream pull request would be, so it can be sent unchanged later.
- **Joining them.** The graphify package exposes `passthru.withGrammars`, which rebuilds its Python environment with extra grammar packages.
  tdl's devShell uses `graphify.withGrammars [ tree-sitter-tdl ]`.

Dependencies run one way, tdl to `mangopkgs`: unmango/pkgs never pins tdl.
`withGrammars` is an interface the package publishes, not an override in tdl's flake.
Splitting the grammar into its own repository would cut it off from `docs/grammar.ebnf` and the corpus checks, and fetching tdl's source from unmango/pkgs would keep the two repositories pinning each other.

### Backing out

The patch is an experiment.
It is removed, with `withGrammars` and the binding, if it fails to apply to two graphify releases in a row without editing graphify's core files by hand.
graphify is moving extractors out of `extract.py`, so the first bump is the real test.

## In this repository

- `.graphifyignore` stops ignoring `testdata/**/*.tdl`, since 64 of the 70 `.tdl` files are conformance and invalid cases; the goldens and other fixtures stay out.
- `checks.graphify` runs `graphify update` over the prelude and two conformance cases, and asserts the graph holds the `Entity` node and an `implements` edge into it.
  It is the one test of the binding and the patch together, and needs no model or network.
- `AGENTS.md`'s knowledge graph section says graphify reads TDL and where the extractor lives.

## Not covered

- **Upstream.** A graphify pull request, and `tree-sitter-tdl` on PyPI, which it would depend on.
- **Generated code.** Edges from a declaration to the code a backend writes for it.
- **Fields and variants** as nodes.
