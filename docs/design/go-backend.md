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

What tells them apart is the file a declaration came from, compared against `prelude.Name` in whole.
Comparing how the name ends would take a user's `my-std.tdl` for the prelude and silently generate nothing for it.
A replacement prelude is named by whoever passed it and is not recognized, so a project that replaces the prelude generates it too.
Marking the prelude on the wire is the fix, and [plugins.md](plugins.md) argues the opposite, that a backend should see prelude declarations as declarations like any other.

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

It carries a cost the other collections do not: a Go map key must be comparable.
TDL says a `Set` holds distinct values and a `Map` is keyed, and says nothing about how a language decides two values are the same, so `Set<bytes>` and `Map<[string], string>` are models a Go backend cannot express.
`map[[]byte]struct{}` is not a weaker guarantee but a compile error, so a set element or a map key whose Go type cannot be one is a warning and the declaration is skipped.
A pointer is comparable whatever it points at, which makes `Set<bytes?>` legal where `Set<bytes>` is not, and an interface is comparable by Go's own rule, so a sealed enum is a legal key and panics only if a variant holding an incomparable field is used as one.

## Structs

Entities, values, and mixins all lower to `ir.Struct`, and all three become a Go struct.

The kinds differ in what they mean, not in what they emit.
Nothing in Go expresses "identity that survives changes to its contents", so an entity and a value are the same declaration with different documentation until a target block's `key` directive names the fields identifying the entity.

A key naming several fields becomes a key type and a method returning it:

```go
type LineItemKey struct {
	Order string
	Sku   string
}

func (l LineItem) Key() LineItemKey {
	return LineItemKey{Order: l.Order, Sku: l.Sku}
}
```

A key naming one field returns that field's type, so `User => key(id)` is `func (u User) Key() UserID`.
The common case reads as what it is, and the cost is that adding a second field changes the return type, which is a breaking change to the model either way.
The key type is named so a consumer can write `map[LineItemKey]LineItem`, and every field in it must be comparable for the same reason.

A parameterized struct is a Go generic type, `type Page[T any] struct`, and applying it is instantiating one, so `Page<Order>` is `Page[Order]`.
A generic entity's key type takes the entity's parameters, as `LineItemKey[T]`.
A method's receiver is named after its type unless a type parameter is already spelled that way, since the two share a scope, and it is `r` then.

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

A generic enum with a field-carrying variant is a generic interface whose marker takes the enum's parameters:

```go
type Result[T any] interface{ isResult(T) }

type ResultOk[T any] struct{ Value T }

func (ResultOk[T]) isResult(T) {}
```

Without the parameters in the marker, a `ResultOk[int]` would satisfy `Result[string]`.
A fieldless enum with parameters keeps its shape and drops them with a warning, where it is declared and wherever it is used.
A Go constant cannot be generic, nothing in a fieldless enum names a parameter, and turning it into the sealed shape to keep them would be a second rule deciding the shape.

## Newtypes and aliases

