{
  lib,
  runCommand,
  stdenv,
  nodejs,
  patchelf,
  go,
  tdl-npm,
}:
let
  # The platform package this system can run.
  cpu = {
    amd64 = "x64";
    arm64 = "arm64";
  };
  native = "tdl-${go.GOOS}-${cpu.${go.GOARCH}}";
in
# Installs @unmango/tdl and this system's platform package from the tarballs
# tdl-npm packed, with no registry, as a project would, and runs tdl through
# the shim. The other platform packages are optional dependencies npm skips,
# since their os or cpu does not match.
runCommand "tdl-npm-check"
  {
    nativeBuildInputs = [
      nodejs
      patchelf
    ];
  }
  ''
    export HOME=$TMPDIR npm_config_cache=$TMPDIR/npm-cache
    mkdir project && cd project
    echo '{ "name": "project", "private": true }' > package.json

    npm install --offline --ignore-scripts --no-audit --no-fund \
      ${tdl-npm}/unmango-tdl-${tdl-npm.version}.tgz \
      ${tdl-npm}/unmango-${native}-${tdl-npm.version}.tgz \
      ${tdl-npm}/unmango-tdl-typescript-${tdl-npm.version}.tgz \
      ${tdl-npm}/unmango-remark-tdl-mermaid-${tdl-npm.version}.tgz

    # The shim's #!/usr/bin/env node is right everywhere but the build sandbox,
    # which has no /usr/bin/env.
    patchShebangs node_modules/@unmango/tdl/tdl.js

    npx --offline tdl version
    cp ${../../testdata/gen/smoke/source.tdl} source.tdl
    npx --offline tdl gen --target typescript -o out source.tdl 2>/dev/null
    find out -name '*.ts' | grep -q . || { echo "tdl gen wrote no TypeScript" >&2; exit 1; }

    # tdl finds tdl-gen-mermaid beside itself, which the shim puts on PATH.
    # On linux it names the system's glibc loader, which the sandbox lacks.
    ${lib.optionalString stdenv.hostPlatform.isLinux ''
      chmod u+w node_modules/@unmango/${native}/bin/tdl-gen-mermaid
      patchelf --set-interpreter ${stdenv.cc.bintools.dynamicLinker} \
        node_modules/@unmango/${native}/bin/tdl-gen-mermaid
    ''}
    npx --offline tdl gen --target mermaid -o out source.tdl
    diff ${../../testdata/mermaid/corpus/smoke/want/smoke.mmd} out/smoke.mmd

    # tdl-gen-typescript, for the content mapper, resolves the same way.
    node -e '
      const { spawnSync } = require("node:child_process");
      const { binaryPath } = require("@unmango/tdl/resolve");
      const run = spawnSync(binaryPath("tdl-gen-typescript"), ["--help"], { encoding: "utf8" });
      if (!run.stderr.includes("usage: tdl-gen-typescript")) {
        console.error("tdl-gen-typescript did not run:", run.error ?? run.stderr);
        process.exit(1);
      }
    '

    # @unmango/tdl-typescript names the command tsc runs for .tdl files.
    node -e '
      const { contentMapper } = require("@unmango/tdl-typescript/package.json").typescript;
      if (contentMapper.exec.join(" ") !== "tdl-gen-typescript content-mapper") {
        console.error("@unmango/tdl-typescript runs", contentMapper.exec);
        process.exit(1);
      }
    '

    # The remark plugin finds tdl through @unmango/tdl/resolve. Its
    # transformer runs on a hand-written tree, since remark is not here.
    cat > remark.mjs <<'EOF'
    import remarkTdlMermaid from "@unmango/remark-tdl-mermaid";
    import { readFileSync } from "node:fs";
    const source = readFileSync("source.tdl", "utf8");
    const tree = { type: "root", children: [{ type: "code", lang: "tdl", meta: "mermaid", value: source }] };
    const file = { message: () => ({}), fail(reason) { throw new Error(reason); } };
    await remarkTdlMermaid()(tree, file);
    const [code] = tree.children;
    const want = readFileSync(process.argv[2], "utf8");
    if (code.lang !== "mermaid" || `''${code.value}\n` !== want) {
      console.error("the remark plugin drew", code);
      process.exit(1);
    }
    EOF
    node remark.mjs ${../../testdata/mermaid/corpus/smoke/want/smoke.mmd}

    # A failing command's exit status reaches the caller.
    if npx --offline tdl check does-not-exist.tdl 2>/dev/null; then
      echo "tdl check of a missing file exited 0 through the shim" >&2
      exit 1
    fi
    touch $out
  ''
