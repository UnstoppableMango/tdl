# Releases

`.github/workflows/release-please.yml` calls the reusable workflow in `unmango/actions` as the thecluster[bot] GitHub App (`vars.RELEASE_APP_CLIENT_ID`, `secrets.RELEASE_APP_PRIVATE_KEY`), so release PRs trigger CI and commits are signed.

release-please owns the version.
Never hand-edit `toolVersion` in `internal/cli/version.go`, `version` in `flake.nix`, `version` in `editors/vscode/package.json` and `package-lock.json`, `version` in `npm/tdl/package.json` and `npm/tdl-typescript/package.json`, `<Version>` in `dotnet/Tdl.Gen.CSharp/Tdl.Gen.CSharp.csproj`, the `version=` line in the README's release-asset example, or `CHANGELOG.md`.
The first two, the csproj line, and the README line carry an `x-release-please-version` annotation; the JSON files are listed in `release-please-config.json`.

When it reports a release, the `assets` job builds `packages.release` at the tag on a GitHub-hosted runner, attests every file with `actions/attest-build-provenance`, and uploads them to the release.
The `npm` job builds `packages.npm` at the tag and publishes the four platform packages, then `@unmango/tdl`, then `@unmango/tdl-typescript`, through npm's trusted publishing, which needs no stored token and writes provenance; each package's trusted publisher on npmjs.com names `release-please.yml`, and a version already published is skipped.
The `nuget` job builds `packages.tdl-gen-csharp-tool` at the tag and pushes it to nuget.org through trusted publishing: `NuGet/login` trades the job's OIDC token for a short-lived key as `vars.NUGET_USER`, whose policy on nuget.org names `release-please.yml`, and `--skip-duplicate` skips a version already pushed.

Two versions are not release-please's:

- `metadata.version` in `tree-sitter/tree-sitter.json`: `tree-sitter generate` copies it into `parser.c`, so a release bump would fail CI's regeneration diff. Bump it by hand with a regeneration.
- `specVersion`: tracks `docs/spec.md`. Never give it the annotation.

The warning that `version.txt` does not exist is harmless.
