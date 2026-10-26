# The tree-sitter grammar's Python binding, which graphify reads TDL with
# (see docs/architecture/graphify.md).
{
  buildPythonPackage,
  lib,
  setuptools,
  wheel,
  tree-sitter,
  pytestCheckHook,
}:
let
  # Read from tree-sitter.json, which tree-sitter generate copies into
  # parser.c and release-please leaves alone.
  inherit (lib.importJSON ../tree-sitter/tree-sitter.json) metadata;
in
buildPythonPackage {
  pname = "tree-sitter-tdl";
  inherit (metadata) version;
  pyproject = true;

  src = ../tree-sitter;

  build-system = [
    setuptools
    wheel
  ];

  nativeCheckInputs = [
    tree-sitter
    pytestCheckHook
  ];

  # The binding's tests parse the prelude, which is outside the package.
  preCheck = ''
    export TDL_PRELUDE=${../prelude/std.tdl}
  '';

  pythonImportsCheck = [ "tree_sitter_tdl" ];

  meta = {
    description = "TDL grammar for tree-sitter, as a Python binding";
    homepage = "https://github.com/UnstoppableMango/tdl";
    license = lib.licenses.mit;
  };
}
