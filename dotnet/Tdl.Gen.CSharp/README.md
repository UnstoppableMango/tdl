# Tdl.Gen.CSharp

The `csharp` backend for [TDL](https://github.com/UnstoppableMango/tdl), the Type Description Language.
`tdl gen` runs `tdl-gen-csharp` from `PATH` to write C# records, enums, and newtypes from a model's `csharp` target block.

Install it beside a `tdl` from npm, the release assets, or `go install` (the Nix package already ships the plugin):

```shell
dotnet tool install -g Tdl.Gen.CSharp
tdl gen --target csharp -o out model.tdl
```

Or install it into a project's own folder, since `tdl` finds plugins only on `PATH` and a local tool manifest's tools are not on it:

```shell
dotnet tool install --tool-path .tools Tdl.Gen.CSharp
PATH="$PWD/.tools:$PATH" tdl gen --target csharp -o out model.tdl
```

It needs .NET 10 or later, and its version matches the `tdl` it was released with.
[docs/backends/csharp.md](https://github.com/UnstoppableMango/tdl/blob/main/docs/backends/csharp.md) lists what it generates.
