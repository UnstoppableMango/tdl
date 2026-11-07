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

    tdl-portable = final.callPackage ./portable.nix {
      inherit version;
      go = final.go_1_27;
    };

    tdl-npm = final.callPackage ./npm.nix { inherit version; };

    tdl-release = final.callPackage ./release.nix {
      inherit version;
      # Its GOOS and GOARCH name the build platform, whose tdl writes the
      # completion scripts.
      go = final.buildPackages.go_1_27;
    };

    tdl-gmk = final.callPackage ./gmk.nix {
      inherit version;
      go = final.go_1_27;
    };

    vscode-tdl = final.callPackage ./vscode-extension.nix { inherit version; };

    tdl-dotnet = final.callPackage ./dotnet.nix { inherit version; };

    tdl-rust = final.callPackage ./rust.nix { inherit version; };

    protoc-gen-buffa = final.callPackage ./protoc-gen-buffa.nix { };

    pythonPackagesExtensions = prev.pythonPackagesExtensions ++ [
      (pyFinal: _: {
        tree-sitter-tdl = pyFinal.callPackage ./tree-sitter-tdl.nix { };
      })
    ];
  }
)
