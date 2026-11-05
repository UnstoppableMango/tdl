{
  runCommand,
  nodejs,
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
runCommand "tdl-npm-check" { nativeBuildInputs = [ nodejs ]; } ''
  export HOME=$TMPDIR npm_config_cache=$TMPDIR/npm-cache
  mkdir project && cd project
  echo '{ "name": "project", "private": true }' > package.json

  npm install --offline --ignore-scripts --no-audit --no-fund \
    ${tdl-npm}/unmango-tdl-${tdl-npm.version}.tgz \
    ${tdl-npm}/unmango-${native}-${tdl-npm.version}.tgz

  # The shim's #!/usr/bin/env node is right everywhere but the build sandbox,
  # which has no /usr/bin/env.
  patchShebangs node_modules/@unmango/tdl/tdl.js

  npx --offline tdl version
  cp ${../../testdata/gen/smoke/source.tdl} source.tdl
  npx --offline tdl gen --target typescript -o out source.tdl 2>/dev/null
  find out -name '*.ts' | grep -q . || { echo "tdl gen wrote no TypeScript" >&2; exit 1; }

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

  # A failing command's exit status reaches the caller.
  if npx --offline tdl check does-not-exist.tdl 2>/dev/null; then
    echo "tdl check of a missing file exited 0 through the shim" >&2
    exit 1
  fi
  touch $out
''
