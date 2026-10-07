# Implementing the Java backend

An implementation plan for [java-backend.md](java-backend.md).
Phases are ordered by dependency, and each states what makes it done.

No phase is done.

## Scope

This plan builds the `java` backend.
It speaks the protocol in [plugins.md](plugins.md) unchanged.

It also moves the class plan, the JSON wire convention, and the `matches` and `length` rules out of `backend/internal/dotnet` into `backend/internal/emit`, as [java-backend.md](java-backend.md#overlap-with-other-targets) says.
If `backend/internal/dotnet` does not exist yet when a phase needs one of them, that phase writes it in `emit` directly, and [csharp-backend-plan.md](csharp-backend-plan.md) imports it from there.

## Layout

```text
backend/java/        # package java, name "java"
cmd/tdl-gen-java/    # the same value served as a subprocess
```

## Testing

The unit tests build `*ir.Model` values with `backend/internal/irtest` and assert on the generated text.
No Go library type checks Java, so the tests also run `javac --release 21 -Xlint:all -Werror` over every response when `javac` is on `PATH`, against the JSpecify and `jackson-annotations` jars.
`checks.gen-java` always does, over `testdata/gen/smoke`, with `jdk21` from nixpkgs and the jars fetched by hash.

`TestConstructionRefuses` compiles the response with a `Main` class that constructs invalid values and runs it with `java`, since a constructor is shown to refuse a value only by running it.
It skips under `-short` and when `java` is not on `PATH`.

`TestHostsAgree` in `internal/gen` holds the in-process and subprocess hosts to byte-identical output.

## Phase 1: records, enums, and the type mapping

Records in all three struct kinds, fieldless enums, fielded enums as sealed interfaces with nested records, newtypes as records holding their parent, aliases expanded, and the primitive and collection mapping with unsigned integers as their bits.
Absence through `Optional` and `@Nullable`, `package-info.java` with `@NullMarked`, `nullness`, and the constructor's null checks and collection copies.
Equality, `hashCode`, and `toString` for a record holding `bytes`.
The file header, `@Generated`, `package`, `name`, names, keyword escaping, qualified JDK names, and collisions.
Javadoc and `@Deprecated`.
The backend is registered in `internal/gen/registry.go`, has a row in the `shipped` table in `internal/gen/hosts_test.go`, is served by `cmd/tdl-gen-java`, and is listed in `nix/cmd.nix`.
`testdata/gen/smoke/source.tdl` gains `target java for smoke { out("java") }`, `nix/default.nix` gains `checks.gen-java`, and AGENTS.md gains a bullet under Backends.

Done when the smoke model compiles with `-Werror`, a `switch` over `Payment` with no `default` compiles and one missing `Cash` does not, two orders holding equal receipts compare equal, and both hosts return byte-identical files.

## Phase 2: identity, defaults, foreign types, and annotations

`key` as a method and a nested `Key` record, the staged `builder` with defaults as optional steps, `foreign` on a declaration, a primitive, or an extern, an extern resolved through its dependency's `java` block, and `annotation`.

Done when a two-package model generated into one `out` compiles, `bytes => foreign("com.google.protobuf", "ByteString")` compiles against protobuf-java and drops the generated `equals`, and a builder missing a required step does not compile.

## Phase 3: generics

Type parameters, boxing of primitive arguments, `requires` as a bound, and the higher-kind, unit-kind, and parameter-name warnings.

Done when `Page<T>` and a generic enum compile, and `Envelope<Long>` fails to compile in a test where `Envelope<T> requires Auditable<T>`.

## Phase 4: classes

The class plan moves to `backend/internal/emit`, gaining the associated-type row, and `backend/internal/dotnet` reads it from there if it exists.
Interfaces with accessor methods, `implements` clauses, a fielded enum's interface extending a class, associated types as interface parameters, and `Projection<To>` for a class with a dependency.

Done when every class in the conformance corpus generates or warns and compiles, an associated type binding compiles, and the shared plan's tests cover each row for every target reading it.

## Phase 5: constraints

Each standard constraint checked in the constructor of the type it is written on, every violation collected before throwing, a newtype checking only its own, and `violations` on a constrained newtype.
`matches` with the RE2 pre-check and `length` by code points, both moved to `backend/internal/emit`.

Done when each standard constraint is refused under `TestConstructionRefuses`, a string of four emoji passes `length(4)`, a `WorkEmail` built from a valid `Email` checks only its own pattern, and a pattern RE2 refuses warns at generation time.

## Phase 6: serialization

`json("jackson")` and `discriminant`: renamed properties and constants, polymorphic fielded enums, newtypes as bare values, absent optionals, required nullables, unsigned converters, and the defaulting creator.
The wire convention moves to `backend/internal/emit`, where the C# and F# serializers read it too.

Done when a test serializes a value of every smoke declaration with Jackson 3, validates the output against the `jsonschema` backend's schema for the smoke model, and deserializes it back to an equal value, and an invalid payload's error names the path to the field.

## Phase 7: units

Java can carry a unit only as a phantom type parameter, so the decision is the one [haskell-backend-plan.md](haskell-backend-plan.md) phase 6 weighs, with less to choose from.

- **Phantom tags.** Each base and named unit is an uninstantiable final class, and `decimal<kg>` is a record `Quantity<Kg>` per base type. A `decimal<kg>` cannot be passed as a `decimal<N>`, and a unit written as an expression has no name to tag with, as `Unit.decl` being unset says.
- **Dropped.** A unit-typed field is its base type with a warning, as phase 1 does.

Done when a unit-typed field either generates a type that keeps its unit or warns that it was dropped, and the declaration holding it is emitted in both cases.

## Phase 8: shipped profiles

`prelude/std/java.tdl` with `jackson`, setting `json("jackson")`.
This waits on phase 3 of [profiles-plan.md](profiles-plan.md).

Done when the smoke check also runs once with the profile applied, and the output compiles against `jackson-annotations`.

## Decisions deferred

- **Build files.** A `pom.xml` or Gradle directive, decided together with C#'s and F#'s project files.
- **Reverse.** A Go reader of class files, which [java-backend.md](java-backend.md#reverse) sketches, is its own plan under [reverse-plan.md](reverse-plan.md).
- **A JVM layer.** `backend/internal/jvm` is created by the second JVM target, not this one.
