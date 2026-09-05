# The flake-parts module holding how tdl is built: the overlay the flake
# exports and the packages read back out of it. flake.nix imports it and keeps
# the development shell and the formatter.
{ inputs, version, ... }:
let
  overlay = import ./overlay.nix {
    inherit version;
    inherit (inputs.nixpkgs) lib;
    inherit (inputs) gomod2nix;
  };
in
{
  flake.overlays.default = overlay;

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
    };
}
