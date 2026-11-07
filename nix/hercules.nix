# Hercules CI's jobs (#1352).
#
# Without onPush, Hercules builds every check, package, and devShell on each
# push. onPush.default, the status `main` requires, builds the checks and every
# package but `demo`; the nightly job builds `demo` and the devShells, which
# bring vhs, chromium, ffmpeg, and vscodium into a lock update's fetch and can
# fail on Hydra without anything in this repository changing.
#
# The systems input's flake module sets `herculesCI.ciSystems`, so this
# declares `flake.herculesCI` as an attribute set its keys merge into; a
# second raw definition would conflict with it.
{ config, lib, ... }:
let
  inherit (config.flake.herculesCI) ciSystems;
  forCiSystems = f: lib.genAttrs ciSystems (system: f system);
in
{
  options.flake = lib.mkOption {
    type = lib.types.submoduleWith {
      modules = [
        {
          options.herculesCI = lib.mkOption {
            type = lib.types.lazyAttrsOf lib.types.raw;
            default = { };
            description = "The `herculesCI` flake output, read by Hercules CI.";
          };
        }
      ];
    };
  };

  config.flake.herculesCI = {
    onPush.default.outputs = {
      checks = forCiSystems (system: config.flake.checks.${system});
      packages = forCiSystems (system: removeAttrs config.flake.packages.${system} [ "demo" ]);
    };

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
}
