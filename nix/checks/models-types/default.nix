# Holds the TypeScript generated from models/ to DefinitelyTyped's
# declarations for the same trees; check.ts says what is compared.
{
  importNpmLock,
  nodejs,
  runCommand,
}:
let
  nodeModules = importNpmLock.buildNodeModules {
    npmRoot = ./.;
    inherit nodejs;
  };
in
# The tree is rebuilt as it is in the repository, since tsconfig.json reaches
# the models by a relative path.
runCommand "tdl-models-types" { nativeBuildInputs = [ nodejs ]; } ''
  mkdir -p nix/checks
  cp -r ${../../../models} models
  cp -r ${./.} nix/checks/models-types
  chmod -R u+w nix
  ln -s ${nodeModules}/node_modules nix/checks/models-types/node_modules
  cd nix/checks/models-types
  npm run check
  touch $out
''
