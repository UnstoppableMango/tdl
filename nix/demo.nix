# Renders docs/demo/demo.tape, the README's demo GIF, with VHS.
# `make demo` builds it and copies the GIF into docs/demo.
{
  runCommand,
  makeFontsConf,
  jetbrains-mono,
  bashInteractive,
  ncurses,
  vhs,
  tdl,
}:
runCommand "tdl-demo"
  {
    # The tape's shell: stdenv's bash has no readline to expand the
    # prompt VHS sets, and ncurses provides `clear`.
    nativeBuildInputs = [
      bashInteractive
      ncurses
      vhs
      tdl
    ];
    FONTCONFIG_FILE = makeFontsConf { fontDirectories = [ jetbrains-mono ]; };
  }
  ''
    export HOME=$TMPDIR
    cp ${../docs/demo/shop.tdl} shop.tdl
    vhs ${../docs/demo/demo.tape}
    mkdir $out
    cp demo.gif $out/
  ''
