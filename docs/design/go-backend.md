# The Go backend

Design document.

The Go backend turns a resolved model into Go source.
It was the first code generator here, and the first to answer what [plugins.md](plugins.md) left open: what generated code should look like.

It is called `go` in a target block, ships compiled into `tdl`, and ships again as `tdl-gen-go` on `PATH`.
Both are the same value behind [`plugin.Backend`](../../plugin/backend.go), as `plugins.md` requires.

## What it emits

One package, in one file per declaration the model owns, named after the declaration in snake case.
The package's doc comment goes on the package clause of the first file by path.
[workflow.md](workflow.md) leaves layout to the backend.
One file per declaration keeps a diff of the generated tree local to the declaration that changed.
A `file` directive in the target block writes every declaration into that one file instead, for a target generating into a package that also holds hand-written code.

Go reads meaning into a file name's suffix (`foo_test.go`, `order_linux.go`), so a name whose last underscore-separated element is `test`, a GOOS, or a GOARCH gets `_tdl` appended: `foo_test_tdl.go`.
The lists are `go/build`'s, from `internal/syslist`, and everything up to the first underscore is skipped as `go/build` skips it, so `linux.go` is an ordinary file.
A declaration named `FooTestTdl` collides with an escaped `FooTest`.

The backend emits only what the model's file declared, recognizing the merged-in prelude by `prelude.Name` in whole, so a user's `my-std.tdl` is not mistaken for it.
A replacement prelude is not recognized and is generated too.

## The type mapping

| TDL | Go | Why |
| --- | --- | --- |
| `string` | `string` | |
| `int` | `int64` | The spec puts no width on `int`, and outgrowing 32 bits should not truncate silently. |
| `int32` | `int32` | |
| `uint32` | `uint32` | |
| `int64` | `int64` | |
| `uint64` | `uint64` | |
| `float32` | `float32` | |
| `float64` | `float64` | |
| `bool` | `bool` | |
| `bytes` | `[]byte` | |
| `uuid` | `string` | The standard library has no UUID type, and picking a third-party one for every consumer is not the backend's call. See [Foreign types](#foreign-types). |
| `instant` | `time.Time` | |
| `date` | `time.Time` | Go has no date-only type; midnight UTC is the ecosystem convention. |
| `duration` | `time.Duration` | |
| `decimal` | `string` | `float64` is wrong for what `decimal` expresses, and every correct answer is a dependency. A string round-trips exactly and leaves the library choice to the consumer. |

`uuid`, `date`, and `decimal` are placeholders a target block replaces with `foreign`: `decimal => foreign("github.com/shopspring/decimal", "Decimal")` gives `decimal.Decimal` everywhere the model says `decimal`.

Collections come from the prelude, so the mapping reads the type constructor:

| TDL | Go |
| --- | --- |
| `List<T>`, `[T]` | `[]T` |
| `Set<T>`, `{T}` | `map[T]struct{}` |
| `Map<K, V>`, `{K -> V}` | `map[K]V` |
| `Option<T>`, `T?` | `*T` |
| `Nullable<T>`, `T \| null` | `*T` |

The backend does not read `SyntacticForm`: both spellings of an option mean the same thing, and lowering is authoritative.

`Set` is a map because `[]T` would permit the duplicates the type forbids.
A Go map key must be comparable, so `Set<bytes>` and `Map<[string], string>` cannot be expressed: a set element or map key whose Go type is not comparable is a warning, and the declaration is skipped.
A pointer is comparable, so `Set<bytes?>` is legal.
An interface is comparable by Go's rule, so a sealed enum is a legal key and panics only if a variant holding an incomparable field is used as one.

## Structs

Entities, values, and mixins all lower to `ir.Struct`, and all three become a Go struct.
Go has no notion of identity, so an entity and a value emit the same code until a target block's `key` directive names the fields identifying the entity.

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
Adding a second field then changes the return type, which is a breaking change to the model either way.
Every key field must be comparable, so a consumer can write `map[LineItemKey]LineItem`.

