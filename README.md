# tdl

TDL (Type Description Language) describes data types — records, enums, primitives, and collections — and compiles them into equivalent type definitions in other structured formats.
It is not a general-purpose programming language: no expressions, no control flow, no runtime.

This repository owns the canonical [language specification](docs/spec.md) and its reference implementation, written in Go.

## Status

Early and incomplete.
The lexer, parser, and `tdl check` / `tdl fmt` commands work for the M1 grammar subset: records, primitives, fields with optionality and defaults, `list`/`map` collections, type references, enums, and annotation syntax.
There is no semantic resolution (`ir`) and no code-generation backends yet — see [`docs/notes.md`](docs/notes.md) for the roadmap.

## Example

```tdl
package example.v1

import "common.tdl" as common

@go(pkg: "example")
type User {
  id: string

  @go(tag: "json:\"full_name,omitempty\"")
  name: string?

  tags: list<string>
  metadata: map<string, string>
  address: common.Address?

  @protobuf(number: 10)
  role: Role = "member"
}

enum Role {
  Admin = "admin"
  Member = "member"
  Guest = "guest"
}
```

## Usage

```shell
tdl check ./types.tdl   # parse and report syntax errors
tdl fmt ./types.tdl      # print canonical formatting; -w to write in place
tdl version               # tool and spec versions
```

## Development

Requires Go 1.24+.

```shell
go build ./...
go test ./...
```

A `flake.nix` (package + devShell) is planned for a later milestone.

## Design philosophy

- A small, strict grammar with its own hand-written lexer and parser — no parser generator, no YAML/JSON stand-in syntax.
- One canonical Go implementation; the spec and the `testdata/conformance` / `testdata/invalid` corpora are the contract any other implementation would need to satisfy.
- A generic `@namespace(key: value, ...)` annotation mechanism carries target-specific intrinsics (Go struct tags, C# attributes, protobuf field numbers, ...) without the core language needing to know about any particular target ahead of time.
- One static binary, no subprocess plugins or IPC.
  See [`docs/notes.md`](docs/notes.md) for why: an earlier version of this repository tried a multi-process, multi-language architecture and it didn't work.
