# Renders docs/demo/demo.tape, the README's demo GIF, with VHS.
# `make demo` builds it and copies the GIF into docs/demo.
{
  runCommand,
  makeFontsConf,
  jetbrains-mono,
  bashInteractive,
  ncurses,
  bat,
  vhs,
  tdl,
}:
runCommand "tdl-demo"
  {
    # The tape's shell: stdenv's bash has no readline to expand the
    # prompt VHS sets, and ncurses provides `clear` and `tabs`.
    nativeBuildInputs = [
      bashInteractive
      ncurses
      bat
      vhs
      tdl
    ];
    FONTCONFIG_FILE = makeFontsConf { fontDirectories = [ jetbrains-mono ]; };
    COLORTERM = "truecolor";
  }
  ''
    export HOME=$TMPDIR

    # bat reads TDL through the syntax tdl installs, as programs.tdl.bat
    # gives it to a user's bat.
    export BAT_CONFIG_DIR=$PWD/bat BAT_CACHE_PATH=$PWD/bat/cache
    mkdir -p bat/syntaxes
    cp ${tdl}/share/bat/syntaxes/tdl.sublime-syntax bat/syntaxes/
    bat cache --build

    cp ${../docs/demo/shop.tdl} shop.tdl
    cp ${../docs/demo/demo.bash} demo.bash

    vhs ${../docs/demo/demo.tape}
    mkdir $out
    cp demo.gif $out/
  ''
