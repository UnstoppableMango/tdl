# Generates Go and TypeScript from models/, builds the Go, and holds the
# TypeScript to DefinitelyTyped's declarations for the same trees; check.ts
# says what is compared. Nothing generated is committed, so this is where
# the models' output is checked.
{
  go,
  importNpmLock,
  nodejs,
  runCommand,
  tdl,
}:
let
  nodeModules = importNpmLock.buildNodeModules {
    npmRoot = ./.;
    inherit nodejs;
  };
in
runCommand "tdl-models-types"
  {
    nativeBuildInputs = [
      go
      nodejs
      tdl
    ];
  }
  ''
    export HOME=$TMPDIR GOCACHE=$TMPDIR/go-cache
    cp -r ${./.} check
    chmod -R u+w check
    ln -s ${nodeModules}/node_modules check/node_modules

    mkdir models
    printf 'module models\n\ngo 1.24\n' > models/go.mod
    for tree in unist mdast hast; do
      tdl gen --target go -o "models/$tree" ${../../../models}/$tree/$tree.tdl
      tdl gen --target typescript -o "check/out/$tree" ${../../../models}/$tree/$tree.tdl
    done

    (cd models && go vet ./...)
    (cd check && npm run check)
    touch $out
  ''
