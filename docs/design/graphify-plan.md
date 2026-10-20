# Implementing TDL in graphify

An implementation plan for [graphify.md](graphify.md).
Phases are ordered by dependency, and each states what makes it done.

Phases 1 and 2 are done.
Phase 3 is built, but `graphify update .` gives 5,819 nodes, above its 4,000 target; the graph was already near 6,500 before TDL joined it.

## Scope

This plan builds the Python binding, the graphify patch in unmango/pkgs, and the devShell wiring.
It publishes nothing outside the two repositories.

## Phase 1: the binding

- `tree-sitter/tree-sitter.json` enables the Python binding, and `tree-sitter init` writes `bindings/python/`, `pyproject.toml`, and `setup.py`, with `scanner.c` in the sources.
- `nix/` builds it as `tree-sitter-tdl` with `buildPythonPackage`, exposed as `packages.tree-sitter-tdl`, and a check imports it and parses `prelude/std.tdl` with no ERROR node.
- The generated files join the formatter exclusions and `.graphifyignore` as other generated code does, and `make check-treesitter` still passes.

Done when `nix build .#tree-sitter-tdl` and `nix flake check` pass.

## Phase 2: the patch

In its own pull request in unmango/pkgs:

- A patch on graphify adds `graphify/extractors/tdl.py` and registers `.tdl`, producing the nodes, edges, and confidences [graphify.md](graphify.md) lists.
- It adds a TDL fixture and cases in `tests/test_languages.py`; `checkPhase` runs them with phase 1's package, taken from a local path, supplied only to the tests.
- The graphify package gains `passthru.withGrammars`.

Done when the package builds with its tests passing, and `graphify.withGrammars [ tree-sitter-tdl ]` extracts the fixture's nodes and edges.

## Phase 3: the devShell

- `make update` picks up phase 2, and the devShell uses `inputs'.mangopkgs.packages.graphify.withGrammars [ tree-sitter-tdl ]`.
- `.graphifyignore` stops ignoring `testdata/**/*.tdl`.
- `checks.graphify` runs `graphify update` over the prelude and two conformance cases and asserts the `Entity` node and an `implements` edge into it.
- `AGENTS.md`'s knowledge graph section says graphify reads TDL, where the extractor lives, and the back-out rule.

Done when `nix flake check` passes and `graphify update .` in the devShell stays below 4,000 nodes with the TDL declarations in it; the pull request reports the node count before and after.
