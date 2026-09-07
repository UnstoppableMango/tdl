# The Go backend

Design document.

The Go backend turns a resolved model into Go source.
It is the first code generator in this repository, and the first thing to answer the question [plugins.md](plugins.md) deliberately left open: what generated code should look like.

It is called `go` in a target block, ships compiled into `tdl`, and ships again as `tdl-gen-go` on `PATH`.
Both are the same value behind [`plugin.Backend`](../../plugin/backend.go), for the reason `plugins.md` gives: a surface only one of the two hosts can reach is one nothing keeps honest.

## What it emits

One package, in one file per declaration the model owns, named after the declaration in snake case.

[workflow.md](workflow.md) already settled that the backend decides layout and that there is no `layout` directive.
One file per declaration is the layout, because a generated tree is read as a diff more often than it is read as a program, and a declaration that moves should not move everything under it.

The prelude arrives merged into `Model.decls` untagged, so the backend emits only what the model's own file declared.
Prelude declarations are the type vocabulary, not output.

## The type mapping

A TDL primitive has no Go type of its own, so each is a decision rather than a translation.

| TDL | Go | Why |
| --- | --- | --- |
| `string` | `string` | |
| `int` | `int64` | The spec puts no width on `int`, and a model that outgrows 32 bits should not be a silent truncation. |
| `bool` | `bool` | |
| `bytes` | `[]byte` | |
| `uuid` | `string` | The standard library has no UUID type, and choosing a third-party one for every consumer is not the backend's call. `foreign` is the way out. |
| `instant` | `time.Time` | |
| `date` | `time.Time` | Go has no date-without-time type either, and a `time.Time` at midnight UTC is what everything else in the ecosystem does. |
| `duration` | `time.Duration` | |
| `decimal` | `string` | This is the uncomfortable one. `float64` is wrong for the thing `decimal` exists to express, and every correct answer is a dependency. A string round-trips exactly and forces the consumer to pick a library rather than the backend picking one badly. |

Three of these are placeholders waiting on `foreign`, which is how a target block names an existing Go type for a TDL one.
Until it lands, a project that wants `decimal.Decimal` has to post-process, and that is the cost of not guessing.

Collections come from the prelude, so the mapping reads the type constructor rather than any syntax:

| TDL | Go |
| --- | --- |
| `List<T>`, `[T]` | `[]T` |
| `Set<T>`, `{T}` | `map[T]struct{}` |
| `Map<K, V>`, `{K -> V}` | `map[K]V` |
| `Option<T>`, `T?` | `*T` |
| `Nullable<T>`, `T \| null` | `*T` |

`SyntacticForm` records which spelling was written, and the backend does not read it.
Both spellings of an option mean the same thing, `ir` says lowering is authoritative, and a generator that emitted a wrapper struct for one and a pointer for the other would make the sugar semantic after the compiler decided it was not.

`Set` as a map is a compromise.
Go has no set, and `[]T` for a set would silently permit the duplicates the type exists to forbid.

## Structs

`entity`, `value`, and `mixin` all lower to `ir.Struct`, and all three become a Go struct.

The kinds differ in what they mean, not in what they emit.
Nothing in Go expresses "identity that survives changes to its contents", so an entity and a value are the same declaration with different documentation, and the `key` fields are what a later phase turns into an identity method.

A mixin's fields are copied into whatever includes it, and `Field.included_from` says where each came from.
The mixin still gets a struct of its own, because it is a declaration a consumer may name.
The including struct does not embed it: the fields are already there, flattened by lowering, and embedding on top would double them.

## Enums

An enum is two different Go shapes, chosen by whether any variant carries fields.

A fieldless enum is a named string type and one constant per variant:

```go
type Status string

const (
	StatusActive  Status = "Active"
	StatusPending Status = "Pending"
)
```

An enum where any variant carries fields is a sealed interface and one struct per variant:

