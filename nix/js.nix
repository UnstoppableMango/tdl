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

  # The SDK's tests replay testdata/plugin and read reverse.md's loss codes,
  # and the mermaid backend's answer testdata/mermaid.
  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../js/package.json
      ../js/package-lock.json
      ../js/tsconfig.base.json
      (lib.fileset.fileFilter (f: !(lib.hasPrefix "." f.name)) ../js/tdl-plugin)
      (lib.fileset.fileFilter (f: !(lib.hasPrefix "." f.name)) ../js/tdl-plugin-debug)
      (lib.fileset.fileFilter (f: !(lib.hasPrefix "." f.name)) ../js/tdl-gen-mermaid)
      (lib.fileset.fileFilter (f: !(lib.hasPrefix "." f.name)) ../js/remark-tdl-mermaid)
      ../testdata/plugin
      ../testdata/mermaid
      ../docs/architecture/reverse.md
    ];
  };

  # The workspace sits in js/, beside the files its tests read.
  sourceRoot = "source/js";

  npmDeps = importNpmLock { npmRoot = ../js; };
  npmConfigHook = importNpmLock.npmConfigHook;

  nativeBuildInputs = [ makeWrapper ];

  # The tests are compiled with the sources; the bun check swaps this phase.
  # The remark plugin's need tdl, so they skip here, and
  # checks.remark-tdl-mermaid runs them against the packaged one.
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
    cp -r package.json node_modules tdl-plugin tdl-plugin-debug tdl-gen-mermaid remark-tdl-mermaid $out/lib/tdl-js/
    for bin in tdl-plugin-debug:tdl-gen-js_debug tdl-gen-mermaid:tdl-gen-mermaid; do
      makeWrapper ${lib.getExe nodejs} $out/bin/''${bin#*:} \
        --add-flags $out/lib/tdl-js/''${bin%%:*}/dist/src/main.js
    done
    runHook postInstall
  '';

  meta = {
    description = "The TDL backends written on the JS/TS plugin SDK, mermaid and the debug port, and the remark plugin drawing tdl fences";
    homepage = "https://github.com/UnstoppableMango/tdl";
    mainProgram = "tdl-gen-js_debug";
    license = lib.licenses.mit;
  };
}
