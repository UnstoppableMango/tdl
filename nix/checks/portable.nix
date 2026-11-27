{
  lib,
  runCommand,
  file,
  go,
  tdl-portable,
}:
let
  # The target this system can run.
  native = "${go.GOOS}-${go.GOARCH}";
in
# Holds tdl-portable to what the release assets and npm packages promise: every
# binary is the right format for its target, the Linux ones are static, none
# needs the store, and the one this system can run generates the smoke model.
runCommand "tdl-portable-check" { nativeBuildInputs = [ file ]; } ''
  fail() { echo "$*" >&2; exit 1; }

  for target in ${lib.escapeShellArgs tdl-portable.targets}; do
    dir=${tdl-portable}/''${target/\//-}
    exe= machine=
    [[ $target == windows/* ]] && exe=.exe
    for bin in tdl tdl-gen-typescript; do
      path=$dir/$bin$exe
      if [[ $target == windows/* ]]; then
        [ -f "$path" ] || fail "$path is missing"
      else
        [ -x "$path" ] || fail "$path is missing or not executable"
      fi
      desc=$(file -b "$path")
      echo "$target $bin: $desc"
      case $target in
        linux/amd64) want="ELF 64-bit LSB executable, x86-64" ;;
        linux/arm64) want="ELF 64-bit LSB executable, ARM aarch64" ;;
        darwin/amd64) want="Mach-O 64-bit x86_64 executable" ;;
        darwin/arm64) want="Mach-O 64-bit arm64 executable" ;;
        # file has spelled the machine both "(console) x86-64, for MS
        # Windows" and "for MS Windows 10.00 (console), x86-64", and arm64
        # both Aarch64 and ARM64.
        windows/amd64) want="PE32+ executable" machine="x86-64" ;;
        windows/arm64) want="PE32+ executable" machine="Aarch64|ARM64" ;;
      esac
      # darwin's bash refuses an empty pattern, so only a set machine is matched.
      [[ $desc == "$want"* && ( -z $machine || $desc =~ $machine ) ]] || fail "$path is not $want $machine"
      if [[ $target == linux/* && $desc != *"statically linked"* ]]; then
        fail "$path is not statically linked"
      fi
      # nixpkgs patches Go to look for zoneinfo and /etc/protocols and
      # /etc/services in the store first; outside Nix the open fails and Go
      # falls back to the system's files or its built-in table, so those two
      # are allowed and nothing else is.
      stray=$(tr '\0' '\n' < "$path" | grep -ao '/nix/store/[a-z0-9]\{32\}-[^/[:space:]]*' | sort -u | grep -v -e '-tzdata-' -e '-iana-etc-' || true)
      [ -z "$stray" ] || fail "$path names $stray"
    done
  done

  export PATH=${tdl-portable}/${native}:$PATH
  tdl version
  cp ${../../testdata/gen/smoke/source.tdl} source.tdl
  tdl gen --target typescript -o out source.tdl
  find out -name '*.ts' | grep -q . || fail "tdl gen wrote no TypeScript"
  touch $out
''
