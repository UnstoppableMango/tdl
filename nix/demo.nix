# Renders docs/demo/demo.tape, the README's demo GIF, with VHS.
# `make demo` builds it and copies the GIF into docs/demo.
{
  runCommandCC,
  linkFarm,
  writeText,
  makeFontsConf,
  jetbrains-mono,
  bashInteractive,
  ncurses,
  bat,
  tree-sitter,
  vhs,
  tdl,
}:
let
  # tree-sitter finds a grammar in a parser directory by its
  # tree-sitter-<name> folder.
  parsers = linkFarm "tdl-demo-parsers" [
    {
      name = "tree-sitter-tdl";
      path = ../tree-sitter;
    }
  ];

  treeSitterConfig = writeText "config.json" (
    builtins.toJSON {
      parser-directories = [ parsers ];
      theme = import ./themes/tree-sitter/catppuccin-mocha.nix;
    }
  );
in
runCommandCC "tdl-demo"
  {
    # The tape's shell: stdenv's bash has no readline to expand the
    # prompt VHS sets, and ncurses provides `clear` and `tabs`.
    nativeBuildInputs = [
      bashInteractive
      ncurses
      bat
      tree-sitter
      vhs
      tdl
    ];
    FONTCONFIG_FILE = makeFontsConf { fontDirectories = [ jetbrains-mono ]; };
    COLORTERM = "truecolor";
  }
  ''
    export HOME=$TMPDIR
    mkdir ts
    cp ${treeSitterConfig} ts/config.json
    export TREE_SITTER_DIR=$PWD/ts TREE_SITTER_LIBDIR=$PWD/ts/lib
    cp ${../docs/demo/shop.tdl} shop.tdl
    cp ${../docs/demo/demo.bash} demo.bash

    # Compile the grammar now, so the tape does not wait on the compiler.
    # runCommandCC provides the compiler tree-sitter calls.
    tree-sitter highlight shop.tdl > /dev/null

    vhs ${../docs/demo/demo.tape}
    mkdir $out
    cp demo.gif $out/
  ''
