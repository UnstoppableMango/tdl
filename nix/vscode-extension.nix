# The VS Code extension, installed through nix rather than the Marketplace:
# programs.tdl.vscode.enable in the home-manager module puts it in a profile,
# and it can be added to vscode-with-extensions or to home-manager's
# programs.vscode.profiles.<name>.extensions by hand.
{
  lib,
  vscode-utils,
  version,
}:
vscode-utils.buildVscodeExtension {
  pname = "tdl";
  inherit version;

  src = ../editors/vscode;

  # sourceRoot is the directory the unpacker copies src into, which takes its
  # name; buildVscodeExtension defaults it to a .vsix's layout, and this is a
  # directory in the tree.
  sourceRoot = "vscode";

  vscodeExtPublisher = "unstoppablemango";
  vscodeExtName = "tdl";
  vscodeExtUniqueId = "unstoppablemango.tdl";

  meta = {
    description = "Syntax highlighting for the Type Description Language";
    homepage = "https://github.com/UnstoppableMango/tdl";
    license = lib.licenses.gpl3Plus;
  };
}
