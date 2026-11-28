# Hercules CI's jobs and the Renovate fixup, from mangonix's flakeModules.ci.
#
# The nightly job builds `demo` and the devShells, which bring vhs, chromium,
# ffmpeg, and vscodium into a lock update's fetch and can fail on Hydra without
# anything in this repository changing.
#
# The fixup regenerates what an update leaves stale. Each target is a file
# target, and a fresh checkout gives every file the same mtime, so `-B`.
# tree-sitter's output is only stable against the treesitter shell's CLI.
{
  ci.nightly.packages = [ "demo" ];

  perSystem.ci.fixup.steps = [
    "nix develop --command make -B tidy dotnet-deps generate textmate"
    "nix develop .#treesitter --command make -B treesitter"
  ];
}
