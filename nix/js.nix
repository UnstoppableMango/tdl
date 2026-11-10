{
  lib,
  buildNpmPackage,
  importNpmLock,
  makeWrapper,
  nodejs,
  version,
}:
buildNpmPackage {
  pname = "tdl-js";
  inherit version;

  # The SDK's tests replay testdata/plugin and read reverse.md's loss codes.
  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../js/package.json
      ../js/package-lock.json
      ../js/tsconfig.base.json
      (lib.fileset.fileFilter (f: !(lib.hasPrefix "." f.name)) ../js/tdl-plugin)
      (lib.fileset.fileFilter (f: !(lib.hasPrefix "." f.name)) ../js/tdl-plugin-debug)
      ../testdata/plugin
      ../docs/architecture/reverse.md
    ];
  };

  # The workspace sits in js/, beside the files its tests read.
  sourceRoot = "source/js";

  npmDeps = importNpmLock { npmRoot = ../js; };
  npmConfigHook = importNpmLock.npmConfigHook;

  nativeBuildInputs = [ makeWrapper ];

  # The tests are compiled with the sources; the bun check swaps this phase.
  doCheck = true;
  checkPhase = ''
    runHook preCheck
    node --test '*/dist/test/**/*.test.js'
    runHook postCheck
  '';

  installPhase = ''
    runHook preInstall
    npm prune --omit=dev --no-audit --no-fund
    mkdir -p $out/lib/tdl-js $out/bin
    cp -r package.json node_modules tdl-plugin tdl-plugin-debug $out/lib/tdl-js/
    makeWrapper ${lib.getExe nodejs} $out/bin/tdl-gen-js_debug \
      --add-flags $out/lib/tdl-js/tdl-plugin-debug/dist/src/main.js
    runHook postInstall
  '';

  meta = {
    description = "The TDL backends written on the JS/TS plugin SDK: the debug port";
    homepage = "https://github.com/UnstoppableMango/tdl";
    mainProgram = "tdl-gen-js_debug";
    license = lib.licenses.mit;
  };
}
