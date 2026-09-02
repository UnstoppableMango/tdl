// External scanner for the tokens the built-in lexer cannot produce.
//
// grammar.js declares `externals: [$.regex_lit]`, so parser.c calls into
// this file and nothing links without it. `/` opens a regex literal and
// divides a unit expression, and no token-local rule separates them: the
// reference implementation scans a regex only when the parser asks, via
// lex.RescanRegexAt, and tree-sitter's `valid_symbols` is the same
// question asked the other way round.
//
// This is the stub. It produces nothing, which leaves every construct but
// a regex literal parsing correctly; tree-sitter/corpus.sh defers the one
// conformance case that needs one. docs/design/treesitter-plan.md phase 6
// writes the scan.
//
// Hand-written, unlike the rest of src/. tools/treesitter never touches it.

#include "tree_sitter/parser.h"

enum TokenType {
	REGEX_LIT,
};

void *tree_sitter_tdl_external_scanner_create(void) {
	return NULL;
}

void tree_sitter_tdl_external_scanner_destroy(void *payload) {
	(void)payload;
}

// The scanner is stateless, so there is nothing to carry across an edit.
unsigned tree_sitter_tdl_external_scanner_serialize(void *payload, char *buffer) {
	(void)payload;
	(void)buffer;
	return 0;
}

void tree_sitter_tdl_external_scanner_deserialize(void *payload, const char *buffer, unsigned length) {
	(void)payload;
	(void)buffer;
	(void)length;
}

bool tree_sitter_tdl_external_scanner_scan(void *payload, TSLexer *lexer, const bool *valid_symbols) {
	(void)payload;
	(void)lexer;
	(void)valid_symbols;
	return false;
}
