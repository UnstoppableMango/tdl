# The flake-parts module holding how tdl is built: the overlay the flake
# exports, the home-manager module, and the packages read back out of the
# overlay. flake.nix imports it and keeps the development shell and the
# formatter.
{
  config,
  inputs,
  version,
  ...
}:
let
  overlay = import ./overlay.nix {
    inherit version;
    inherit (inputs.nixpkgs) lib;
    inherit (inputs) gomod2nix;
  };
in
{
  flake.overlays.default = overlay;

  # `homeModules` is the name the flake schema uses; `homeManagerModules` is
  # the name most configurations already reference.
  flake.homeModules = {
    default = ./hm-module.nix;
    tdl = ./hm-module.nix;
  };

  flake.homeManagerModules = {
    inherit (config.flake.homeModules) default tdl;
  };

  # The flake-parts module a project using the language imports, as opposed to
  # a configuration installing it for a person.
  flake.flakeModules = {
    default = ./flake-module.nix;
    tdl = ./flake-module.nix;
  };

  perSystem =
    { pkgs, system, ... }:
    {
      _module.args.pkgs = import inputs.nixpkgs {
        inherit system;
        overlays = [ overlay ];
      };

      packages = {
        default = pkgs.tdl;
        inherit (pkgs) tdl vscode-tdl;
      };

      # Holds the home-manager module to what it promises: the CLI lands in
      # home.packages and the extension lands in the VS Code profile. It reads
      # the evaluated options rather than home.path or activationPackage,
      # which would build VS Code itself to say the same thing. It gets its
      # own nixpkgs because `programs.vscode.enable` evaluates the editor, and
      # the editor is unfree.
      checks.hm-module =
        let
          hm = inputs.home-manager.lib.homeManagerConfiguration {
            pkgs = import inputs.nixpkgs {
              inherit system;
              overlays = [ overlay ];
              config.allowUnfree = true;
            };
            modules = [
              ./hm-module.nix
              {
                home = {
                  username = "tdl";
                  homeDirectory = "/home/tdl";
                  stateVersion = "24.11";
                };
                programs.vscode.enable = true;
                programs.tdl.enable = true;
              }
            ];
          };
        in
        assert pkgs.lib.assertMsg (builtins.elem pkgs.tdl hm.config.home.packages)
          "programs.tdl.enable did not add tdl to home.packages";
        assert pkgs.lib.assertMsg
          (builtins.elem pkgs.vscode-tdl hm.config.programs.vscode.profiles.default.extensions)
          "programs.tdl.vscode.enable did not add vscode-tdl to the default profile";
        pkgs.runCommand "tdl-hm-module" { } "touch $out";

      # Holds the flake-parts module to what it promises, by evaluating a
      # consumer flake that imports it and building what came out. The
      # fixture is a conformance case because the corpus is already held to
      # both properties asserted here: it parses clean and it is stored in
      # canonical form. tdl-gen is not built, since --verify compares against
      # generated output on disk and no fixture here has any.
      checks.flake-module =
        let
          consumer =
            inputs.flake-parts.lib.mkFlake
              {
                inputs = {
                  inherit (inputs) nixpkgs flake-parts;
                  self = { };
                };
              }
              {
                systems = [ system ];
                imports = [ ./flake-module.nix ];
                perSystem = _: {
                  _module.args.pkgs = pkgs; # already carries the overlay
                  tdl = {
                    enable = true;
                    src = ../testdata/conformance/entity;
                    files = [ "source.tdl" ];
                  };
                };
              };
        in
        pkgs.linkFarmFromDrvs "tdl-flake-module" [
          consumer.checks.${system}.tdl-check
          consumer.checks.${system}.tdl-fmt
          consumer.devShells.${system}.tdl
        ];
    };
}
