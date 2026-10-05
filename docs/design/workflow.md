# Consumer workflow

Design document, not user documentation.
It describes the workflow a model author should have once TDL is complete.

The shipped commands are `ast`, `check`, `fmt`, `gen`, `ir`, `lsp`, `play`, `tokens`, and `version` (`internal/cli/root.go`).
Each takes files as arguments.
The project model below (`tdl.toml`, `tdl init`, `[deps]`, `[plugins]`, and a bare `tdl gen` over a whole project) is unbuilt, as is `--format json` for diagnostics.

The consumer is a *model author*: someone who writes `.tdl` files describing their domain and generates code from them.
Backend and prelude authors have other documents.

## The shape of a project

```text
billing/
  tdl.toml
  billing.tdl
  targets.tdl
  gen/
    go/
    sql/
```

A directory is a package, and every `.tdl` file in it carries the same `package` declaration.
A project is one or more package directories plus an optional `tdl.toml` at the root.
Without one, `tdl` treats the current directory as a single package.
A project needs the manifest once it has more than one package, external dependencies, plugins, or a replaced prelude.

## Bootstrap

```shell
tdl init                # writes tdl.toml and a starter .tdl
tdl init target go      # appends a go target block wired to ./gen/go
```

`tdl init target <name>` is scaffolding: it writes a target block with the conventional output path and commented common directives.

## The manifest

```toml
[project]
name = "billing"
sources = ["."]
prelude = "std"

[deps]
acme = { path = "../vendor/acme" }

[plugins]
sql = { command = "tdl-gen-sql" }
```

- `sources` lists package roots and globs.
- `prelude` selects the prelude; omitted, it is the built-in one. `--prelude` overrides it for one invocation (`tdl ir` and `tdl lsp` already take it).
- `[deps]` maps an import prefix to a location.
- `[plugins]` declares external backends.

Manifest keys never describe how types map to a language; that belongs in target blocks.

## Imports

```tdl
import "common.tdl" as common
import "acme/money.tdl" as money
```

An import string resolves as a path relative to the importing file (built).
If its leading segment matches a `[deps]` key, it resolves against that dependency's root instead.

Dependencies are local paths; `tdl` fetches nothing and has no lock file.
Cross-repository sharing is a submodule, a vendor directory, or Nix.

## Generating

```shell
tdl gen                       # every target block in the project
tdl gen --target go           # one target
tdl gen --target go --watch   # regenerate on save
tdl gen --verify              # exit non-zero if output would change
tdl gen --clean               # remove previously generated files first
tdl gen --target go -o ./out  # override the target block's output path
```

Every flag shown is built; today each invocation names its files.

Every target block runs by default: a target block exists, so it generates.
The drift check is `--verify`, so `check` always means "validate the model".

Output paths live in the target block:

```tdl
target go for billing {
  out("./gen/go")
  package("github.com/acme/billing")

  User.email => tag("json:\"email_address\"")
  Money => foreign("github.com/acme/money", "Money")
}
```

`-o` overrides `out` for one invocation.
Nothing else about a target is configurable from the command line, so every mapping is reviewed and versioned in the model.

### File layout

The backend decides.
There is no `layout` directive; a different layout needs a different backend.

### Stale output

`tdl gen` only writes, so a deleted type leaves its file until `tdl gen --clean` removes it.
`--clean` deletes only inside a directory carrying the `.tdl-output` marker the first `tdl gen` into it writes (`internal/gen/marker.go`); cleaning a directory with files and no marker is an error.

## Backends

`go`, `graphql`, `protobuf`, `salesforce`, `smithy`, `thrift`, and `typescript` are built in, along with `debug`.
Any other target name resolves to `tdl-gen-<name>` on `PATH`, with no verification, signing, or version check, the trust model of `git` subcommands and `protoc` plugins.
Declaring a plugin in `[plugins]` documents the dependency and can pin a command; a plugin on `PATH` works without it.

## Inner loop

```shell
tdl check                     # parse and resolve, report errors
tdl gen --target go --watch   # regenerate continuously
tdl lsp                       # language server
```

`tdl check` parses, resolves names and target paths, and reports.
Editors run it on save or connect to `tdl lsp` ([lsp.md](lsp.md)).

`tdl gen --watch` is meant to subsume `tdl play`, with play's views becoming flags on the debug commands.

## Diagnostics

```text
billing.tdl:14:3: unknown type "Momey"
billing.tdl:31:1: target path "User.emial" names nothing
```

`--format json` would emit structured diagnostics for editors and CI annotations.

## Checking generated code into git

Both workflows are supported and neither is the default.

- Committed: CI runs `tdl gen --verify` and fails when a model changed without regenerating. Generated code is reviewed in diffs, and consumers need no `tdl`.
- Build-time: the output directory is gitignored and generation is a build step. Drift is impossible, and every consumer needs `tdl` and its plugins.

## Targets from dependencies

A dependency may ship target blocks, so shared types can say how they appear in Go without every consumer restating it.
Those entries merge with the root project's (partly built: a dependency's block-scope directives reach its `ir.Import`, and its declaration-level directives do not).

Origin outranks specificity: any root entry beats any dependency entry, whatever the spec's field-over-type-over-class ladder says, and the ladder decides among entries of the same origin.
A dependency author cannot rely on a narrow rule surviving, and in exchange a consumer's file always wins.
