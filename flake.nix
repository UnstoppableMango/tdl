{
  description = "tdl - Type Description Language";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    systems.url = "github:UnstoppableMango/nix-systems";

    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };

    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.inputs.systems.follows = "systems";
    };

    # graphify and gremlins for the devShell
    mangopkgs = {
      url = "github:unmango/pkgs";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.systems.follows = "systems";
      inputs.flake-parts.follows = "flake-parts";
      inputs.treefmt-nix.follows = "treefmt-nix";
    };

    # Only checks.hm-module evaluates this; the module itself takes no input.
    home-manager = {
      url = "github:nix-community/home-manager";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import inputs.systems;

      imports = with inputs; [
        systems.flakeModule or { }
        treefmt-nix.flakeModule
        ./nix
      ];

      _module.args.version = "0.5.5"; # x-release-please-version

      perSystem =
        { inputs', pkgs, ... }:
        let
          # graphify reading TDL through the grammar's Python binding, built
          # against graphify's own interpreter.
          graphify = inputs'.mangopkgs.packages.graphify.withGrammars (ps: [
            (ps.callPackage ./nix/tree-sitter-tdl.nix { })
          ]);
        in
        {
          checks.graphify = pkgs.callPackage ./nix/checks/graphify.nix { inherit graphify; };

          devShells.default = pkgs.mkShell {
            packages = with pkgs; [
              direnv
              # 10 builds dotnet/; 8 builds the C# backend's generated code,
              # whose floor is net8.0.
              (dotnetCorePackages.combinePackages [
                dotnetCorePackages.sdk_10_0
                dotnetCorePackages.sdk_8_0
              ])
              go_1_27 # TODO: consolidate with nix/overlay.nix
              gomod2nix
              gopls
              golangci-lint
              ghc
              gnumake
              nixfmt
              nodejs
              buf
              markdownlint-cli2
              protobuf # protoc, for buf's C# generator
              protoc-gen-go
              smithy-cli
              typescript
              tree-sitter
              deepsource
              zip
              biome
              graphify
              inputs'.mangopkgs.packages.gremlins
            ];
            # The zod backend's tests type check and run its output with the
            # zod package, found here.
            NODE_PATH = "${pkgs.callPackage ./nix/zod.nix { }}/lib/node_modules";
          };

          devShells.treesitter = pkgs.mkShell {
            packages = [
              pkgs.go_1_27
              pkgs.gnumake
              pkgs.nodejs
              pkgs.tree-sitter
            ];
          };

          # What `make vscode-test` needs: the extension's toolchain, go to
          # build the server, and an editor under a virtual display.
          # VSCodium rather than a build @vscode/test-electron downloads, which
          # is linked against a loader NixOS does not have, and rather than
          # VS Code, which is unfree. VS_CODE names the Electron binary and
          # not bin/codium: that is the command-line launcher, which starts
          # the editor in the background and exits 0 before a test has run.
          devShells.vscode = pkgs.mkShell {
            packages = [
              pkgs.go_1_27
              pkgs.gnumake
              pkgs.nodejs
              pkgs.xvfb-run
            ];
            VS_CODE = "${pkgs.vscodium}/lib/vscode/codium";
          };

          treefmt = {
            programs = {
              actionlint.enable = true;
              biome = {
                enable = true;
                includes = [ "editors/vscode/**/*.ts" ];
                settings = removeAttrs (builtins.fromJSON (builtins.readFile ./editors/vscode/biome.json)) [
                  "$schema"
                  "files"
                ];
              };
              buf.enable = true;
              gofmt.enable = true;
              jsonfmt.enable = true;
              mdformat = {
                enable = true;
                settings.wrap = "keep";
              };
              nixfmt.enable = true;
              taplo.enable = true;
              yamlfmt = {
                enable = true;
                settings.formatter.retain_line_breaks_single = true;
              };
              zizmor.enable = true;
            };

            settings.formatter.jsonfmt.options = [
              "--indent"
              "\t"
            ];

            settings.global.excludes = [
              ".claude/**"
              "*.golden"
              "CHANGELOG.md"
              "nix/gomod2nix.toml"
              "flake.lock"
              "*.tdl"
              ".github/skills/**/SKILL.md"
              "tree-sitter/src/*.json"
              "editors/vscode/syntaxes/*.json"
            ];
          };
        };
    };
}