```go
type Shape interface{ isShape() }

type ShapeCircle struct{ Radius float64 }

func (ShapeCircle) isShape() {}
```

The split is the load-bearing decision in this backend, and it is worth saying why the two uniform answers are worse.

Always emitting the interface makes `Status` unusable as a map key, unserializable without hand-written code, and impossible to write as a constant, all to express a choice among three names.
Always emitting constants cannot express a variant with fields at all, and the language calls that its sum type.

So the shape follows the declaration.
This means adding a field to one variant changes the generated Go for the whole enum, which is a real cost and the reason the rule is written down here rather than discovered from the output.
It is also honest: adding a field to a variant is a breaking change to the model, and the generated code saying so is better than it not.

The string a fieldless variant carries is the variant's name as written.
A backend that invented a wire format would be making a decision that belongs to the consumer, and `tag` is where a consumer states one.

## Newtypes and aliases

A newtype is `type N Base`: distinct, not interchangeable, which is what the language says it is.
Its `where` constraints are not enforced in phase 1; validation is its own phase and its own set of decisions about where the check lives.

An alias is transparent and is expanded rather than emitted.
`ir` already calls it an abbreviation, so there is nothing to generate.

## Directives

The backend understands three, and declares all three in its handshake so the compiler can check them before generating anything.

- `package("github.com/acme/billing")`, on the target block. The Go package clause is the last path segment. Written as an import path because that is what a consumer of the generated code will write, and the clause is derivable from it while the reverse is not.
- `name("Account")`, on a declaration or a field. Overrides the Go identifier. TDL names and Go names disagree often enough that a rename has to be expressible, and renaming in the model would change the model to suit one backend.
- `tag("json:\"email_address\"")`, on a field. Emitted verbatim as the struct tag. The backend does not parse it: a struct tag is an open convention, and any grammar imposed here would be one more thing to keep current with whatever reflects over it.

A directive the backend does not declare is a warning from the compiler and is passed through anyway, so a target block can carry a directive for a future phase without failing today.

Directives arrive resolved and arbitrated, tagged with the block they came from.
The backend filters by target and never computes precedence.

A directive expanded from a class carries `from_class`, which the backend does not read.
A directive written against `Auditable` and one written against `User` mean the same thing by the time they arrive; where it came from is for a diagnostic to explain, not for the generator to act on.

## Names

A declaration's `Meta.name` is fully qualified, and the Go identifier is its last segment, exported.
A field's name is bare and is exported the same way.

Exporting is unconditional.
Nothing in TDL says a declaration or a field is private, and inventing a rule out of the leading character would make a case change in the model a visibility change in the output.

There is no initialism table, so `id` becomes `Id` rather than `ID`.
A table would be right most of the time and produce, the rest of the time, a name the consumer cannot predict from the model.
`name("ID")` is how they say what they want instead, and a rule that is always wrong in the same way is easier to work with than one that is usually right.

An identifier that collides with a Go keyword after exporting cannot, since exporting capitalizes and every Go keyword is lower case.

## Diagnostics

The backend reports what it cannot handle rather than emitting something plausible and wrong.

A type parameter, a unit-typed field, a class declaration, an extern, and a `where` constraint each produce a warning with the node's position and are skipped.
Each is a phase in [go-backend-plan.md](go-backend-plan.md), and each is a set of decisions rather than an oversight.

A warning does not stop a run, so a model that is mostly generatable generates.

## Formatting

The backend runs the output through `go/format` before returning it.

`Response.post` exists on the wire for exactly this and nothing on the compiler side reads it, so a backend that wanted formatted output today would have to depend on `gofmt` being installed.
Formatting in process needs no such thing, and `go/format` is the same code `gofmt` is.

If formatting fails, the unformatted source is returned along with an error-severity diagnostic.
Malformed output the consumer can read beats no output at all, and a generator that cannot produce parseable Go has a bug that should be visible.
