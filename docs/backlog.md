# Backlog

Work that is wanted but not scheduled.
Anything with a plan lives in [design/](design/) instead.

## tdl fmt as a treefmt formatter

`nix fmt` excludes `.tdl` files.
Adding `tdl fmt` as a custom formatter in `flake.nix` needs the locally built binary, so formatting the repository would depend on building it.

## Anonymous union types

A field wants `A | B` without a declaration; `enum` requires every alternative to be named and declared.
`TypeRef` already admits `T | null` as sugar for `Nullable<T>`, so the general case opens up the right-hand side.

The open question is arity: `A | B | C` wants a variadic `Union`, and kinds have nothing variadic, while nesting into `Either` makes association and order significant.
The answer decides whether this needs a kind-system change, and with it what `ir` carries and how the recursion rules treat a union.

## An exponent on a parenthesized unit term

`docs/grammar.ebnf` has `UnitTerm = identifier [ "^" int_lit ] | "(" UnitExpr ")" .`, but `parser.parseUnitTerm` accepts `(kg*m)^2` and lowers it to `kg^2*m^2`.
The parser is right; the fix is `UnitTerm = ( identifier | "(" UnitExpr ")" ) [ "^" int_lit ] .`, a sentence in the spec's units section, and a regenerated tree-sitter grammar.

## JetBrains

A plugin over the platform's PSI model, narrowed by its LSP API.
The other editors are planned in [design/editors-plan.md](design/editors-plan.md) and [design/lsp-editors-plan.md](design/lsp-editors-plan.md).

## MCP server

Lets an agent query a resolved model: an entity's fields, what satisfies a class, what a target block maps.
`tdl ir --format json` already emits the model, so a first version is a thin wrapper; which tools to offer beyond that is the open question.

## Importing a model with imports

A reverse backend is given only the target's files, so a model that imports another TDL file does not come back: its imports print, and lowering them needs the files they name.
`ImportRequest` could carry the imported TDL, or the host could lower the printed file itself.

## Importing protobuf services and nested types

`tdl import --from protobuf` warns and skips a service written by hand, and hoists a nested message or enum to the top level.
A service wants the `rpc` and `stream` primitives made up, as a oneof's enum is; a nested type wants a TDL spelling for nesting, or a `name` directive Generate reads.

## Importing what Thrift has and TDL does not

`tdl import --from thrift` warns and skips a const, a service, a union that is not a fielded enum's shape, and an `include`, which fails the import.
An exception is read as a struct, and `required`, a default, and annotations other than `deprecated` are dropped.
A union of primitives could be an enum whose variants hold one field each, if Generate wrote that shape back; defaults are a TDL field's `= value`, if Generate wrote them.

## Importing what GraphQL has and TDL does not

`tdl import --from graphql` warns and skips an interface, an input type, a custom scalar it does not know, a union that is not a fielded enum's shape, a type extension, and a directive other than `@deprecated`.
A field's arguments are dropped, so a `Query` type reads as a struct, and the schema's root operation types are not kept.
An interface could be a class, as [reverse.md](design/reverse.md) suggests, and a field with arguments wants TDL surface for operations, which Smithy and protobuf services want too.

## Importing what Go has and TDL does not

`tdl import --from go` warns and skips a func, a var, a method Generate does not write, an embedded field, an array, a channel, and an interface without a marker, and reads `int` and the narrow integers as wider primitives.
An embedded struct could be an `include`, if Generate embedded a mixin rather than copying its fields; an interface with methods wants TDL surface for operations, as GraphQL's and Smithy's do.
