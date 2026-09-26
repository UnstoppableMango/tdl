# The home-manager module: `programs.tdl` installs the CLI, and
# `programs.tdl.vscode` hands the extension to every VS Code-based editor
# that is enabled. Both packages are read out of `pkgs` rather than out of
# the flake that exports this file, so the module evaluates anywhere the
# overlay has been added and `nixpkgs.overlays` stays the one thing a
# consumer wires up.
{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.programs.tdl;

  # Every VS Code-based editor home-manager declares a module for. All six
  # come from one `mkVscodeModule`, so `profiles.<name>.extensions` means
  # the same thing in each, and one extension package serves them all.
  #
  # `or false` is for a home-manager predating a module: an editor it has
  # never heard of is one this is not installed into, rather than an
  # evaluation error.
  known = [
    "vscode"
    "vscodium"
    "cursor"
    "windsurf"
    "kiro"
    "antigravity"
  ];
  enabled = lib.filter (name: config.programs.${name}.enable or false) known;
in
{
  options.programs.tdl = {
    enable = lib.mkEnableOption "tdl, the type description language";

    package = lib.mkPackageOption pkgs "tdl" { };

    vscode = {
      enable = lib.mkEnableOption "the tdl VS Code extension" // {
        default = enabled != [ ];
        defaultText = lib.literalMD "whether any VS Code-based editor is enabled";
      };

      package = lib.mkPackageOption pkgs "vscode-tdl" { };

      editors = lib.mkOption {
        type = lib.types.listOf (lib.types.enum known);
        default = enabled;
        defaultText = lib.literalMD "every VS Code-based editor that is enabled";
        example = [ "vscodium" ];
        description = ''
          The VS Code-based editors the tdl extension is added to, each named
          by its home-manager module under `programs`.
        '';
      };

      profiles = lib.mkOption {
        type = lib.types.listOf lib.types.str;
        default = [ "default" ];
        example = [
          "default"
          "work"
        ];
        description = "Editor profiles the tdl extension is added to.";
      };
    };
  };

  config = lib.mkIf cfg.enable (
    lib.mkMerge (
      [
        { home.packages = [ cfg.package ]; }

        (lib.mkIf cfg.vscode.enable {
          # An extension handed to a disabled editor is dropped without a
          # word, so say so rather than leaving the user to find it in the
          # editor.
          assertions = [
            {
              assertion = cfg.vscode.editors != [ ];
              message =
                "programs.tdl.vscode.enable needs an editor: enable one of "
                + lib.concatMapStringsSep ", " (name: "programs.${name}") known
                + ", or name one in programs.tdl.vscode.editors.";
            }
          ]
          ++ map (name: {
            assertion = config.programs.${name}.enable or false;
            message = "programs.tdl.vscode.editors names ${name}, but programs.${name}.enable is false.";
          }) cfg.vscode.editors;
        })
      ]
      ++ map (
        name:
        lib.mkIf (cfg.vscode.enable && lib.elem name cfg.vscode.editors) {
          programs.${name}.profiles = lib.genAttrs cfg.vscode.profiles (_: {
            extensions = [ cfg.vscode.package ];
          });
        }
      ) known
    )
  );
}
