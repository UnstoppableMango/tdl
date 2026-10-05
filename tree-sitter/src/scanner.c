// External scanner for regex_lit, the one token the generated lexer cannot
// produce.
//
// `/` opens a regex literal and divides a unit expression, and the token
// before it does not tell them apart. The reference parser calls
// lex.RescanRegexAt when it wants a regex; here `valid_symbols` says
// whether the grammar admits one at this position.
//
// The shape scanned is lex.RegexPattern, `/([^/\\\n]|\\[^\n])*/`, and the
// loop below follows lex.RescanRegexAt.
//
// Hand-written, unlike the rest of src/.

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

// The scanner keeps no state. The ABI requires both functions.
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

static bool is_space(int32_t c) {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r';
}

bool tree_sitter_tdl_external_scanner_scan(void *payload, TSLexer *lexer, const bool *valid_symbols) {
	(void)payload;

	if (!valid_symbols[REGEX_LIT]) {
		return false;
	}

	// Leading whitespace is skipped as extras, not part of the literal.
	while (is_space(lexer->lookahead)) {
		lexer->advance(lexer, true);
	}

	if (lexer->lookahead != '/') {
		return false;
	}
	lexer->advance(lexer, false);

	while (lexer->lookahead != '/') {
		// A regex does not span a line. Returning false leaves the slash to
		// the built-in lexer, which makes it an ERROR.
		if (lexer->eof(lexer) || lexer->lookahead == '\n') {
			return false;
		}
		if (lexer->lookahead == '\\') {
			lexer->advance(lexer, false);
			if (lexer->eof(lexer) || lexer->lookahead == '\n') {
				return false;
			}
		}
		lexer->advance(lexer, false);
	}

	lexer->advance(lexer, false);
	lexer->result_symbol = REGEX_LIT;
	return true;
}
