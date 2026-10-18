import os
from pathlib import Path
from unittest import TestCase, skipUnless

from tree_sitter import Language, Parser, Query
import tree_sitter_tdl

# The prelude sits outside the package source, so a build that copies only
# tree-sitter/ names it through TDL_PRELUDE.
PRELUDE = Path(
    os.environ.get("TDL_PRELUDE", Path(__file__).parents[4] / "prelude" / "std.tdl")
)


class TestLanguage(TestCase):
    def test_can_load_grammar(self):
        try:
            Parser(Language(tree_sitter_tdl.language()))
        except Exception:
            self.fail("Error loading TDL grammar")

    @skipUnless(PRELUDE.exists(), "the prelude is not in this source tree")
    def test_parses_prelude(self):
        parser = Parser(Language(tree_sitter_tdl.language()))
        tree = parser.parse(PRELUDE.read_bytes())
        self.assertFalse(tree.root_node.has_error)

    def test_highlights_query_compiles(self):
        Query(Language(tree_sitter_tdl.language()), tree_sitter_tdl.HIGHLIGHTS_QUERY)
