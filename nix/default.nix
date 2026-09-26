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
  flake = {
    overlays.default = overlay;

    homeModules = {
      default = ./hm-module.nix;
      tdl = ./hm-module.nix;
    };

    homeManagerModules = {
      inherit (config.flake.homeModules) default tdl;
    };

    flakeModules = {
      default = ./flake-module.nix;
      tdl = ./flake-module.nix;
    };
  };

  perSystem =
    { pkgs, system, ... }:
    {
      _module.args.pkgs = import inputs.nixpkgs {
        inherit system;
        overlays = [ overlay ];
      };

      packages = {
        inherit (pkgs) tdl vscode-tdl;
        default = pkgs.tdl;
      };

      checks.hm-module =
        let
          pkgs = import inputs.nixpkgs {
            inherit system;
            overlays = [ overlay ];
            config.allowUnfree = true;
          };
        in
        pkgs.callPackage ./checks/hm-module.nix {
          inherit (inputs.home-manager.lib) homeManagerConfiguration;
        };

      # Holds the smithy backend to output Smithy accepts. No Go library can
      # say so, which is why this is a check rather than a Go test: it
      # generates from the smoke fixture and hands the result to the CLI.
      checks.gen-smithy =
        pkgs.runCommand "tdl-gen-smithy"
          {
            nativeBuildInputs = [
              pkgs.tdl
              pkgs.smithy-cli
            ];
          }
          ''
            export HOME=$TMPDIR
            cp ${../testdata/gen/smoke/source.tdl} source.tdl
            tdl gen --target smithy -o out source.tdl
            smithy validate --quiet --no-config out/*.smithy
            touch $out
          '';

      # Holds the typescript backend to declarations tsc accepts under
      # --strict, for the same reason gen-smithy exists.
      checks.gen-typescript =
        pkgs.runCommand "tdl-gen-typescript"
          {
            nativeBuildInputs = [
              pkgs.tdl
              pkgs.typescript
            ];
          }
          ''
            cp ${../testdata/gen/smoke/source.tdl} source.tdl
            tdl gen --target typescript -o out source.tdl
            tsc --noEmit --strict out/*.ts
            touch $out
          '';

      # Holds the salesforce backend to well-formed metadata XML. Apex has
      # no parser outside an org, so the classes are checked by deploying,
      # which no check can do.
      checks.gen-salesforce =
        pkgs.runCommand "tdl-gen-salesforce"
          {
            nativeBuildInputs = [
              pkgs.tdl
              pkgs.findutils
              pkgs.libxml2
            ];
          }
          ''
            cp ${../testdata/gen/smoke/source.tdl} source.tdl
            tdl gen --target salesforce -o out source.tdl
            find out -name '*.xml' -exec xmllint --noout {} +
            touch $out
          '';

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
