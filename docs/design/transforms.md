# Transforms

Design document.
Nothing here is built; [transforms-plan.md](transforms-plan.md) orders the work.

A transform is a plugin that takes a resolved model and returns a model, diagnostics, or both.
It lets a project lint or rewrite its model in any language without forking the compiler, the role a transformer plays between parse and stringify in unified.

Generating and importing already sit at the two ends of the pipeline.
A transform sits between lowering and generation:

```text
source -> parse -> lower -> transform* -> generate
```

## What a transform is for

- **Lint.** Every entity carries a `key`, names are PascalCase, a deprecated field names its replacement, no enum has one variant.
  The transform returns diagnostics and no model.
- **Rewrite.** Add audit fields to every entity satisfying `Auditable`, expand a project-specific constraint into standard ones, derive a `Summary` value from each entity, attach directives a convention implies.
  The transform returns a model.

A lint is the common case and the cheap one, so the design keeps it cheap: a response with no model changes nothing and costs no further work.

## Declaring a transform

```tdl
transform lint for billing {
  require_key
  naming(pascal)

  Legacy => allow(naming)
  Entity => require(doc)
}
```

`transform <name> for <package> { ... }` is a top-level declaration with the body of a target block.
`<name>` names the transform, as a target block's name names its backend.
Its entries are directives, resolved with the same paths, nested blocks, and specificity ladder, and they reach the transform on the nodes they resolve to, tagged with the block's name.

Reusing the target block's body is the point.
A transform's configuration is per declaration, per class, and per field for free: `Legacy => allow(naming)` silences one rule on one declaration with no suppression comment syntax.
When [profiles](profiles.md) land, kind selectors and `with` apply to a transform block unchanged, so `@entity => require(doc)` and a shared lint profile need no further design.

`transform` is a declaration keyword and so reserved, and, like every reserved word, a field name when followed by `:`.

A transform block lowers to an `ir.TargetBlock` with `transform` set.
A transform and a target with the same name in one model is an error, because a directive names its block only by name.

## Running

Transforms run after lowering succeeds and before anything reads the model.

