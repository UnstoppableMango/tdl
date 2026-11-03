{
  buildGoApplication,
  lib,
  go,
  version,
}:
let
  # The platforms release assets and npm packages are built for, as GOOS/GOARCH.
  targets = [
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
  ];

  # Everything else is compiled into tdl; the content mapper runs tdl-gen-typescript.
  commands = [
    "./cmd/tdl"
    "./cmd/tdl-gen-typescript"
  ];
in
buildGoApplication {
  pname = "tdl-portable";
  inherit version go;

  src = lib.cleanSource ../.;
  modules = ./gomod2nix.toml;

  # Static on linux, and linked only to libSystem on darwin as every darwin Go
  # binary is, so they run outside Nix.
  CGO_ENABLED = 0;

  # Every target is cross-compiled from the build platform, and Go cross-compiles
  # pure Go with GOOS and GOARCH alone.
  buildPhase = ''
    runHook preBuild
    for target in ${lib.escapeShellArgs targets}; do
      export GOOS=''${target%/*} GOARCH=''${target#*/}
      echo "Building $GOOS/$GOARCH"
      go build -trimpath -ldflags="-s -w" -o "$NIX_BUILD_TOP/dist/$GOOS-$GOARCH/" ${lib.escapeShellArgs commands}
    done
    runHook postBuild
  '';

  installPhase = ''
    runHook preInstall
    mkdir -p $out
    cp -r "$NIX_BUILD_TOP/dist/." $out/
    runHook postInstall
  '';

  # packages.default runs the tests; these binaries are built from the same source.
  doCheck = false;

  # The Go linker ad-hoc signs darwin/arm64 binaries, and stripping them again
  # with the build platform's tools would break the signature.
  dontFixup = true;

  passthru = { inherit targets; };

  meta = {
    description = "tdl and tdl-gen-typescript for linux and darwin, amd64 and arm64, static on linux";
    homepage = "https://github.com/UnstoppableMango/tdl";
    license = lib.licenses.mit;
    platforms = lib.platforms.unix;
  };
}
