# Renders docs/demo/demo.tape, the README's demo GIF, with VHS.
# `make demo` builds it and copies the GIF into docs/demo.
{
  lib,
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

  # Catppuccin Mocha, the tape's theme and the one bat uses for Go and
  # TypeScript, for each capture tree-sitter/queries/highlights.scm names.
  treeSitterConfig = writeText "config.json" (
    builtins.toJSON {
      parser-directories = [ parsers ];
      theme = {
        keyword = "#cba6f7";
        type = "#f9e2af";
        property = "#b4befe";
        module = "#fab387";
        string = "#a6e3a1";
        "string.regex" = "#f5c2e7";
        number = "#fab387";
        boolean = "#fab387";
        "constant.builtin" = "#fab387";
        constructor = "#94e2d5";
        "function.call" = "#89b4fa";
        attribute = "#f5c2e7";
        "variable.parameter" = "#eba0ac";
        operator = "#89dceb";
        "punctuation.bracket" = "#9399b2";
        "punctuation.delimiter" = "#9399b2";
        comment = {
          color = "#6c7086";
          italic = true;
        };
        "comment.documentation" = {
          color = "#6c7086";
          italic = true;
        };
      };
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