| Command | Runs transforms |
| --- | --- |
| `tdl check` | yes, and reports their diagnostics |
| `tdl gen` | yes, then generates from the result |
| `tdl ir` | yes; `--transform=false` prints the lowered model |
| `tdl lsp` | on save, see [Language server](#language-server) |
| `tdl fmt` | no, it reads source only |
| `tdl import` | no |

Transforms run in source order of their blocks, each receiving the model the previous one returned.
Order is visible and reviewable in the model, and a project wanting a different order moves a block.

An error diagnostic from a transform stops the run before the next transform, as a lowering error stops lowering between passes, so no transform reads a model a previous one rejected.

### Scoped to targets

A target block may name transforms that run only for it, with the `transform` directive tdl reads itself, as it reads `out`:

```tdl
target protobuf for billing {
  transform(flatten)
}
```

A transform block named by any target block's `transform` directive runs only for those targets, after every unscoped transform, in the order the target block lists them.
Any other transform block runs for every command in the table.
`transform` is repeatable in a target block, and naming a transform block that does not exist is an error at the name.

Scoping covers the rewrite that suits one language and would be wrong for another, such as flattening mixins for a target without inheritance, without asking the transform to know which target it serves.

## The protocol

A transform speaks the [plugin protocol](plugins.md) in a third mode.
Framing, handshake, directive declarations, diagnostics, reuse, and timeouts are unchanged.

```proto
enum Mode {
  MODE_UNSPECIFIED = 0;
  MODE_IMPORT      = 1;
  MODE_TRANSFORM   = 2;
}

message Features {
  bool reuse     = 1;
  bool reverse   = 2;
  bool transform = 3;
}

message TransformRequest {
  string           name  = 1; // the transform block's name
  tdl.ir.v1.Model  model = 2;
  bool             check = 3; // the result will not be generated from
}

message TransformResponse {
  tdl.ir.v1.Model     model       = 1; // absent: unchanged
  repeated Diagnostic diagnostics = 2;
}
```

Every field is additive, so a plugin built before transforms reads its handshake unchanged and refuses a mode it does not know.

`name` plays the part `Request.target` does: the transform keeps the directives tagged with it, with `plugin.Directives`.
`check` is set by `tdl check` and the language server, so a transform that only rewrites may skip its work, as a backend may skip work under `dry_run`; ignoring it is correct.

The model arrives as a backend receives it: one package, the prelude merged in untagged, every target and transform block's directives on their nodes.

A plugin declares the `transform` feature to receive a request, and one that does not refuses the mode, as a generator refuses import.

### Resolving a name

`internal/transform` holds a registry of built-in transforms, as `internal/gen` does for backends.
Any other name resolves to `tdl-transform-<name>` on `PATH`, with the trust model `tdl-gen-<name>` has.
A separate prefix keeps `tdl gen`'s target names and a project's transform names from colliding; one binary serving both installs under both names.

### In Go

```go
type Transformer interface {
	Describe() Description
	Transform(ctx context.Context, req *TransformRequest) (*TransformResponse, error)
}
```

`plugin.Serve` and `plugin.ServeConn` take a `Describer`, which both `Backend` and `Transformer` satisfy, so every existing caller compiles unchanged.
`ServeConn` serves `MODE_TRANSFORM` to a value implementing `Transformer` whose description declares the feature.

`TestHostsAgree` gains a transform column: a built-in transform returns the same bytes in process and as a plugin.

## Checking a returned model

A transform is arbitrary code and its model is untrusted: an `ID` may index nothing, a field may name a type of the wrong kind, a computed field may disagree with what it is computed from.

tdl checks a returned model by lowering it again.
`internal/unlower` turns it into an `*ast.File`, and `sema.Lower` lowers that tree, with the loader the original lowering used, so imports resolve.
A model that does not lower is an error naming the transform, with lowering's diagnostics beneath it, and nothing after it runs.

This is the check `tdl import` already makes, for the same reason: there is one implementation of what a valid model is, and a second, an `ir` validator, would drift from it.
It also means a transform edits what a model author writes, never what lowering computes.
Struct kinds, satisfaction, mixin fields, inherited constraints, and a class directive's expansion are recomputed, so a transform adding a field to a mixin sees it on every type including it in the next transform's model.

The tree is lowered directly, without printing, so positions survive.
`unlower` gains an option to carry each node's `ir.Position` onto the `ast` node it writes, and lowering copies `ast` positions into `ir` as it always has.

### Positions of new nodes

A diagnostic later in the pipeline needs somewhere to point.
A transform gives a node it creates the position of the node that prompted it, so an error in a derived `Summary` points at the entity it was derived from.
A node with no position is given the position of the transform block's name, so it always points into the model's own files.

The prelude is not the transform's to change.
A prelude declaration that differs from the one sent is an error naming the transform, since unlowering writes only the model's own declarations and the change would otherwise vanish silently.

### Lint is free

A response with no model skips unlowering and lowering, and the next transform receives the model it was given.
A rewrite that changed nothing may return the model it was sent; tdl compares it with `proto.Equal` before lowering again.

## Diagnostics

A transform's diagnostics are the protocol's `Diagnostic`, printed like lowering's.
An error fails `tdl check` and `tdl gen`; a warning is printed and nothing stops.

A rule a user can switch off is configuration, so it belongs in the transform block, not in tdl.
`naming(pascal)` and `Legacy => allow(naming)` above are directives the transform declares and reads; the host knows nothing of rules.
A transform should put its rule in the diagnostic's `code`, such as `naming`, so a user knows what to write.
`tdl.toml`'s `[lossy]` table silences loss codes only, and does not grow a lint table.

Once `ir.Position` carries an end, a transform reports a range, and the editor underlines the span rather than a point.

## Language server

The language server runs unscoped transforms when a document is saved, never on a keystroke, and publishes their diagnostics beside lowering's.
A transform declaring `reuse` is started once per session and kept alive, so a save costs one request.

Go to definition, hover, and the outline read the lowered model, not the transformed one: a reader navigates what they wrote.

## What a transform will not do

**Write source.** A rewrite changes the model a backend receives, never the `.tdl` files.
A fix applied to source is a code action, see [Deferred](#deferred).

**Change another package.** An imported declaration arrives as an `ir.Extern`, and lowering again would discard a change to one, so a change is an error naming the transform, as a change to the prelude is.

**Generate files.** A transform that wants to write a file is a backend.

## Deferred

- **Fixes.** A diagnostic carrying a source edit, offered by the language server as a code action and applied by `tdl fix`. It needs the edit expressed against source, which a model-level transform cannot produce.
- **Parameters.** A transform block takes configuration only as directives; `transform lint("strict") for billing` would need a grammar for arguments in a header.
- **Ordering across files.** Transform blocks run in source order, and a block in an included file runs at the include. A project that splits its transforms across files gets the order the files compose in.
- **Transforms in `tdl.toml`.** The project model in [workflow.md](workflow.md) is unbuilt. When `[plugins]` lands it may name a transform's command, as it names a backend's.
