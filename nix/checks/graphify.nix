{
  lib,
  runCommand,
  jq,
  graphify,
}:
let
  # The prelude and two conformance cases, laid out as in the repository so
  # the extractor finds prelude/std.tdl above each case.
  corpus = lib.fileset.toSource {
    root = ../..;
    fileset = lib.fileset.unions [
      ../../prelude/std.tdl
      (lib.fileset.fileFilter (file: file.hasExt "tdl") ../../testdata/conformance/entity)
      (lib.fileset.fileFilter (file: file.hasExt "tdl") ../../testdata/conformance/instances)
    ];
  };
in
# The one test of the binding and graphify's TDL extractor together: the
# prelude's Entity is a node, and a conformance case's declaration implements it.
runCommand "graphify-tdl"
  {
    nativeBuildInputs = [
      graphify
      jq
    ];
  }
  ''
    export HOME=$TMPDIR
    cp -r ${corpus} corpus
    chmod -R u+w corpus
    cd corpus
    graphify update . > update.log 2>&1 || { cat update.log; exit 1; }

    jq -e '
      (.nodes | map(select(.label == "Entity" and .source_file == "prelude/std.tdl")) | .[0].id) as $entity
      | $entity != null
        and ([.links[] | select(.relation == "implements" and .target == $entity)] | length > 0)
    ' graphify-out/graph.json > /dev/null || {
      echo "graphify-out/graph.json has no Entity node with an implements edge into it" >&2
      cat update.log >&2
      exit 1
    }
    touch $out
  ''
