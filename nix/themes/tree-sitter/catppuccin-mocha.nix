# Catppuccin Mocha as a tree-sitter CLI theme, for each capture
# tree-sitter/queries/highlights.scm names. nix/demo.nix uses it to match
# the tape's terminal theme and the one bat uses for Go and TypeScript.
let
  italic = color: {
    inherit color;
    italic = true;
  };
in
{
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
  comment = italic "#6c7086";
  "comment.documentation" = italic "#6c7086";
}
