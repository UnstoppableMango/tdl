{
  lib,
  stdenvNoCC,
  buildPackages,
  bun,
  fetchurl,
  unzip,
  tdl-js,
  version,
}:
let
  # The plugin as tdl-js built it, with the dependencies bun bundles in.
  entry = "${tdl-js}/lib/tdl-js/tdl-gen-mermaid/dist/src/main.js";

  # Each tdl-portable target's Bun, which bun build --compile writes the
  # bundle into. A release must match the Bun that bundles, so bumping
  # nixpkgs' bun means bumping these hashes.
  bunVersion = "1.4.4";
  releases = {
    "linux/amd64" = {
      name = "bun-linux-x64-baseline";
      hash = "sha256-4ERutVmVdy65iUvrrppr42jZhai5Jx7hJEhvmUbu3XY=";
    };
    "linux/arm64" = {
      name = "bun-linux-aarch64";
      hash = "sha256-uqa7NvUdblhWHgK76oJQOKnfRU9nPW13bou/hvjHhQ4=";
    };
    "darwin/amd64" = {
      name = "bun-darwin-x64";
      hash = "sha256-UKNJH3Orwv6BZzFxqFdi07wswWgMzsrdmD3pmt93l34=";
    };
    "darwin/arm64" = {
      name = "bun-darwin-aarch64";
      hash = "sha256-wkDX8My5DCY6hJ/NsJt+5iYw2SE0jtaq/ahrnVd7xRA=";
    };
  };
  fetch =
    r:
    fetchurl {
      url = "https://github.com/oven-sh/bun/releases/download/bun-v${bunVersion}/${r.name}.zip";
      inherit (r) hash;
    };

  bundler = buildPackages.bun;

  # Every byte as bun wrote it: the bundle is a section of the executable,
  # which strip and patchelf's rpath rewrite break, and bun signs a darwin
  # executable itself.
  common = {
    inherit version;
    dontUnpack = true;
    dontFixup = true;
    nativeBuildInputs = [ bundler ];
    meta = {
      homepage = "https://github.com/UnstoppableMango/tdl";
      license = lib.licenses.mit;
    };
  };

  # tdl-gen-mermaid for linux and darwin, amd64 and arm64, as
  # <goos>-<goarch>/tdl-gen-mermaid beside tdl-portable's layout, for the
  # release tarballs and the npm platform packages. Each is the official
  # Bun for its platform, so the linux ones need the system's glibc.
  portable = stdenvNoCC.mkDerivation (
    common
    // {
      pname = "tdl-gen-mermaid-portable";
      nativeBuildInputs = [
        bundler
        unzip
      ];
      buildPhase = ''
        runHook preBuild
        export HOME=$TMPDIR
        ${lib.concatStrings (
          lib.mapAttrsToList (target: r: ''
            unzip -q ${fetch r} -d $TMPDIR/bun
            bun build --compile --target=${r.name} \
              --compile-executable-path=$TMPDIR/bun/${r.name}/bun \
              ${entry} --outfile $out/${lib.replaceStrings [ "/" ] [ "-" ] target}/tdl-gen-mermaid
          '') releases
        )}
        runHook postBuild
      '';
      passthru.targets = lib.attrNames releases;
      meta = common.meta // {
        description = "tdl-gen-mermaid as a Bun single executable for linux and darwin, amd64 and arm64";
      };
    }
  );
in
assert lib.assertMsg (bundler.version == bunVersion)
  "nix/mermaid.nix fetches Bun ${bunVersion}, but nixpkgs' bun is ${bundler.version}: update bunVersion and the hashes";
# tdl-gen-mermaid as a Bun single executable for this platform, written into
# nixpkgs' own bun so it runs under Nix, for nix/cmd.nix to ship.
stdenvNoCC.mkDerivation (
  common
  // {
    pname = "tdl-gen-mermaid";
    buildPhase = ''
      runHook preBuild
      export HOME=$TMPDIR
      bun build --compile --compile-executable-path=${lib.getExe bun} \
        ${entry} --outfile $out/bin/tdl-gen-mermaid
      runHook postBuild
    '';
    passthru = { inherit portable; };
    meta = common.meta // {
      description = "The TDL mermaid backend as a Bun single executable";
      mainProgram = "tdl-gen-mermaid";
    };
  }
)
