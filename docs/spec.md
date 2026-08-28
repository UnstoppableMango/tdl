# TDL Language Specification — v0.1.0 (draft)

TDL (Type Description Language) describes data types: records, enums, primitives, and collections.
It is not a general-purpose programming language and has no expressions, control flow, or runtime semantics.
A TDL file only describes the shape of data.

This document is the canonical specification.
`github.com/unstoppablemango/tdl` owns the reference implementation.
The corpus at `testdata/conformance/` and `testdata/invalid/` in this repository is plain text, not Go code, so any independent implementation can check itself against the same fixtures.

This draft covers the M1 subset of the language: lexical structure, grammar, and the type system.
Annotation *semantics* (what a given namespace like `@go` or `@protobuf` means to a backend) are out of scope here, since no backends exist yet.
Annotations are defined syntactically and parsed, but uninterpreted.

## 1. Lexical structure

A TDL source file is UTF-8 text with the extension `.tdl`.

### 1.1 Comments

A `//` starts a line comment that runs to the end of the line.
There are no block comments.

### 1.2 Identifiers

```
identifier = letter { letter | digit }
letter     = "a"…"z" | "A"…"Z" | "_"
digit      = "0"…"9"
```

### 1.3 Keywords

The following identifiers are reserved and cannot be used as a type, field, enum, or enum-value name:

```
package  import  as  type  enum  union  true  false
```

`union` is reserved for a future generic sum-type declaration.
It is not implemented by the M1 parser.

Keywords also cannot currently be used as annotation argument names (`@go(package: "x")` is a syntax error — use a non-keyword name such as `pkg`).
This is a known M1 simplification, not a permanent restriction.

### 1.4 Literals

```
string_lit = '"' { unescaped_char | escape_sequence } '"'
escape_sequence = '\' ( 'n' | 't' | '"' | '\' )
int_lit    = digit { digit }
float_lit  = digit { digit } "." digit { digit }
bool_lit   = "true" | "false"
```

### 1.5 Punctuation

```
{ } ( ) [ ] < > : , = ? . @
```

## 2. Grammar

The formal grammar is in [`grammar.ebnf`](./grammar.ebnf).
This section walks through it with examples.

### 2.1 File structure

```tdl
package example.v1

import "common.tdl" as common

type User {
  id: string
  name: string?
  tags: list<string>
  metadata: map<string, string>
  address: common.Address?
  role: Role = "member"
}

enum Role {
  Admin = "admin"
  Member = "member"
  Guest = "guest"
}
```

A file has an optional `package` declaration (at most one, first in the file), followed by zero or more `import` declarations, followed by zero or more `type` and `enum` declarations in any order.

`import "path.tdl" as alias` is resolved relative to the importing file.
**In M1, imports are parsed but not resolved.**
`tdl check` validates a single file's syntax only.
It does not verify that an imported file exists or that a qualified reference like `common.Address` resolves to a real declaration.
Cross-file resolution is deferred to the `ir` package in a later milestone.

### 2.2 Type declarations

```
type Name {
  field1: Type
  field2: Type?
  field3: Type = default
}
```

Each field is `name: Type`, optionally suffixed with `?` to mark it optional, optionally followed by `= <literal>` to give it a default value.
Fields are separated by newlines.
There is no comma or semicolon between fields.

### 2.3 Primitive types

| Name      | Description                    |
|-----------|---------------------------------|
| `string`  | UTF-8 text                      |
| `bool`    | boolean                         |
| `int8`    | 8-bit signed integer            |
| `int16`   | 16-bit signed integer           |
| `int32`   | 32-bit signed integer           |
| `int64`   | 64-bit signed integer           |
| `uint8`   | 8-bit unsigned integer          |
| `uint16`  | 16-bit unsigned integer         |
| `uint32`  | 32-bit unsigned integer         |
| `uint64`  | 64-bit unsigned integer         |
| `float32` | 32-bit floating point           |
| `float64` | 64-bit floating point           |
| `bytes`   | arbitrary binary data           |

This set is deliberately small, chosen to map cleanly onto the primitive type systems of the initial target formats (JSON Schema, Go, TypeScript, Protobuf) with no ambiguous or lossy conversions.

### 2.4 Collections

- `list<T>` — an ordered sequence of `T`.
- `map<K, V>` — a mapping from `K` to `V`.
  `K` must be `string` or an integer primitive (`int8`…`uint64`).
  This matches both JSON object-key and Protobuf map-key constraints, so no target backend needs a fallback or error path for an unsupported key type.

### 2.5 References

A field or collection element may reference another declared type: either bare (`Address`, resolved within the same file/package) or qualified (`common.Address`, resolved through an import alias).

### 2.6 Enums

```
enum Name {
  Variant1 = "literal-or-int"
  Variant2
}
```

Each variant may carry an explicit literal (string or integer).
A variant with no explicit literal has no assigned value in M1.
A later milestone defines the default representation (likely the variant name as a string) once a backend needs one.

### 2.7 Annotations

```
annotation ::= '@' identifier [ '(' arg (',' arg)* ')' ]
arg        ::= identifier ':' literal
```

Annotations attach target-specific extension data to a `type`, `field`, or `enum`/enum-value declaration:

```tdl
@go(pkg: "example")
type User {
  @go(tag: "json:\"full_name,omitempty\"")
  @protobuf(number: 10)
  name: string
}
```

The `identifier` right after `@` is a **namespace**, conventionally matching a backend's name (`go`, `csharp`, `protobuf`, `jsonschema`, ...).
The core grammar places no restriction on which namespaces exist.
This is intentional.
A backend that doesn't recognize a namespace simply ignores it.
A backend that does recognize one interprets its arguments however it needs to (see the forthcoming `ir` package documentation for how backends consume annotations once they exist).

Multiple annotations may precede the same declaration, including two annotations with the same namespace.

### 2.8 Reserved but unimplemented

The `union` keyword and generic type-parameter syntax on `type` declarations are reserved by the grammar but rejected/unsupported by the M1 parser.
Reserving them now means their eventual addition is additive to the grammar, not a breaking change to existing `.tdl` source.

## 3. Conformance

A conforming implementation must:

1. Accept every file in `testdata/conformance/` without a syntax error.
2. Reject every file in `testdata/invalid/` with a syntax error.
3. Format via `tdl fmt` deterministically and idempotently: formatting already-canonical output must be a no-op.

Golden per-target output files (`*.golden`) will be added to the conformance corpus once backends exist, at which point conformance also requires matching that output exactly.
