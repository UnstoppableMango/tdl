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
        inherit (pkgs)
          tdl
          tdl-gmk
          vscode-tdl
          tdl-dotnet
          ;
        default = pkgs.tdl;
        tree-sitter-tdl = pkgs.python3Packages.tree-sitter-tdl;
        # The README's demo GIF; `make demo` copies it into docs/demo.
        demo = pkgs.callPackage ./demo.nix { };
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

      # Building the binding imports it and parses the prelude with it.
      checks.tree-sitter-tdl = pkgs.python3Packages.tree-sitter-tdl;

      # Building the .NET plugins runs their tests: the SDK's replay
      # testdata/plugin, and the C# backend's compile what it generates.
      checks.dotnet-sdk = pkgs.tdl-dotnet;

      # Runs the .NET debug port from PATH through tdl gen, and holds it to
      # the file the Go debug backend writes for the same model.
      checks.gen-dotnet-debug =
        pkgs.runCommand "tdl-gen-dotnet-debug"
          {
            nativeBuildInputs = [
              pkgs.tdl
              pkgs.tdl-dotnet
            ];
          }
          ''
            export HOME=$TMPDIR DOTNET_CLI_TELEMETRY_OPTOUT=1
            cp ${../testdata/plugin/catalog.tdl} go.tdl
            sed 's/^target debug /target dotnet_debug /' go.tdl > dotnet.tdl
            tdl gen -o go go.tdl
            tdl gen -o dotnet dotnet.tdl
            sed -i 's/^target dotnet_debug$/target debug/' dotnet/model.txt
            diff go/model.txt dotnet/model.txt
            touch $out
          '';

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

      # Type checks the zod backend's module against the oldest zod it
      # supports, and loads it under node, which fails if a schema reads a
      # const before it is initialized.
      checks.gen-zod =
        let
          zod = pkgs.callPackage ./zod.nix { };
        in
        pkgs.runCommand "tdl-gen-zod"
          {
            nativeBuildInputs = [
              pkgs.tdl
              pkgs.nodejs
              pkgs.typescript
            ];
          }
          ''
            cp ${../testdata/gen/smoke/source.tdl} source.tdl
            tdl gen --target zod -o out source.tdl
            cd out
            ln -s ${zod}/lib/node_modules node_modules
            echo '{"type": "module"}' > package.json
            tsc --strict --exactOptionalPropertyTypes \
              --module nodenext --moduleResolution nodenext --target es2022 *.zod.ts
            node --input-type=module \
              -e 'const m = await import("./smoke.zod.js"); m.Payment.parse({ kind: "Cash" });'
            touch $out
          '';

      # Builds the csharp plugin's files with every warning an error and the
      # SDK's analyzers, which the plugin's own tests do not run. The project
      # references nothing but the framework, so it restores nothing from
      # NuGet.
      checks.gen-csharp =
        pkgs.runCommand "tdl-gen-csharp"
          {
            nativeBuildInputs = [
              pkgs.tdl
              pkgs.dotnet-sdk
            ];
          }
          ''
            export HOME=$TMPDIR DOTNET_CLI_TELEMETRY_OPTOUT=1 DOTNET_NOLOGO=1 DOTNET_SKIP_FIRST_TIME_EXPERIENCE=1
            cp ${../testdata/gen/smoke/source.tdl} source.tdl
            tdl gen --target csharp -o out source.tdl
            cp ${../testdata/csharp/check.csproj} out/check.csproj
            dotnet build out/check.csproj
            touch $out
          '';

      # Type checks the haskell backend's module with every warning an error.
      checks.gen-haskell =
        pkgs.runCommand "tdl-gen-haskell"
          {
            nativeBuildInputs = [
              pkgs.tdl
              pkgs.ghc
            ];
          }
          ''
            cp ${../testdata/gen/smoke/source.tdl} source.tdl
            tdl gen --target haskell -o out source.tdl
            ghc -fno-code -Wall -Werror out/*.hs
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

      # Lints the openapi backend's document in each version with vacuum,
      # a second implementation beside the schemas its tests validate
      # against. The rules turned off are ones a document holding schemas
      # and no paths cannot satisfy, and any other warning fails.
      checks.gen-openapi =
        let
          ruleset = pkgs.writeText "vacuum-ruleset.yaml" ''
            extends: [[vacuum:oas, recommended]]
            rules:
              info-description: off
              component-description: off
              oas3-api-servers: off
              oas3-missing-example: off
              oas3-unused-component: off
              oas2-api-host: off
              oas2-api-schemes: off
              oas2-unused-definition: off
          '';
        in
        pkgs.runCommand "tdl-gen-openapi"
          {
            nativeBuildInputs = [
              pkgs.tdl
              pkgs.vacuum-go
            ];
          }
          ''
            for v in 3.1 3.0 2.0; do
              sed "s/out(\"openapi\")/out(\"openapi\") openapi(\"$v\")/" \
                ${../testdata/gen/smoke/source.tdl} > source.tdl
              tdl gen --target openapi -o "out-$v" source.tdl
              vacuum lint --no-banner --no-style --details --fail-severity warn \
                --ruleset ${ruleset} "out-$v"/*.openapi.yaml
            done
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

      # Loads the make functions into GNU make and lists a model with them,
      # from a subdirectory so paths are relative to the loading makefile.
      checks.gmk =
        pkgs.runCommand "tdl-gmk"
          {
            nativeBuildInputs = [ pkgs.gnumake ];
          }
          ''
            mkdir -p project/model/sub project/shim
            cd project
            printf 'package shop\nimport "../shim/type.tdl" as _\n' > model/order.tdl
            echo 'package shop' > model/sub/money.tdl
            echo 'package shim' > shim/type.tdl
            printf 'load ${pkgs.tdl-gmk}/lib/tdl.so\nfiles := $(tdl-files model)\nimports := $(tdl-imports model/order.tdl)\nall: ; @echo "$(files)|$(imports)"\n' > Makefile
            got=$(cd model && make -s -f ../Makefile)
            want='model/order.tdl model/sub/money.tdl|model/order.tdl shim/type.tdl'
            if [ "$got" != "$want" ]; then
              echo "got:  $got" >&2
              echo "want: $want" >&2
              exit 1
            fi
            touch $out
          '';

      # Evaluates consumer flakes that import flake-module.nix and builds
      # their outputs. tdl-gen is left out: --verify compares against
      # generated output on disk, and the fixture has none.
      checks.flake-module =
        let
          consumer =
            tdl:
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
                  inherit tdl;
                };
              };
          model = {
            src = ../testdata/conformance/entity;
            files = [ "source.tdl" ];
          };
          # Setting src is enough to enable the module.
          implied = consumer model;
          defines = flake: builtins.attrNames (flake.checks.${system} // flake.devShells.${system});
        in
        # enable = false turns everything off, and a consumer setting nothing
        # gets nothing.
        assert defines (consumer (model // { enable = false; })) == [ ];
        assert defines (consumer { }) == [ ];
        pkgs.linkFarmFromDrvs "tdl-flake-module" [
          implied.checks.${system}.tdl-check
          implied.checks.${system}.tdl-fmt
          implied.devShells.${system}.tdl
          # A shell pulling devShells.tdl in with inputsFrom still gets TDL_GMK.
          (pkgs.runCommand "tdl-flake-module-inputs-from" { } ''
            ${(pkgs.mkShellNoCC { inputsFrom = [ implied.devShells.${system}.tdl ]; }).shellHook}
            test -f "$TDL_GMK"
            touch $out
          '')
        ];
    };
}
