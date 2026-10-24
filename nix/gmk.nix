{
  buildGoApplication,
  lib,
  go,
  gnumake,
  version,
}:
buildGoApplication {
  pname = "tdl-gmk";
  inherit version go;

  src = lib.cleanSource ../.;
  modules = ./gomod2nix.toml;

  # gnumake.h lives in gnumake's out output; there is no dev output.
  buildInputs = [ gnumake ];
  CGO_CFLAGS = "-I${gnumake}/include";

  # A shared object rather than an executable, so neither the default build
  # nor install applies. make derives the setup symbol from the base name,
  # which is why the file is tdl.so.
  buildPhase = ''
    runHook preBuild
    go build -tags makeplugin -buildmode=c-shared -o tdl.so ./cmd/tdl-gmk
    runHook postBuild
  '';

  # go test ./... runs internal/makefn's tests; checks.gmk loads this object
  # into make.
  doCheck = false;

  installPhase = ''
    runHook preInstall
    install -Dm644 tdl.so $out/lib/tdl.so
    runHook postInstall
  '';

  meta = {
    description = "GNU make functions for listing TDL files and their imports";
    homepage = "https://github.com/UnstoppableMango/tdl";
    license = lib.licenses.mit;
    platforms = lib.platforms.unix;
  };
}
