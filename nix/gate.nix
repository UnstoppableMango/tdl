# The checks that make `nix flake check` the whole merge gate (#1187): the
# race run, coverage, and the lints, each a derivation, so Hercules CI runs
# them and a contributor runs the same with one command.
{ version, ... }:
{
  perSystem =
    {
      config,
      inputs',
      pkgs,
      lib,
      ...
    }:
    let
      goCheck = pkgs.callPackage ./checks/go.nix {
        go = pkgs.go_1_27;
        inherit version;
      };

      # Every external tool a test runs. Without -short a test fails when
      # its tool is missing, so a tool left out here fails the check by name.
      tools = [
        config.packages.plugins # tdl-gen-csharp and tdl-gen-mermaid run their recorded corpora
        (pkgs.callPackage ./tsc.nix { }) # 7.1, for the content mapper
        pkgs.nodejs
        pkgs.biome # a post command
        pkgs.smithy-cli
        pkgs.ghc
        pkgs.buf
        pkgs.gnumake
        inputs'.mangopkgs.packages.likec4
        (pkgs.callPackage ./cml-check.nix {
          inherit (inputs'.mangopkgs.packages) context-mapper-cli;
        })
      ];

      # zod for the zod backend's tests, and TypeScript 5's compiler API for
      # the typescript importer, which TypeScript 7's native tsc lacks.
      NODE_PATH = lib.makeSearchPath "lib/node_modules" [
        (pkgs.callPackage ./zod.nix { })
        pkgs.typescript_5
      ];
    in
    {
      # go test -race over the module, with every tool. It stays apart from
      # the coverage run: under -race every covered statement is an atomic
      # the detector tracks, which timed the protocompile tests out.
      checks.go-test = goCheck {
        name = "go-test";
        CGO_ENABLED = 1;
        nativeBuildInputs = tools;
        env = { inherit NODE_PATH; };
        script = ''
          gotest -race ./...
          touch $out
        '';
      };

      # The -short coverage profile, as $out/cover.profile, for a Hercules
      # effect to upload once the shared CI module carries one; until then the
      # Linux Actions job uploads its own. -short runs no external tool; the
      # race run does, and test code is outside the profile anyway.
      checks.go-cover = goCheck {
        name = "go-cover";
        script = ''
          gotest -short -coverpkg=./... -coverprofile=cover.profile ./...
          mkdir -p $out
          mv cover.profile $out/
        '';
      };

      checks.golangci-lint = goCheck {
        name = "golangci-lint";
        nativeBuildInputs = [ pkgs.golangci-lint ];
        script = ''
          export GOLANGCI_LINT_CACHE=$TMPDIR/golangci-lint
          golangci-lint run ./...
          touch $out
        '';
      };

      # Biome as `make vscode-check` and `make js-check` run it, each
      # directory with its own biome.json; building vscode-tdl and tdl-js
      # typechecks both.
      checks.biome =
        pkgs.runCommand "tdl-biome"
          {
            nativeBuildInputs = [ pkgs.biome ];
            src = lib.fileset.toSource {
              root = ../.;
              fileset = lib.fileset.unions [
                ../editors/vscode/biome.json
                ../editors/vscode/src
                ../editors/vscode/test
                ../js/biome.json
                (lib.fileset.fileFilter (f: f.hasExt "ts") ../js)
              ];
            };
          }
          ''
            cp -r $src src
            chmod -R u+w src
            (cd src/editors/vscode && biome check)
            (cd src/js && biome check)
            touch $out
          '';

      # `nix build .#release --rebuild`: the release uploads and attests the
      # tarballs as built, so a second build of the same inputs must write
      # the same bytes.
      checks.tdl-release-reproducible =
        let
          again = pkgs.tdl-release.overrideAttrs { TDL_REBUILD = "1"; };
        in
        pkgs.runCommand "tdl-release-reproducible" { } ''
          diff ${pkgs.tdl-release}/SHA256SUMS ${again}/SHA256SUMS
          touch $out
        '';

      # markdownlint over the files .markdownlint-cli2.yaml names.
      checks.markdownlint =
        pkgs.runCommand "tdl-markdownlint"
          {
            nativeBuildInputs = [ pkgs.markdownlint-cli2 ];
            src = lib.fileset.toSource {
              root = ../.;
              fileset = lib.fileset.unions [
                ../.markdownlint-cli2.yaml
                ../.markdownlint.yaml
                (lib.fileset.fileFilter (f: f.hasExt "md" || f.hasExt "markdown") ../.)
              ];
            };
          }
          ''
            cd $src
            markdownlint-cli2
            touch $out
          '';

      # `make check-treesitter`: the grammar regenerated from
      # docs/grammar.ebnf with the treesitter devShell's CLI is what is
      # committed, and the conformance corpus parses through it.
      checks.treesitter = goCheck {
        name = "treesitter";
        nativeBuildInputs = [
          pkgs.tree-sitter
          pkgs.nodejs
          pkgs.gnumake
          pkgs.stdenv.cc
        ];
        script = ''
          cp -r tree-sitter $TMPDIR/committed
          # The root Makefile's treesitter target, which reads git for
          # other targets' inputs.
          go run ./tools/treesitter
          make -B -C tree-sitter generate
          diff -r $TMPDIR/committed tree-sitter
          bash tree-sitter/corpus.sh
          touch $out
        '';
      };
    };
}