A parameterized struct is a Go generic type, `type Page[T any] struct`, and `Page<Order>` is `Page[Order]`.
A generic entity's key type takes the entity's parameters, as `LineItemKey[T]`.
A method's receiver is named after its type, or `r` when a type parameter is already spelled that way.

A mixin's fields are copied into whatever includes it by lowering, and `Field.included_from` says where each came from.
The mixin gets its own struct, since a consumer may name it, and the including struct does not embed it, which would double the fields.

## Enums

An enum is one of two Go shapes, chosen by whether any variant carries fields.

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

Always emitting the interface would make `Status` unusable as a map key or a constant; always emitting constants cannot express a variant with fields.
The cost is that adding a field to one variant changes the generated Go for the whole enum, which is a breaking change to the model anyway.

A fieldless variant's string is its name as written; a consumer states a wire format with `tag`.
A variant's doc comment and deprecation go on its constant.

A generic enum with a field-carrying variant is a generic interface whose marker takes the enum's parameters, so a `ResultOk[int]` does not satisfy `Result[string]`:

```go
type Result[T any] interface{ isResult(T) }

type ResultOk[T any] struct{ Value T }

func (ResultOk[T]) isResult(T) {}
```

A fieldless enum with parameters keeps its shape and drops them with a warning, where it is declared and wherever it is used, since a Go constant cannot be generic.

## Newtypes and aliases

