# Hercules CI's jobs (#1352, #1187).
#
# Without onPush, Hercules builds every check, package, and devShell on each
# push. onPush.default, the status `main` requires, builds the checks and every
# package but `demo`; onPush.coverage, a status nothing requires, runs the
# coverage effect, so a missing secret or an upload cannot block a merge; the
# nightly job builds
# `demo` and the devShells, which bring vhs, chromium, ffmpeg, and vscodium
# into a lock update's fetch and can fail on Hydra without anything in this
# repository changing.
#
# `herculesCI` is a function of the checkout, which the coverage effect
# reads. The systems input's flake module sets `herculesCI.ciSystems` as an
# attribute set, so the option takes either and merges both into one
# function; a second raw definition would conflict with it.
{
  config,
  inputs,
  lib,
  withSystem,
  ...
}:
let
  attrs = lib.types.lazyAttrsOf lib.types.raw;

  # ciSystems needs no checkout; the attribute set's definition holds it.
  inherit (config.flake.herculesCI { }) ciSystems;
  forCiSystems = f: lib.genAttrs ciSystems (system: f system);

  # The checks' coverage profile, uploaded to Codecov and DeepSource. A
  # failed upload is logged and never fails the job, so neither service can
  # block a merge. The tokens are Hercules secrets: `tdl-codecov` holds
  # `token`, and `tdl-deepsource` holds `dsn`.
  coverage =
    repo:
    withSystem "x86_64-linux" (
      { config, system, ... }:
      let
        # codecov-cli needs test-results-parser, which is source-available
        # rather than free.
        pkgs = import inputs.nixpkgs {
          inherit system;
          config.allowUnfreePredicate = pkg: lib.getName pkg == "test-results-parser";
        };
        hci-effects = inputs.hercules-ci-effects.lib.withPkgs pkgs;
        profile = "${config.checks.go-cover}/cover.profile";
      in
      hci-effects.mkEffect {
        name = "coverage";
        inputs = [
          pkgs.codecov-cli
          pkgs.deepsource
        ];
        secretsMap = {
          codecov = "tdl-codecov";
          deepsource = "tdl-deepsource";
        };
        effectScript = ''
          cp ${profile} cover.profile
          codecovcli upload-process \
            --token "$(readSecretString codecov .token)" \
            --git-service github \
            --slug ${lib.escapeShellArg "${repo.owner}/${repo.name}"} \
            --sha ${repo.rev} \
            ${lib.optionalString (repo.branch != null) "--branch ${lib.escapeShellArg repo.branch}"} \
            --disable-search \
            --file cover.profile \
            || echo "Codecov upload failed; coverage never blocks a merge"
          DEEPSOURCE_DSN="$(readSecretString deepsource .dsn)" GIT_COMMIT_SHA=${repo.rev} \
            deepsource report --analyzer test-coverage --key go --value-file cover.profile \
            || echo "DeepSource upload failed; coverage never blocks a merge"
        '';
      }
    );
in
{
  options.flake = lib.mkOption {
    type = lib.types.submoduleWith {
      modules = [
        {
          options.herculesCI = lib.mkOption {
            type = lib.types.coercedTo attrs (a: _: a) (lib.types.functionTo attrs);
            default = { };
            description = "The `herculesCI` flake output, read by Hercules CI.";
          };
        }
      ];
    };
  };

  config.flake.herculesCI = args: {
    onPush.default.outputs = {
      checks = forCiSystems (system: config.flake.checks.${system});
      packages = forCiSystems (system: removeAttrs config.flake.packages.${system} [ "demo" ]);
    };

    onPush.coverage.outputs.effects.coverage = coverage args.primaryRepo;

    onSchedule.nightly = {
      when = {
        hour = [ 4 ];
        minute = 0;
      };
      outputs = {
        packages = forCiSystems (system: {
          inherit (config.flake.packages.${system}) demo;
        });
        devShells = forCiSystems (system: config.flake.devShells.${system});
      };
    };
  };

  # The output as Hercules evaluates it, for a made-up checkout: it builds
  # only x86_64-linux, the one system with agents, and onPush.coverage runs
  # the coverage effect over the coverage check's profile. No ci.nix may
  # shadow it.
  config.perSystem =
    { pkgs, ... }:
    let
      ci = config.flake.herculesCI {
        primaryRepo = {
          owner = "UnstoppableMango";
          name = "tdl";
          rev = "0000000000000000000000000000000000000000";
          branch = "main";
        };
      };
      effect = ci.onPush.coverage.outputs.effects.coverage;
    in
    {
      checks.hercules =
        assert ci.ciSystems == [ "x86_64-linux" ];
        assert builtins.attrNames ci.onPush.default.outputs.checks == [ "x86_64-linux" ];
        assert effect.isEffect or false;
        assert lib.hasInfix "go-cover" effect.effectScript;
        # Hercules evaluates either file in place of the flake's herculesCI.
        assert !(builtins.pathExists ../ci.nix || builtins.pathExists ./ci.nix);
        pkgs.runCommand "tdl-hercules" { } "touch $out";
    };
}
