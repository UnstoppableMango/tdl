{
  lib,
  runCommand,
  stdenv,
  file,
  patchelf,
  go,
  tdl,
  tdl-gen-mermaid,
}:
let
  inherit (tdl-gen-mermaid) portable;
  # The target this system can run.
  native = "${go.GOOS}-${go.GOARCH}";
  want = ../../testdata/mermaid/corpus/smoke/want/smoke.mmd;
in
# Holds tdl-gen-mermaid to what pkgs.tdl, the release tarballs, and the npm
# packages promise: tdl finds the one pkgs.tdl ships and it draws the smoke
# model as the tests expect, every portable executable is the right format,
# and the one this system can run draws the same.
runCommand "tdl-gen-mermaid-check"
  {
    nativeBuildInputs = [
      tdl
      file
      patchelf
    ];
  }
  ''
    fail() { echo "$*" >&2; exit 1; }
    cp ${../../testdata/gen/smoke/source.tdl} source.tdl

    [ "$(readlink -f "$(command -v tdl-gen-mermaid)")" = "$(readlink -f ${tdl}/bin/tdl-gen-mermaid)" ] \
      || fail "pkgs.tdl does not put tdl-gen-mermaid on PATH"
    tdl gen --target mermaid -o out source.tdl
    diff ${want} out/smoke.mmd

    for target in ${lib.escapeShellArgs portable.targets}; do
      path=${portable}/''${target/\//-}/tdl-gen-mermaid
      [ -x "$path" ] || fail "$path is missing"
      desc=$(file -b "$path")
      echo "$target: $desc"
      case $target in
        linux/amd64) want="ELF 64-bit LSB executable, x86-64" ;;
        linux/arm64) want="ELF 64-bit LSB executable, ARM aarch64" ;;
        darwin/amd64) want="Mach-O 64-bit x86_64 executable" ;;
        darwin/arm64) want="Mach-O 64-bit arm64 executable" ;;
      esac
      case $desc in "$want"*) ;; *) fail "$target is not $want" ;; esac
    done

    # The portable one, first on PATH. On linux it names the system's glibc
    # loader, which the sandbox lacks, so a copy names the store's; patchelf
    # leaves the rest alone, and the bundle with it.
    mkdir bin
    cp ${portable}/${native}/tdl-gen-mermaid bin/
    chmod u+w bin/tdl-gen-mermaid
    ${lib.optionalString stdenv.hostPlatform.isLinux ''
      patchelf --set-interpreter ${stdenv.cc.bintools.dynamicLinker} bin/tdl-gen-mermaid
    ''}
    PATH=$PWD/bin:$PATH tdl gen --target mermaid -o portable source.tdl
    diff ${want} portable/smoke.mmd
    touch $out
  ''
