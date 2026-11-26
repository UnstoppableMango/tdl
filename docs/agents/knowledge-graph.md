# Knowledge graph

graphify, in the devShell, builds a graph of the code that an agent queries instead of grepping.
`graphify update .` rebuilds it from the syntax tree alone into `graphify-out/`, which is gitignored; it needs no API key.
`graphify query "<question>"`, `graphify path "<A>" "<B>"`, and `graphify explain "<name>"` return a scoped subgraph.
`.graphifyignore` leaves out generated code and fixtures, except the corpora's TDL, which keeps the graph near 6,000 nodes, against the 25,000 the hosted free plan allows.
graphify reads `.tdl` files through an extractor that is not upstream: `pkgs/graphify/tdl.patch` in `unmango/pkgs`, which never pins this repository.
`flake.nix` builds the devShell's graphify with `withGrammars` and the grammar's Python binding, and `checks.graphify` graphs the prelude and two conformance cases and asserts a declaration implements `Entity`.
If the patch fails to apply to two graphify releases in a row, `unmango/pkgs` drops it, and with it TDL in the graph (see `docs/architecture/graphify.md`).
Semantic extraction, `graphify extract` without `--code-only` or `/graphify .` in an assistant, reads the docs through a model and spends tokens; the code graph does not.
`.mcp.json` serves the graph to Claude Code over stdio with `graphify-mcp`, so it needs the devShell on `PATH` and a built graph.
The devShell's graphify comes from `unmango/pkgs`, which runs it from a Python environment, so the interpreter graphify re-executes itself through can import it; nixpkgs' package cannot.
`graphify claude install` and `graphify hook install` write `.claude/settings.json` and `.git/hooks`, which stay local.
