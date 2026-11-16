# A Go command over the whole module in the sandbox, with the vendored
# modules gomod2nix builds from nix/gomod2nix.toml and no network: what
# checks.go-test, checks.go-cover, and checks.golangci-lint share. script
# runs from the module root with HOME, the Go cache, and GOFLAGS set, and
# writes $out. Its gotest runs go test and, on a failure, ends the log with
# the failing tests, since Nix shows only a failed build's last lines.
{
  buildGoApplication,
  go,
  lib,
  version,
}:
{
  name,
  script,
  nativeBuildInputs ? [ ],
  env ? { },
  CGO_ENABLED ? 0,
}:
buildGoApplication {
  pname = "tdl-${name}";
  inherit
    version
    go
    nativeBuildInputs
    CGO_ENABLED
    ;

  src = lib.cleanSource ../..;
  modules = ../gomod2nix.toml;

  inherit env;

  # The config hook sets up vendor/ and GOFLAGS; nothing is built or
  # installed but what script writes.
  dontBuild = true;
  doCheck = true;
  checkPhase = ''
    runHook preCheck
    export HOME=$TMPDIR
    set -o pipefail
    gotest() {
      if ! go test "$@" 2>&1 | tee "$TMPDIR/go-test.log"; then
        echo "failing tests:"
        grep -E -e '--- FAIL|^panic:' "$TMPDIR/go-test.log" | tail -n 20
        return 1
      fi
    }
    ${script}
    runHook postCheck
  '';
  dontInstall = true;
  dontFixup = true;
  allowGoReference = true;
}
