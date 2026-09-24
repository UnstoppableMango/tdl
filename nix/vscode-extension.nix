# The VS Code extension, installed through nix rather than the Marketplace:
# programs.tdl.vscode.enable in the home-manager module puts it in a profile,
# and it can be added to vscode-with-extensions or to home-manager's
# programs.vscode.profiles.<name>.extensions by hand.
{
  lib,
  buildNpmPackage,
  importNpmLock,
  vscode-utils,
  version,
}:
let
  # The language client bundled into one file, with what the extension
  # ships beside it. importNpmLock fetches each package by the integrity
  # hash package-lock.json already records, so a dependency update needs no
  # hash edited here. node_modules and dist are gitignored, so a flake's
  # copy of the directory never carries a local install.
  bundle = buildNpmPackage {
    pname = "vscode-tdl-bundle";
    inherit version;

    src = ../editors/vscode;

    npmDeps = importNpmLock { npmRoot = ../editors/vscode; };
    npmConfigHook = importNpmLock.npmConfigHook;
    npmBuildScript = "bundle";

    # The typecheck runs here, so `nix flake check` holds the extension to
    # it. Biome is not run: npm's binary is linked against a loader the
    # sandbox does not have, and treefmt already runs nixpkgs' biome check.
    doCheck = true;
    checkPhase = ''
      runHook preCheck
      npm run typecheck
      runHook postCheck
    '';

    installPhase = ''
      runHook preInstall
      mkdir -p $out
      cp -r package.json language-configuration.json syntaxes dist $out/
      runHook postInstall
    '';
  };
in
vscode-utils.buildVscodeExtension {
  pname = "tdl";
  inherit version;

  src = bundle;

  # sourceRoot is the directory the unpacker copies src into, which takes
  # the bundle's name; buildVscodeExtension defaults it to a .vsix's layout.
  sourceRoot = bundle.name;

  vscodeExtPublisher = "unstoppablemango";
  vscodeExtName = "tdl";
  vscodeExtUniqueId = "unstoppablemango.tdl";

  passthru = { inherit bundle; };

  meta = {
    description = "Language support for the Type Description Language";
    homepage = "https://github.com/UnstoppableMango/tdl";
    license = lib.licenses.gpl3Plus;
  };
}
