{
  lib,
  buildNpmPackage,
  importNpmLock,
  jq,
  tdl,
  vscode-utils,
  version,
}:
let
  bundle = buildNpmPackage {
    pname = "vscode-tdl-bundle";
    inherit version;

    src = ../editors/vscode;

    npmDeps = importNpmLock { npmRoot = ../editors/vscode; };
    npmConfigHook = importNpmLock.npmConfigHook;
    npmBuildScript = "bundle";

    doCheck = true;
    checkPhase = ''
      runHook preCheck
      npm run typecheck
      runHook postCheck
    '';

    nativeBuildInputs = [ jq ];

    installPhase = ''
      runHook preInstall
      mkdir -p $out
      cp -r language-configuration.json syntaxes dist $out/
      jq -e '.contributes.configuration.properties | has("tdl.server.path")' \
        package.json > /dev/null
      jq '.contributes.configuration.properties."tdl.server.path".default = $path' \
        --arg path ${lib.escapeShellArg (lib.getExe tdl)} package.json > $out/package.json
      runHook postInstall
    '';
  };
in
vscode-utils.buildVscodeExtension {
  pname = "tdl";
  inherit version;

  src = bundle;

  # The unpacker copies src into a directory named after the bundle;
  # buildVscodeExtension defaults sourceRoot to a .vsix layout.
  sourceRoot = bundle.name;

  vscodeExtPublisher = "unstoppablemango";
  vscodeExtName = "tdl";
  vscodeExtUniqueId = "unstoppablemango.tdl";

  passthru = { inherit bundle; };

  meta = {
    description = "Language support for the Type Description Language";
    homepage = "https://github.com/UnstoppableMango/tdl";
    license = lib.licenses.mit;
  };
}