A newtype is `type N Base`.
A parameterized newtype is generic the same way, `type Ids[T any] []T`, except over a bare parameter, which Go refuses: that one warns and is skipped.
Its `where` constraints are its [Validate method](#validation).

An alias is expanded rather than emitted.
A parameterized alias is expanded with its arguments substituted, so given `alias Pair<K, V> = {K -> V}`, `Pair<string, int>` is `map[string]int64`.

## Generics

Type parameters reach the backend unmonomorphized, and each becomes a Go type parameter spelled as TDL spells it.

TDL cannot say a parameter is comparable, so the backend infers it.
A parameter reaching a set element, a map key, or a key field is `comparable`, and so is one handed to another declaration's parameter that has to be.
This is decided across the whole model before rendering, since a declaration may hand its parameter to one declared after it.

The compiler does not check a constraint whose argument is itself a parameter, and Go does.
So every use of a constrained declaration is checked, and one Go would refuse, such as `Bag<bytes>` or an unconstrained `T` handed to `Envelope<T>`, warns at the use and skips the declaration holding it.

A parameter Go cannot express warns at the parameter and skips its declaration:

- a higher kind, written as `f: type -> type` or only used as `f<T>`;
- a `unit` kind, which waits on phase 6 of [go-backend-plan.md](go-backend-plan.md);
- a name that is a Go keyword, a Go predeclared name, a package generated code imports (`errors`, `fmt`, `regexp`, `time`, `utf8`), or the Go name of a generated declaration.

The last rule is conservative on purpose: `int` is Go's `int64`, so a parameter named `int64` would capture every field typed `int`.

## Classes

A class is an interface with one unexported method, and each declaration satisfying the class carries that method:

```go
type Auditable interface {
	Timestamped
	isAuditable()
}

func (Order) isAuditable() {}
```

TDL conformance is nominal and declared, and an unexported method makes it so in Go: nothing outside the package can implement the interface.
A class it requires is embedded, and the satisfying declarations come from `Model.Satisfying`, which already closes over required classes.
A `requires` clause is a constraint naming the interface, so `Envelope<T> requires Auditable<T>` is `type Envelope[T Auditable] struct`, and a parameter that must also be comparable gets `interface{ comparable; Auditable }`.

A class's fields are not interface methods: Go refuses a field and a method with one name, and generated code never calls into a type argument's values.

A marker goes in the satisfying declaration's file.
A sealed enum is an interface and cannot carry a method, so it embeds the class and every variant carries the marker.

What Go cannot express warns where it was written:

| Case | Result |
| --- | --- |
| A class taking parameters, as a multi-parameter or higher-kinded class does | Skipped |
| A class reaching itself through what it requires | Skipped |
| A class requiring associated types | Emitted without them |
| A conditional instance | No marker, since Go cannot give a method to only some instantiations |
| An instance for a prelude type or a type in another package | No marker, since Go cannot add a method to either |
| A newtype over a pointer or an interface | No marker, for the same reason |
| A field whose Go name is the marker's | No marker |
| A `requires` naming a prelude class, a class that is not generated, a class in another package, or anything but a bare parameter | The declaration is emitted without that constraint |
| A class requiring one of those, where the interface would embed it | The interface is emitted without the embed |

`Entity` is the prelude's and is not generated, so `requires Entity<T>` is the second-to-last row and `class Auditable requires Entity` is the last.
Both warn, because an interface missing its embed is satisfied by types that do not satisfy what the model requires.

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

`Validate` reports every violation, joined; `validate` threads the path a container prefixes, which a joined error could not carry past its first line.

A message is the path, the constraint as written, and a detail: `Order.items[1].quantity: max(100): got 200`.
It never echoes a string's or bytes' contents, since those reach logs and are often addresses or secrets; a set element or map key with such a value is named `?`.

The spec leaves the standard constraints' meaning to backends:

| Constraint | Checks |
| --- | --- |
| `min`, `max` | An integer, compared; a float bound compares as `float64` |
| `length` | A string's characters, or the length of bytes, a list, a set, or a map; an integer means exactly that length |
| `matches` | A string, unanchored, against a pattern compiled once per package |
| `oneOf` | A string, integer, or bool equal to one argument, or a fieldless enum's variant named by one |
| `unique` | A list's elements, when Go can compare them; a set is unique already |

A string's length counts characters, since `length(3..254)` on a name means what a person reads.
A pattern is compiled at generation time, since Go's RE2 refuses what other engines accept and a refused pattern would panic when the package loads.
An unmapped `decimal` is a placeholder string, and a text check on `"1.50"` is not what a model means, so every standard constraint on one warns.

A type validates the values it holds: a field whose type validates, what a pointer points at, and every element of a collection, at any depth.
A sealed enum's variants each carry the pair when they have something to check, and a field holding the enum asks the value it holds.
A struct with nothing to check gets neither method.

A newtype is checked in one place, from the set the compiler accumulated down its chain, so it does not call its parent.
A newtype over a struct validates the struct as well.
A newtype over a pointer or an interface has no methods in Go, so its constraints warn.

A generic type checks its own fields and not its type arguments' values, since asserting a method on a `T` holding a nil pointer panics without `reflect`.

What the backend cannot check warns at the constraint and the rest is generated: an unknown name, a constraint on a type it gives no meaning to, a pattern Go refuses, and a length that can never hold.

## Foreign types

`Money => foreign("github.com/shopspring/decimal", "Decimal")` says a declaration is a type another package declares.
The generated package imports and refers to it and declares nothing for it, so every field naming it reads `decimal.Decimal`.
A primitive is a declaration like any other, so `decimal`, `uuid`, and `date` are mapped the same way.

Every import gets an alias derived from the path, because a package's name is not always its path's last segment and the model does not say which it is.
The alias is the last segment as an identifier, the last two joined when that is taken, and then a number, so two packages ending in `template` become `template` and `htmltemplate`.
A major version suffix and a `go-` prefix are dropped, so `gopkg.in/yaml.v3`, `example.com/money/v2`, and `github.com/google/go-cmp` are `yaml`, `money`, and `cmp`.
An alias that would shadow a declaration, another import, or a predeclared identifier counts as taken, and a type parameter spelled like one warns as [Generics](#generics) describes.
Mapping onto a package the backend imports itself, such as `time`, is not a collision.

A foreign declaration takes type arguments like any other, so a mapped `Holder<string>` is `sync.Map[string]`.

A foreign type is assumed comparable, since whether it is a legal map key is the declaring package's answer, and refusing it would refuse `Set<uuid>` once `uuid` is mapped.
A mapping that is wrong about it fails at the consumer's build.

Go declares a method beside its type, so a foreign type gets none: a `where` constraint, a class it satisfies, and a `key` on it each warn.
A mapping the backend cannot refer to, with no import path or naming something that is not an exported Go identifier, warns and the declaration is generated as though it were unmapped.

An extern, a declaration an imported TDL package owns, is mapped the same way: a target path can name a declaration a `_` import merged in, so `Money => foreign("github.com/acme/money", "Money")` reaches it.
An extern nothing maps has no Go type, so a declaration naming it warns and is skipped, and so is each declaration naming that one.

## Directives

The backend understands seven and declares all seven in its handshake, so the compiler checks them before generating.

- `package("github.com/acme/billing")`, on the target block.
  The package clause is the last path segment; the import path is what a consumer writes, and the clause derives from it.
  A last segment that is a Go keyword, as in `github.com/acme/type`, is an error, since every file carries the clause.
- `name("Account")`, on a declaration, a field, or an enum variant.
  Overrides the Go identifier without renaming the model.
  On a variant it replaces the whole identifier, enum prefix included, and a constant's value stays the variant's TDL name.
- `tag("json:\"email_address\"")`, on a field.
  Emitted verbatim as the struct tag; a struct tag is an open convention, so the backend does not parse it.
  On a declaration or a variant it has nothing to set, so it warns at the directive.
- `key(order, sku)`, on an entity.
  The fields identifying it, as bare names, generating the `Key()` method under [Structs](#structs).
  The handshake declares any number of arguments and no kinds, since `arg_kinds` constrains by position; the backend checks that each is a name.
- `foreign("github.com/shopspring/decimal", "Decimal")`, on a declaration.
  The import path of a Go package and the type it declares, under [Foreign types](#foreign-types).
  Two arguments, because splitting one string on its last dot would get `gopkg.in/yaml.v3` wrong.
- `file("model.go")`, on the target block.
  Writes every declaration into that one file, sharing one import block; on a declaration it warns.
- `roundtrip`, bare, on the target block.
  Writes as `//tdl:` comment directives what import would not read back, under [Import](#import).

A directive the backend does not declare is a compiler warning and is passed through anyway.

Directives arrive resolved and arbitrated, tagged with their block; the backend filters by target and never computes precedence.
It does not read `from_class`.

## Names

A declaration's `Meta.name` is fully qualified, and the Go identifier is its last segment, exported.
A field's name is bare and is exported the same way.
A type parameter keeps its TDL spelling; [Generics](#generics) says which spellings are refused.

Exporting is unconditional, since TDL has no notion of private, and it keeps every identifier clear of Go's lower-case keywords.

There is no initialism table, so `id` becomes `Id`, not `ID`.
A table would produce names a consumer cannot predict from the model; `name("ID")` says what they want instead.

## Diagnostics

The backend reports what it cannot handle rather than emitting something plausible and wrong.

These warn with the node's position, and the declaration reaching one is skipped: a unit-typed field, an extern with no mapping, a set element or map key Go cannot compare, a type parameter Go cannot express, and a type argument breaking a constraint.
`emit.Cascade` then skips every declaration naming a skipped one, directly or through an option, a collection, or an alias, with the warning at the referring declaration naming the cause.

These warn and the declaration is still emitted:

- a `where` constraint the backend cannot check, a `requires` clause Go cannot state, a class's associated types, and a fieldless enum's parameters;
- a field whose Go name is `Validate` or `validate`, and the type is emitted without the methods;
- what [Foreign types](#foreign-types) lists for a foreign type, and what [Classes](#classes) lists as getting no marker;
- a `key` the backend cannot generate, and the entity is emitted without it: a key on a value or a mixin, an argument that is not a name, a field named twice or not at all, a field Go cannot compare, a field whose Go name is `Key`, and a key type colliding with a declaration of the same name.

A warning does not stop a run, so a model that is mostly generatable generates.
An error stops it, and only two things earn one: output `go/format` cannot parse, and a package clause Go will not accept.
Under `roundtrip`, a keyword clause gets a trailing underscore instead, since an annotation carries the package.

A warning about something the model loses carries a loss code from [reverse.md](reverse.md), so `tdl.toml` can silence it; [Import](#import) lists the ones only import would notice.

## Import

`tdl import --from go` reads the `.go` files of one package with `go/parser`, sorted by path, without type checking.
A `_test.go` file warns and is not read.
The package clause is the TDL package, except `main`, which is none.

| Go | TDL |
| --- | --- |
| struct | struct, conforming to `Entity` when it has a `Key` method |
| named `string` type with typed string constants | enum, each constant's value a variant |
| interface whose only method is the marker `is<Name>`, carried by structs named `<Name><Variant>` that nothing else names | enum with fields, one variant per struct |
| any other interface whose only method is its marker | class, its embedded interfaces its superclasses |
| `isX()` on a type, for a class `X` | conformance to `X` |
| any other named type | newtype |
| `type T = X` | alias |
| `[T any]`, `[T comparable]`, `[T C]`, `[T interface{ comparable; C }]` | `<T>`, with `requires C<T>` for a class |
| `[]T`, `map[K]struct{}`, `map[K]V`, `*T`, `G[A]` | `[T]`, `{K}`, `{K -> V}`, `T?`, `G<A>` |
| `string`, `bool`, `int32`, `int64`, `uint32`, `uint64`, `float32`, `float64`, `[]byte` | the primitive of that name, `bytes` for `[]byte` |
| `int`, `int8`, `int16`, `rune`, `uint`, `uint8`, `uint16`, `uintptr` | the nearest wider primitive, with `lossy.primitive` |
| `time.Time`, `time.Duration` | `instant`, `duration` |
| another package's `pkg.T` | an empty struct `T`, with a `foreign` directive |
| a func, a var, a const of another kind, a method Generate does not write, an embedded field, an array, a channel, a func type, an anonymous struct or interface, a pointer to a pointer | `lossy.unsupported` |

A field's name is its Go name with the leading capitals lower case, but for one starting the next word, so `ID` is `id` and `HTTPServer` is `httpServer`.
A variant's name is a constant's value, or what its struct's name adds to the interface's.
A doc comment is the Go one's text, and a last paragraph starting `Deprecated: ` is the deprecation, its lines the reason's.

What Generate writes is read back as the model it says:

- a constraint, from the message `validate` writes, `%s.<field>: <constraint>: <detail>`, since the constraint is written as TDL; a newtype's inherited constraints, which `validate` checks again, are dropped for lowering to add back;
- a `key`, from what `Key` returns: one field, or the fields of the `<Name>Key` struct, which is not a declaration of its own.

The target block holds a `name` directive where the convention would write another name, a struct field's `tag`, a `key`, a `foreign` mapping, and a `file` directive when one file holds declarations not named for it; several such files warn, since they regenerate one per declaration.

Generating, every fact the package cannot hold warns with its loss code: `int`, `uuid`, `decimal`, and `date` written as another primitive, collection sugar written by name, `T | null` and `Option<T>`, an alias expanded, an entity without a key or a mixin, an include, a constraint not checked, `owned`, a default, a name or a package clause that reads back otherwise, a doc comment gofmt changes, conformance its markers do not say, a key not generated, a foreign declaration that is not an empty struct, a class's fields, a unit, generics Go cannot state, and files whose path order is not the model's.

Under a `roundtrip` directive, each of those is carried instead as a `//tdl:` comment directive, which a doc comment's text leaves out, its value a Go string literal:

- on a declaration's type: `source`, the declaration as TDL, and `at`, its index among the file's items when the files' order is not the model's;
- on the first file's package clause: `package` when the clause does not hold it, `doc` for the package's doc comment when the Go one does not read back, `import` for each import, and `item` for each top-level item with no Go type, as its index and the item as TDL. An alias, a unit, an instance, a target block, a foreign declaration, and a declaration Go cannot express are each one.

Which declarations carry a `source` is decided by reading the unannotated files back as import would, so a declaration is carried whole exactly when its Go reads back otherwise.
A model with nothing in a file still has its package's annotations, in `tdl.go`.

The normal form for comparing packages is each file through `go/format`, keeping doc comments only.

## Formatting

The backend runs its output through `go/format` in process, so it depends on neither `gofmt` nor `Response.post`, which nothing reads.
If formatting fails, the unformatted source is returned with an error-severity diagnostic.
