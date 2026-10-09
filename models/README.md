# Models

TDL descriptions of other projects' schemas, with the code each generates committed beside it.
Unlike `examples/`, these are meant to be imported: a Go program uses `github.com/unstoppablemango/tdl/models/mdast`, and TypeScript reads `mdast/ts/mdast.ts`.

| Model | Describes | Go package | TypeScript |
| --- | --- | --- | --- |
| `unist/unist.tdl` | [unist](https://github.com/syntax-tree/unist), the node shape every unified tree shares | `models/unist` | `unist/ts/unist.ts` |
| `mdast/mdast.tdl` | [mdast](https://github.com/syntax-tree/mdast), markdown as remark parses it | `models/mdast` | `mdast/ts/mdast.ts` |
| `hast/hast.tdl` | [hast](https://github.com/syntax-tree/hast), HTML as rehype parses it | `models/hast` | `hast/ts/hast.ts` |

## Regenerating

```shell
tdl gen models/unist/unist.tdl models/mdast/mdast.tdl models/hast/hast.tdl
```

`nix flake check` runs `tdl check`, `tdl fmt --check`, and `tdl gen --verify` over every model through `nix/flake-module.nix`, the way a consuming project would.

## Checking against DefinitelyTyped

`nix/checks/models-types/check.ts` holds the generated TypeScript to `@types/unist`, `@types/mdast`, and `@types/hast`, and `checks.models-types` runs it.
Every tree DefinitelyTyped accepts must be one the generated types accept, and each generated node must be one DefinitelyTyped accepts, apart from what a model knowingly loosens.

```shell
cd nix/checks/models-types && npm ci && npm run check
```

## What the models loosen

- **Children.** mdast and hast narrow what each parent holds, and TDL cannot name a subset of an enum's variants, so every `children` is a list of any node.
- **Heading depth.** `min(1) max(6)` is a constraint, so TypeScript sees `number` where DefinitelyTyped spells `1 | 2 | 3 | 4 | 5 | 6`.
- **hast property values.** hast allows a boolean, a number, a string, null, or a list of strings and numbers. TDL has no untagged union (see `docs/backlog.md`), so `PropertyValue` is a string.
- **Shared types.** `mdast.tdl` and `hast.tdl` repeat `unist.tdl`'s `Point`, `Position`, and `Data`, because the go and typescript backends do not generate a type from another package yet.

The Go types are not JSON wire types: the go backend writes no `json` tags and does not decode a fielded enum from its `type` property.
