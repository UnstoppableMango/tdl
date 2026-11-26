{
  lib,
  runCommand,
  unzip,
  go,
  tdl-portable,
  tdl-release,
}:
let
  inherit (tdl-release) version;

  # The tarball this system can run.
  native = "tdl_${version}_${go.GOOS}_${go.GOARCH}.tar.gz";

  # What every tarball holds, sorted as tar lists it, and every zip, sorted
  # as zipinfo lists it.
  zipped = [
    "LICENSE"
    "README.md"
    "completions/tdl.bash"
    "completions/tdl.fish"
    "completions/tdl.ps1"
    "completions/tdl.zsh"
    "tdl-gen-mermaid.exe"
    "tdl-gen-typescript.exe"
    "tdl.exe"
  ];
  want = [
    "LICENSE"
    "README.md"
    "completions/"
    "completions/tdl.bash"
    "completions/tdl.fish"
    "completions/tdl.zsh"
    "tdl"
    "tdl-gen-mermaid"
    "tdl-gen-typescript"
  ];
in
# Holds the release assets to what a release promises: one tarball per linux
# and darwin tdl-portable target and one zip per windows target, each kind
# holding the same files, SHA256SUMS naming each, and the tarball for this
# system unpacking to a tdl that runs and completes.
runCommand "tdl-release-check" { nativeBuildInputs = [ unzip ]; } ''
  fail() { echo "$*" >&2; exit 1; }
  cd ${tdl-release}

  sha256sum -c SHA256SUMS
  [ "$(wc -l < SHA256SUMS)" = ${toString (lib.length tdl-portable.targets)} ] \
    || fail "SHA256SUMS does not name one archive per target"

  for target in ${lib.escapeShellArgs tdl-portable.targets}; do
    if [[ $target == windows/* ]]; then
      archive=tdl_${version}_''${target/\//_}.zip
      [ -f "$archive" ] || fail "$archive is missing"
      zipinfo -1 "$archive" | LC_ALL=C sort > $TMPDIR/have
      diff -u ${builtins.toFile "zipped" (lib.concatLines zipped)} $TMPDIR/have \
        || fail "$archive does not hold what every zip holds"
      continue
    fi
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
