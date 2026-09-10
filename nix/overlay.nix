# The nixpkgs overlay this flake exports, and where both packages are named.
# perSystem imports nixpkgs with it and reads its packages back out, so
# `nix build .#` takes the same path a consumer does.
#
# gomod2nix's overlay is composed in because tdl is built with its
# buildGoApplication: a consumer adds one overlay rather than two, at the cost
# of buildGoApplication and mkGoEnv also landing in their pkgs.
{
  lib,
  gomod2nix,
  version,
}:
lib.composeExtensions gomod2nix.overlays.default (
  final: _prev: {
    tdl = final.callPackage ./cmd.nix {
      inherit version;
      go = final.go_1_27; # TODO: consolidate with flake.nix; optionally go.mod too
    };

    vscode-tdl = final.callPackage ./vscode-extension.nix { inherit version; };
  }
)
