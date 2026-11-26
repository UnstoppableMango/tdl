# Yolo

The `yolo` target is a delegated backend: it hands the mapping to a language model instead of encoding it.
Use it for a language TDL has no backend for, such as Elixir, Kotlin, Swift, or Dart, or for a one-off target, such as an ORM schema, a form, or a doc page, that will never deserve a backend of its own.
The name is a warning to anyone reading the model: nothing about the output is promised.

It is a plugin, `tdl-gen-yolo`, which `tdl gen` runs from `PATH` and the Nix package ships beside `tdl`.
`tdl` itself never talks to a model.
[#1505](https://github.com/UnstoppableMango/tdl/issues/1505) is its spec, and its sub-issues are the phases: phase 1 is built, and checks, retries, and an agent CLI are not.

```tdl
target yolo for shop {
  out("gen/ex")
  language("elixir")
  prompt("Ecto embedded schemas, one module per declaration")
}
```

## What it does not promise

The model writes whatever files it decides on, and the same request can come back different every time.
Nothing checks that they compile, follow the JSON wire convention, or hold every declaration, until a check command does (phase 2).
Read what it writes before committing it, as you would a colleague's first draft.

It never stands in for a missing backend: `tdl gen` runs it only for a block that says `target yolo`, and no `tdl-gen-<language>` name runs it.
It does not import.

## How it works

The backend holds one conversation per target block with the model.
The prompt holds the model as TDL, the block's `language` and `prompt`, what each loss code means, the JSON wire convention, and the [specification](../spec.md).
The model answers with two tools:

- `write_file(path, content)` writes one file under `out`.
  A path that is absolute or holds `..` ends the conversation with an error, and nothing is written.
- `report_loss(decl, code, message)` reports what the files cannot carry.
  Each becomes a warning positioned at the declaration, field, or variant it names, under its loss code, so `tdl.toml`'s `[lossy]` table can allow it as it would any backend's.
  An unknown code or name is refused back to the model.

A conversation that writes no file is an error, and one is cut off after 50 turns.

## Directives

Every directive goes in the target block.

| Directive | Meaning |
| --- | --- |
| `language("elixir")` | The language to write. Required. |
| `prompt("...")` | Free text added to the prompt, to fit the output to your codebase. |
| `model("claude-opus-5-5")` | The model answering. `claude-opus-5-5` when absent. |
| `effort("medium")` | How much effort the model spends: `low`, `medium`, `high`, `xhigh`, or `max`. `medium` when absent. |

Changing any directive, the model's name included, changes the request and so regenerates.

### Checks and agents

Two directives are planned, each naming a command in `tdl.toml` the way [`post`](../architecture/plugins.md) does:

```toml
[check]
mix = "mix compile --warnings-as-errors"

[agent]
claude = "claude -p"
```

- `check("mix")` will run the `[check]` command over a copy of the output and send a failure back to the model, at most `tries(N)` times, 3 by default; the last failure is an error that writes nothing ([#1508](https://github.com/UnstoppableMango/tdl/issues/1508)).
- `agent("claude")` will hand the whole job to an agent CLI with its own tools in place of the API call ([#1509](https://github.com/UnstoppableMango/tdl/issues/1509)).

## Regeneration and reuse

The backend declares itself nondeterministic, so `tdl` keeps its output while the request is unchanged ([ADR-0002](../adr/0002-host-reuses-nondeterministic-output.md)).
`tdl gen` records a hash of the request in `.tdl-output`, with each file it wrote and the warnings the model reported.
While the next request hashes the same and the files are there unedited, it keeps them, reports the recorded warnings again, and starts no conversation.

So regenerating is a deliberate act: change the model or a directive, or run `tdl gen --clean`.
Moving a declaration within its file does not regenerate, and editing or removing a kept file does.
`tdl gen --verify`, the flake module's `tdl-gen` check, and CI only compare hashes, so they need no key and no network.
Commit the output and `.tdl-output` together.

## Cost

Each regeneration is a paid conversation with the model, billed to the key in `ANTHROPIC_API_KEY`; without one, `tdl gen` fails before anything is sent.
The prompt carries the whole specification, several thousand tokens, before the source of your own model.
`effort` trades cost for care: `high` and above spend noticeably more, and the 50-turn cap bounds a conversation that never ends.
Reuse is what keeps the bill to the regenerations you ask for.

## Features

What the output does with each feature of the language is up to the model, so no cell can be measured: grading one recorded answer would promise nothing about the next.
Each Generate cell reads "Depends on the model", and `make features` checks the table's shape without rewriting it.
It does not import, so every Import cell is No.

<!-- tdl:features yolo -->

| Feature | Generate | Import | Target spelling | Notes |
| --- | --- | --- | --- | --- |
| Entity, value, and mixin | Depends on the model | No | | |
| Mixin `include` | Depends on the model | No | | |
| Fieldless enum | Depends on the model | No | | |
| Fielded enum | Depends on the model | No | | |
| Newtype | Depends on the model | No | | |
| Alias | Depends on the model | No | | |
| Optional and nullable | Depends on the model | No | | |
| List, set, and map | Depends on the model | No | | |
| `min` | Depends on the model | No | | |
| `max` | Depends on the model | No | | |
| `length` | Depends on the model | No | | |
| `matches` | Depends on the model | No | | |
| `oneOf` | Depends on the model | No | | |
| `unique` | Depends on the model | No | | |
| `owned` | Depends on the model | No | | |
| Field default | Depends on the model | No | | |
| `key` directive | Depends on the model | No | | The backend declares no `key` directive; ask for keys in `prompt`. |
| `name` directive | Depends on the model | No | | The backend declares no `name` directive; a name the language forces it to change is reported as `lossy.name`. |
| `number` directive | Depends on the model | No | | The backend declares no `number` directive. |
| Doc comment | Depends on the model | No | | |
| Type parameters | Depends on the model | No | | |
| Class | Depends on the model | No | | |
| Unit | Depends on the model | No | | |
| Import and extern | Depends on the model | No | | Each extern warns `lossy.unsupported`, since the model is not shown what its dependency generates ([#1521](https://github.com/UnstoppableMango/tdl/issues/1521)). |

<!-- /tdl:features -->
