# Consumer workflow

Design document, not user documentation.
It describes the workflow a model author should have once TDL is complete.
Almost none of it is implemented; the shipped commands are `check`, `fmt`, `ast`, `tokens`, `play`, and `version` over the M1 grammar subset.

The consumer here is a *model author*: someone who writes `.tdl` files describing their own domain and runs the compiler to get Go, TypeScript, or SQL out.
Backend authors and prelude authors are different audiences with different documents.

## The shape of a project

```
billing/
  tdl.toml
  billing.tdl
  targets.tdl
  gen/
    go/
    sql/
```

A directory is a package.
Every `.tdl` file in it carries the same `package` declaration.
A project is one or more package directories plus a `tdl.toml` at the root.

`tdl.toml` is optional.
Without it, `tdl` treats the current directory as a single package and discovers targets from the `.tdl` files it finds.
A project needs the manifest once it has more than one package, external dependencies, plugins, or a replaced prelude.

## Bootstrap

```shell
tdl init                # writes tdl.toml and a starter .tdl
tdl init target go      # appends a go target block wired to ./gen/go
tdl init target sql
```

`tdl init target <name>` is scaffolding, not registration.
It writes a target block with the conventional output path and the directives most projects override, commented.

## The manifest

```toml
[project]
name = "billing"
sources = ["."]
prelude = "std"

[gen]
default = ["go", "sql"]

[deps]
acme = { path = "../vendor/acme" }

[plugins]
sql = { command = "tdl-gen-sql" }
```

`sources` lists package roots and globs.
`default` under `[gen]` names the targets `tdl gen` runs with no `--target`.
`prelude` selects the prelude; omitting it uses the built-in one.
`[deps]` maps an import prefix to a location.
`[plugins]` declares external backends.

Manifest keys never describe *how* types map to a language.
That belongs in target blocks, always.

## Imports

```tdl
import "common.tdl" as common
import "acme/money.tdl" as money
```

An import string resolves as a filesystem path relative to the importing file.
If the leading path segment matches a key in `[deps]`, it resolves against that dependency's root instead.

Dependencies are local paths.
The manifest syntax leaves room for a source URL and a pinned revision, but `tdl` does not fetch anything and has no lock file.
Cross-repo sharing today is a submodule, a vendor directory, or Nix.

## Generating

```shell
tdl gen                       # every target in the default set
tdl gen --target go           # one target
tdl gen --target go --watch   # regenerate on save
tdl gen --check               # exit non-zero if output would change
tdl gen --clean               # remove previously generated files first
tdl gen --target go -o ./out  # override the target block's output path
```

Output paths live in the target block:

```tdl
target go for billing {
  out "./gen/go"
  package "github.com/acme/billing"

  User.email => tag "json:\"email_address\""
  Money      => foreign "github.com/acme/money" "Money"
}
```

`-o` on the command line overrides `out` for that invocation.
Nothing else about a target is configurable from the command line: if a mapping needs to change, it changes in the model's target block where it is reviewed and versioned.

### File layout

The backend decides.
The Go backend emits idiomatic Go, the SQL backend emits whatever SQL wants.
There is no `layout` directive and no way to ask for one-file-per-declaration.
A consumer who needs a different layout needs a different backend.

### Stale output

`tdl gen` only writes.
Deleting a type leaves its generated file on disk until `tdl gen --clean` removes the files a previous run wrote.
Cleaning is opt-in because output directories are not always exclusively owned by tdl.

## Backends

Targets named `go`, `ts`, and `sql` are built in.
Any other target name resolves to `tdl-gen-<name>` on `PATH`; `tdl` resolves the model and pipes the IR to that executable's stdin.
There is no verification, signing, or version check, the same trust model as `git` subcommands and `protoc` plugins.
Declaring a plugin in `[plugins]` documents the dependency and lets a project pin an explicit command, but a plugin on `PATH` works without being declared.

## Inner loop

```shell
tdl check                     # parse and resolve, report errors
tdl gen --target go --watch   # regenerate continuously
tdl lsp                       # language server
```

`tdl check` is the fast path: parse, resolve names, resolve target paths, report.
Editors run it on save, or connect to `tdl lsp` for inline diagnostics, hover, and go-to-definition.

`tdl gen --watch` subsumes the current `tdl play` playground.
Play's token, AST, and formatted views become flags on the existing debug commands rather than a separate watch mode.

## Diagnostics

```
billing.tdl:14:3: unknown type "Momey"
billing.tdl:31:1: target path "User.emial" names nothing
```

`--format json` emits structured diagnostics for editors and CI annotation tooling.

## Checking generated code into git

Both workflows are supported and neither is the default.

Committed:

```yaml
- run: tdl gen --check
```

Generated code is reviewed in diffs, and consumers of the repo do not need `tdl` installed.
CI fails when someone edits a model without regenerating.

Build-time:

```
gen/
```

The output directory is gitignored and generation is a build step.
Drift is impossible, no generated code appears in review, and every consumer needs `tdl` and any plugins available.

## Open questions

- Whether `tdl gen --check` and `tdl check` sharing a name is confusing enough to rename one.
- What `tdl gen --clean` uses to know what it wrote: a lock file in the output directory, a header comment, or re-deriving the file set from the model.
- Whether `[gen] default` belongs in the manifest at all, given target blocks already declare which targets exist.
- How a target block in a dependency interacts with one in the consuming project.
