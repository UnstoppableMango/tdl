# Nix

`nix/` holds the packaging:

- `cmd.nix`: the CLI. `meta.mainProgram` is what `lib.getExe` reads, since the package installs fifteen binaries.
  It also installs bat's syntax as `share/bat/syntaxes/tdl.sublime-syntax`.
- `vscode-extension.nix`: the editor extension (see [VS Code](grammar.md#vs-code)).
- `gmk.nix`: `tdl-gmk`, `cmd/tdl-gmk` built as the GNU make loadable object `lib/tdl.so`, with gnumake's header for cgo; `checks.gmk` loads it into make.
- `portable.nix`: `tdl-portable`, `tdl` and `tdl-gen-typescript` for linux and darwin, amd64 and arm64, cross-compiled with `CGO_ENABLED=0` into `<goos>-<goarch>/`, for release assets and npm (#1385).
  The linux binaries are static; the darwin ones link only `libSystem`, as every Go binary on darwin does.
  `packages.default` links against the store's glibc loader, so it does not run outside Nix.
  `dontFixup` keeps the build platform's `strip` off the darwin/arm64 binaries, which the Go linker signs.
  `checks.tdl-portable` checks each binary's format and static linking, allows only the store paths nixpkgs' Go patches in as zoneinfo and `/etc/protocols` fallbacks, and runs the binary for the checking system over the smoke model.
- `npm.nix`: `packages.npm`, the six npm packages packed as tarballs: `@unmango/tdl` from `npm/tdl/`, `@unmango/tdl-typescript`, the content mapper's manifest, from `npm/tdl-typescript/`, and one `@unmango/tdl-<os>-<cpu>` per `tdl-portable` target, written with `jq` around its two binaries, with `os` and `cpu` set, which `@unmango/tdl` names in `optionalDependencies` at the flake's version.
  The build fails if either `package.json`'s version, which release-please bumps, is not the flake's.
  `npm/tdl/tdl.js` is the `bin`: it finds the platform package through `lib/resolve.js`, exported as `@unmango/tdl/resolve` for `@unmango/tdl-typescript`, and runs its binary, passing through arguments, stdio, and the exit status. No package has an install script.
  The shim sits beside `package.json` because `.gitignore` ignores every `bin/`.
  `checks.npm-tdl` installs `@unmango/tdl`, the checking system's platform package, and `@unmango/tdl-typescript` offline with `--ignore-scripts`, then runs `tdl version`, `tdl gen` over the smoke model, and `tdl-gen-typescript` through the resolver, and reads the mapper command from `@unmango/tdl-typescript`.
- `release.nix`: `packages.release`, the release assets: `tdl_<version>_<os>_<arch>.tar.gz` per `tdl-portable` target, each holding both binaries, `LICENSE`, `README.md`, and `completions/` written by the build platform's `tdl`, plus `SHA256SUMS`.
  Members are sorted, owned by root, dated `SOURCE_DATE_EPOCH`, and gzipped without a name, so two builds give the same `SHA256SUMS`.
  `checks.tdl-release` holds every tarball to the same file list, verifies the sums, and runs the checking system's `tdl`.
- `dotnet.nix`: the .NET plugins, built with `buildDotnetModule`, which runs their tests (see [.NET SDK](sdks.md#net-sdk)): `tdl-gen-csharp`, compiled with Native AOT and linked by `stdenv.cc`, and the SDK's debug port, a self-contained single file.
  The same csproj packs as the nuget.org tool in `dotnet-tool.nix`.
  On darwin the AOT link names the Swift runtime and ICU, so `swift` and `darwin.ICU` are build inputs there, as in nixpkgs' own dotnet `aot` test.
- `dotnet-tool.nix`: `tdl-gen-csharp-tool`, `Tdl.Gen.CSharp.<version>.nupkg`, the same project packed as a framework-dependent .NET tool for nuget.org (#1492), with no runtime and `-p:PublishAot=false`, since AOT pins the runtime patch the package asks for.
  `checks.dotnet-tool` installs it with `dotnet tool install --tool-path` from its folder alone, with no network, and holds what `tdl gen` writes for the smoke model through the tool's shim, and every diagnostic, to what the AOT plugin writes.
- `rust.nix`: `tdl-rust`, the Rust workspace built with `rustPlatform` from `rust/Cargo.lock`, which runs its tests (see [Rust SDK](sdks.md#rust-sdk)); it installs only `tdl-gen-rust_debug`, which `cmd.nix` does not ship.
- `js.nix`: `tdl-js`, the JS workspace built with `buildNpmPackage` and `importNpmLock` from `js/package-lock.json`, which runs its tests under node (see [JS SDK](sdks.md#js-sdk)); it installs only `tdl-gen-js_debug`, a node script `cmd.nix` does not ship.
- `protoc-gen-buffa.nix`: buffa's protoc plugin, which nixpkgs lacks, fetched from crates.io by hash, for `command make generate` and `checks.generate`.
- `tree-sitter-tdl.nix`: the grammar's Python binding in `tree-sitter/bindings/python/`, for graphify (see `docs/architecture/graphify.md`); the overlay adds it to every Python package set, and `checks.tree-sitter-tdl` runs its tests, which parse the prelude.
  The binding, `pyproject.toml`, and `setup.py` are `tree-sitter init` output with the highlights query exposed, the tests extended, and `scanner.c` listed with the sources so a source distribution carries it.
- `zod.nix`: the `zod` npm package, fetched by hash, for `checks.gen-zod` and the devShell's `NODE_PATH`.
- `demo.nix`: renders `docs/demo/demo.tape`, the README's GIF, with VHS. It is a package and not a check, since it runs a browser.
  The tape's `cat`, from `docs/demo/demo.bash`, highlights everything with bat, TDL through the syntax the `tdl` package installs.
- `hercules.nix`: Hercules CI's jobs, on `herculesCI.ciSystems` alone (x86_64-linux, from the `systems` input).
  `onPush.default` builds the checks and every package but `demo`, and no devShell.
  `onSchedule.nightly`, at 04:00 UTC, builds `demo` and every devShell, so vhs and the editors stay out of a pull request's fetch, and a Hydra failure in them blocks nothing; its result shows only on Hercules.
  It declares `flake.herculesCI` as an attribute set, since the `systems` input's module sets `ciSystems` in it and flake-parts refuses two raw definitions of one output.
- `overlay.nix`: names both packages and composes gomod2nix's overlay, so a consumer adding it also gets `buildGoApplication` and `mkGoEnv`.
- `hm-module.nix`: the home-manager module.
- `flake-module.nix`: the flake-parts module a consuming project imports.
- `default.nix`: the flake-parts module `flake.nix` imports; exports the overlay, the modules, and the packages.

`flake.nix` holds the inputs, the devShell, the treefmt configuration, and `version`.
`perSystem` imports nixpkgs with the overlay and reads `packages.default` and `packages.vscode-tdl` back out of it, so `nix build .#` takes the same path a consumer does.

`hm-module.nix` declares `programs.tdl`.
`enable` adds `package` to `home.packages`; `vscode.enable` (default `programs.vscode.enable`) adds `vscode.package` to the profiles in `vscode.profiles` (default `[ "default" ]`); `bat.enable` (default `programs.bat.enable`) adds the package's syntax to `programs.bat.syntaxes`.
Both packages default to `pkgs.tdl` and `pkgs.vscode-tdl` through `mkPackageOption`, so the module needs no flake input.
It is exported as `homeModules.default` and `homeManagerModules.default`, each with a `tdl` alias.
`checks.hm-module` evaluates a minimal configuration and asserts both packages land where the options say; it uses `allowUnfree` because `programs.vscode.enable` evaluates the editor.

`flake-module.nix` declares `perSystem.tdl`.
`enable` defaults to `src != null`, so setting `src` turns the module on and `enable = false` turns it off.
It defines `devShells.tdl` for `inputsFrom`, which under `make.enable` exports `TDL_GMK` from its `shellHook`, since `inputsFrom` drops a shell's variables, and carries nixpkgs' `gnumake`, plus one check per property a model should hold: `tdl-check` parses and lowers, `tdl-fmt` runs `tdl fmt --check`, and `tdl-gen` runs `tdl gen --verify`.
Each check is one invocation over all files.
`files` is a list of strings relative to one `src`, because a `path` is copied into the store alone and an `include` would stop resolving.
`gen.files` is separate and empty by default, because `tdl gen` fails on a file with no target block.
It is exported as `flakeModules.default` with a `tdl` alias.
`checks.flake-module` evaluates a consumer flake against `testdata/conformance/entity` that sets `src` but not `enable` and builds the result, without `tdl-gen`, which needs generated output on disk; it also asserts that `enable = false`, or setting nothing, defines no check or shell.
It also asserts that `post.packages`, the commands `tdl.toml`'s `[post]` table names, reach the `PATH` of `tdl-gen` and `devShells.tdl`.
