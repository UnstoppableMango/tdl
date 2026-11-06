# A Homebrew formula for tdl-gen-csharp

The Homebrew tap (#1391) carries `tdl` alone.
It will not carry `tdl-gen-csharp`, the .NET plugin serving the `csharp` target, as a formula of its own or inside `tdl`'s.

## Why this is out of scope

Every Go backend is compiled into `tdl`, and the release assets the formula installs are cross-compiled from one Linux builder.
`tdl-gen-csharp` is a Native AOT binary, and .NET's AOT compiler does not cross-compile between operating systems, so a darwin build needs a darwin builder, and the darwin link names the Swift runtime and ICU.
A formula would add a second build pipeline for one target.

The users who want C# have better routes:

- In a .NET build, the C# source generator (#1199) runs the `csharp` backend in process from `Tdl.Gen.CSharp`, through `tdl request`, and needs only `tdl` on `PATH`, which the `tdl` formula provides.
- Outside one, `tdl-gen-csharp` can ship as a framework-dependent `dotnet tool` on NuGet: one package for every platform, no AOT, installed with `dotnet tool install`, where a .NET user already looks.

Nix keeps shipping the AOT binary beside the Go binaries in `nix/cmd.nix`.

## Prior requests

- #1430: "A tdl-csharp Homebrew formula for tdl-gen-csharp"
