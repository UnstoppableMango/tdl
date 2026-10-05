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

      # `nix flake check` builds checks and not packages, and building the
      # extension is what typechecks and bundles it.
      checks.vscode-tdl = pkgs.vscode-tdl;

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

      # Validates the smithy backend's output with the Smithy CLI, since no Go
      # library can.
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

      # Type checks the typescript backend's output with tsc --strict.
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

      # Checks the jsonschema backend's document against its metaschema,
      # with a second implementation beside the one its tests use.
      checks.gen-jsonschema =
        pkgs.runCommand "tdl-gen-jsonschema"
          {
            nativeBuildInputs = [
              pkgs.tdl
              pkgs.check-jsonschema
            ];
          }
          ''
            cp ${../testdata/gen/smoke/source.tdl} source.tdl
            tdl gen --target jsonschema -o out source.tdl
            check-jsonschema --check-metaschema out/*.schema.json
            touch $out
          '';

      # Checks the salesforce backend's metadata XML is well formed. Nothing
      # checks the Apex, which has no parser outside an org.
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

      # Evaluates a consumer flake that imports flake-module.nix and builds
      # its outputs. tdl-gen is left out: --verify compares against generated
      # output on disk, and the fixture has none.
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