A newtype is `type N Base`: distinct, not interchangeable, which is what the language says it is.
A parameterized newtype is generic the same way, `type Ids[T any] []T`, except over a bare parameter: Go refuses a type parameter as the whole of a type declaration, so that one warns and is skipped.
Its `where` constraints are its [Validate method](#validation), checked in one place from the set the compiler accumulated down the chain.

An alias is transparent and is expanded rather than emitted.
`ir` already calls it an abbreviation, so there is nothing to generate.
A parameterized alias is expanded with its arguments standing for its parameters, so given `alias Pair<K, V> = {K -> V}`, `Pair<string, int>` is `map[string]int64` wherever it is written.

## Generics

Type parameters reach the backend unmonomorphized, and each becomes a Go type parameter spelled as TDL spells it.

Go needs a map key to be comparable, and TDL has no way to say a parameter is, so the backend infers it.
A parameter reaching a set element, a map key, or a key field is `comparable`, and so is one handed to another declaration's parameter that has to be.
That is decided across the whole model before anything is rendered, since a declaration may hand its parameter to one declared after it.
Inference is the only way `Bag<T> { items: {T} }` compiles, and it is safe to infer because `comparable` can never be written, where a class constraint always is.

The compiler does not check a constraint whose argument is itself a parameter, and Go does.
So every use of a constrained declaration is checked, and one Go would refuse, such as `Bag<bytes>` or an unconstrained `T` handed to `Envelope<T>`, warns at the use and skips the declaration holding it.

A parameter Go cannot express warns at the parameter and skips its declaration:

- a higher kind, written as `f: type -> type` or only used as `f<T>`, since Go has no higher-kinded type parameters;
- a `unit` kind, which waits on phase 6 of [go-backend-plan.md](go-backend-plan.md);
- a name that is a Go keyword, one of Go's predeclared names, a package generated code imports (`errors`, `fmt`, `regexp`, `time`, `utf8`), or the Go name of a generated declaration.

The last rule is conservative on purpose.
`int` is Go's `int64`, so a parameter named `int64` is legal TDL and would capture every field typed `int`.

## Classes

A class is an interface whose one method is unexported, and each declaration satisfying the class carries that method:

```go
type Auditable interface {
	Timestamped
	isAuditable()
}

func (Order) isAuditable() {}
```

Conformance in TDL is nominal and always declared, and an unexported method makes it so in Go: nothing outside the package can implement the interface, and nothing inside implements it without being generated to.
A class it requires is embedded, and the satisfying declarations are read from `Model.Satisfying`, which already closes over the classes a class requires.
A `requires` clause is a constraint naming that interface, so `Envelope<T> requires Auditable<T>` is `type Envelope[T Auditable] struct`, and a parameter that also has to be comparable is constrained by `interface{ comparable; Auditable }`.

A class's fields are not methods of the interface.
The declarations satisfying it declare them already, Go refuses a field and a method with one name, and a getter under another name would be API the model never asked for.
Generated code never calls into a type argument's values, so a constraint's only job is to say which types may be arguments, and the marker says exactly that.

A marker goes in the satisfying declaration's file.
A sealed enum is an interface, which cannot carry a method, so it embeds the class and every variant carries the marker.

What Go cannot express warns where it was written:

| Case | Result |
| --- | --- |
| A class taking parameters, as a multi-parameter or higher-kinded class does | Skipped, since a Go interface cannot state a relationship between types |
| A class requiring associated types | Emitted without them |
| A conditional instance | No marker, since Go cannot give a method to only some instantiations |
| An instance for a prelude type or a type in another package | No marker, since Go cannot add a method to either |
| A newtype over a pointer or an interface | No marker, for the same reason |
| A field whose Go name is the marker's | No marker |
| A `requires` naming a prelude class, a class that is not generated, a class in another package, or anything but a bare parameter | The declaration is emitted without that constraint |

`Entity` is the prelude's, and the prelude is not generated, so `requires Entity<T>` is the last row.

## Validation

`where` constraints are a pair of methods on the type they check:

```go
func (e Email) Validate() error {
	return errors.Join(e.validate("Email", nil)...)
}

func (e Email) validate(path string, errs []error) []error {
	if !patternEmail_0.MatchString(string(e)) {
		errs = append(errs, fmt.Errorf("%s: matches(/^[^@]+@[^@]+$/): no match", path))
	}
	if count := utf8.RuneCountInString(string(e)); count < 3 || count > 254 {
		errs = append(errs, fmt.Errorf("%s: length(3..254): got %d", path, count))
	}
	return errs
}
```

`Validate` reports every violation, joined, and `validate` threads the path a container prefixes.
Without the second, a container prefixing a child's joined error would prefix only its first line.

A message is the path, the constraint as written, and a detail: `Order.items[1].quantity: max(100): got 200`.
It echoes numbers, counts, indices, and enum values, and never a string's or bytes' contents, since validation errors reach logs and those contents are often addresses or secrets.
A set element or a map key is named by its value when that is a number or an enum's, and as `?` otherwise, for the same reason.

The spec gives the standard constraints' forms and leaves their meaning to backends, so this is the meaning here:

| Constraint | Checks |
| --- | --- |
| `min`, `max` | An integer, compared; a float bound compares as `float64` |
| `length` | A string's characters, or the length of bytes, a list, a set, or a map; an integer means exactly that length |
| `matches` | A string, unanchored, against a pattern compiled once per package |
| `oneOf` | A string, integer, or bool equal to one argument, or a fieldless enum's variant named by one |
| `unique` | A list's elements, when Go can compare them; a set is unique already |

A string's length is characters rather than bytes, because a model writing `length(3..254)` for a name means what a person reads.
A pattern is compiled when the code is generated, since Go's `regexp` is RE2 and refuses what other engines accept, and a pattern it refuses would panic when the package loads.
`decimal` is a placeholder string until foreign types, and a text check on `"1.50"` is not what a model means, so every standard constraint on one warns.

A type validates the values it holds: a field whose type validates, what a pointer points at, and every element of a collection, at any depth.
A sealed enum is an interface, which cannot carry a method, so each variant with something to check carries the pair, and a field holding the enum asks the value it holds.
A struct with nothing to check gets neither method.

A newtype is checked in one place, from the whole set the compiler accumulated down its chain, so a newtype over a newtype does not call its parent.
A newtype over a struct validates the struct as well.
A newtype over a pointer or an interface has no methods in Go, so its constraints warn.

A generic type checks its own fields and not its type arguments' values.
Asserting a method on a `T` that holds a nil pointer panics, and avoiding that needs `reflect`.

What the backend cannot check warns at the constraint, and the rest is generated: a name it does not know, since the set is open; a constraint on a type it gives no meaning to; a pattern Go refuses; and a length that can never hold.

## Directives

The backend understands four, and declares all four in its handshake so the compiler can check them before generating anything.

- `package("github.com/acme/billing")`, on the target block.
  The Go package clause is the last path segment.
  Written as an import path because that is what a consumer of the generated code will write, and the clause is derivable from it while the reverse is not.
  A last segment that comes out a Go keyword, as `github.com/acme/type` does, is an error and not a warning: the clause is one identifier every file carries, so what it makes unusable is the whole output rather than a declaration to skip.
- `name("Account")`, on a declaration or a field.
  Overrides the Go identifier.
  TDL names and Go names disagree often enough that a rename has to be expressible, and renaming in the model would change the model to suit one backend.
- `tag("json:\"email_address\"")`, on a field.
  Emitted verbatim as the struct tag.
  The backend does not parse it: a struct tag is an open convention, and any grammar imposed here would be one more thing to keep current with whatever reflects over it.
- `key(order, sku)`, on an entity.
  The fields identifying it, as bare names, generating the `Key()` method described under [Structs](#structs).
  The handshake declares any number of arguments and no kinds, since `arg_kinds` constrains by position; that each is a name is checked by the backend.

A directive the backend does not declare is a warning from the compiler and is passed through anyway, so a target block can carry a directive for a future phase without failing today.

Directives arrive resolved and arbitrated, tagged with the block they came from.
The backend filters by target and never computes precedence.

A directive expanded from a class carries `from_class`, which the backend does not read.
A directive written against `Auditable` and one written against `User` mean the same thing by the time they arrive; where it came from is for a diagnostic to explain, not for the generator to act on.

## Names

A declaration's `Meta.name` is fully qualified, and the Go identifier is its last segment, exported.
A field's name is bare and is exported the same way.
A type parameter keeps its TDL spelling, since nothing outside its declaration names it; [Generics](#generics) says which spellings are refused.

Exporting is unconditional.
Nothing in TDL says a declaration or a field is private, and inventing a rule out of the leading character would make a case change in the model a visibility change in the output.

There is no initialism table, so `id` becomes `Id` rather than `ID`.
A table would be right most of the time and produce, the rest of the time, a name the consumer cannot predict from the model.
`name("ID")` is how they say what they want instead, and a rule that is always wrong in the same way is easier to work with than one that is usually right.

An identifier that collides with a Go keyword after exporting cannot, since exporting capitalizes and every Go keyword is lower case.

## Diagnostics

The backend reports what it cannot handle rather than emitting something plausible and wrong.

A unit-typed field, an extern, a set element or map key Go cannot compare, a type parameter Go cannot express, and a type argument breaking a constraint each produce a warning with the node's position, and the declaration reaching one is skipped.
A declaration naming a skipped one, directly or through an option, a collection, or an alias, is skipped with it and warns at the use, since it would otherwise name a type the package does not declare.
A `where` constraint the backend cannot check, a `requires` clause Go cannot state, a class's associated types, and a fieldless enum's parameters warn and the declaration is still emitted, since what is missing is the constraint and not the type.
A field whose Go name is `Validate` or `validate` warns the same way, and the type is emitted without the methods.
What [Classes](#classes) lists as getting no marker warns, and the declaration is emitted without the marker.
A `key` the backend cannot generate warns the same way and the entity is emitted without it: a key on a value or a mixin, an argument that is not a name, a field named twice or not at all, a field Go cannot compare, a field whose Go name is `Key`, and a key type colliding with a declaration of the same name.
Everything above has a phase or a deferred decision in [go-backend-plan.md](go-backend-plan.md), or a section here saying why Go cannot express it, and each is a set of decisions rather than an oversight.
A key warning is a mistake in the model, except a field Go cannot compare, which is the comparability decision again.

A warning does not stop a run, so a model that is mostly generatable generates.
An error stops it, and the two things that earn one are output `go/format` refuses to parse and a package clause Go will not accept, both of which are the package rather than a part of it.

## Formatting

The backend runs the output through `go/format` before returning it.

`Response.post` exists on the wire for exactly this and nothing on the compiler side reads it, so a backend that wanted formatted output today would have to depend on `gofmt` being installed.
Formatting in process needs no such thing, and `go/format` is the same code `gofmt` is.

If formatting fails, the unformatted source is returned along with an error-severity diagnostic.
Malformed output the consumer can read beats no output at all, and a generator that cannot produce parseable Go has a bug that should be visible.
