{
  lib,
  gomod2nix,
  version,
}:
lib.composeExtensions gomod2nix.overlays.default (
  final: prev: {
    tdl = final.callPackage ./cmd.nix {
      inherit version;
      go = final.go_1_27; # TODO: consolidate with flake.nix; optionally go.mod too
    };

    tdl-gmk = final.callPackage ./gmk.nix {
      inherit version;
      go = final.go_1_27;
    };

    vscode-tdl = final.callPackage ./vscode-extension.nix { inherit version; };

    tdl-dotnet = final.callPackage ./dotnet.nix { inherit version; };

    pythonPackagesExtensions = prev.pythonPackagesExtensions ++ [
      (pyFinal: _: {
        tree-sitter-tdl = pyFinal.callPackage ./tree-sitter-tdl.nix { };
      })
    ];
  }
)
