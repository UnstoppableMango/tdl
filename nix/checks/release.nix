{
  lib,
  runCommand,
  go,
  tdl-portable,
  tdl-release,
}:
let
  inherit (tdl-release) version;

  # The tarball this system can run.
  native = "tdl_${version}_${go.GOOS}_${go.GOARCH}.tar.gz";

  # What every tarball holds, sorted as tar lists it.
  want = [
    "LICENSE"
    "README.md"
    "completions/"
    "completions/tdl.bash"
    "completions/tdl.fish"
    "completions/tdl.zsh"
    "tdl"
    "tdl-gen-typescript"
  ];
in
# Holds the release assets to what a release promises: one tarball per
# tdl-portable target holding the same files, SHA256SUMS naming each, and the
# tarball for this system unpacking to a tdl that runs and completes.
runCommand "tdl-release-check" { } ''
  fail() { echo "$*" >&2; exit 1; }
  cd ${tdl-release}

  sha256sum -c SHA256SUMS
  [ "$(wc -l < SHA256SUMS)" = ${toString (lib.length tdl-portable.targets)} ] \
    || fail "SHA256SUMS does not name one tarball per target"

  for target in ${lib.escapeShellArgs tdl-portable.targets}; do
    tarball=tdl_${version}_''${target/\//_}.tar.gz
    [ -f "$tarball" ] || fail "$tarball is missing"
    tar -tzf "$tarball" | sort > $TMPDIR/have
    diff -u ${builtins.toFile "want" (lib.concatLines want)} $TMPDIR/have \
      || fail "$tarball does not hold what every tarball holds"
  done

  # Members by name, as the README extracts tdl.
  mkdir $TMPDIR/unpacked && tar -xzf ${native} -C $TMPDIR/unpacked tdl completions
  cd $TMPDIR/unpacked
  ./tdl version
  grep -q __start_tdl completions/tdl.bash || fail "completions/tdl.bash is not tdl's"
  grep -q '#compdef tdl' completions/tdl.zsh || fail "completions/tdl.zsh is not tdl's"
  grep -q 'complete -c tdl' completions/tdl.fish || fail "completions/tdl.fish is not tdl's"
  touch $out
''
