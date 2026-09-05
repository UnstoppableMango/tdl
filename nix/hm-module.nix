# The home-manager module: `programs.tdl` installs the CLI, and
# `programs.tdl.vscode` hands the extension to the VS Code module. Both
# packages are read out of `pkgs` rather than out of the flake that exports
# this file, so the module evaluates anywhere the overlay has been added and
# `nixpkgs.overlays` stays the one thing a consumer wires up.
{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.programs.tdl;
in
{
  options.programs.tdl = {
    enable = lib.mkEnableOption "tdl, the type description language";

    package = lib.mkPackageOption pkgs "tdl" { };

    vscode = {
      enable = lib.mkEnableOption "the tdl VS Code extension" // {
        default = config.programs.vscode.enable;
        defaultText = lib.literalExpression "config.programs.vscode.enable";
      };

      package = lib.mkPackageOption pkgs "vscode-tdl" { };

      profiles = lib.mkOption {
        type = lib.types.listOf lib.types.str;
        default = [ "default" ];
        example = [
          "default"
          "work"
        ];
        description = "VS Code profiles the tdl extension is added to.";
      };
    };
  };

  config = lib.mkIf cfg.enable (
    lib.mkMerge [
      { home.packages = [ cfg.package ]; }

      (lib.mkIf cfg.vscode.enable {
        # An extension handed to a disabled VS Code is dropped without a word,
        # so say so rather than leaving the user to find it in the editor.
        assertions = [
          {
            assertion = config.programs.vscode.enable;
            message = "programs.tdl.vscode.enable requires programs.vscode.enable.";
          }
        ];

        programs.vscode.profiles = lib.genAttrs cfg.vscode.profiles (_: {
          extensions = [ cfg.vscode.package ];
        });
      })
    ]
  );
}
