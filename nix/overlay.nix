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
