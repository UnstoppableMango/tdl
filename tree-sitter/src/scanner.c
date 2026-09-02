// External scanner for the tokens the built-in lexer cannot produce.
//
// `/` opens a regex literal and divides a unit expression, and nothing
// local to the token tells them apart: `matches` is a contextual keyword,
// so the token before the slash is an ordinary identifier either way.
//
// The reference implementation answers this by asking rather than
// guessing. lex.Lexer scans every other token without context, and the
// parser calls lex.RescanRegexAt when it wants a regex. tree-sitter's
// `valid_symbols` is the same question from the other side: it says
// whether the grammar admits a regex at this position, so the scanner
// produces one only where the parser would have asked for one.
//
// The shape scanned here is lex.RegexPattern, `/([^/\\\n]|\\[^\n])*/`,
// and the loop below is lex.RescanRegexAt read across.
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

// The scanner reads no state across tokens, so an edit leaves it nothing
// to carry. Both halves stay for the ABI, which calls them regardless.
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

	// Whitespace between the previous token and the slash belongs to the
	// extras rather than to the literal, so it is advanced over as skipped.
	while (is_space(lexer->lookahead)) {
		lexer->advance(lexer, true);
	}

	if (lexer->lookahead != '/') {
		return false;
	}
	lexer->advance(lexer, false);

	while (lexer->lookahead != '/') {
		// A regex does not span a line, so end of line and end of file are
		// both unterminated. Returning false leaves the slash to the
		// built-in lexer, which is what makes it an ERROR rather than a
		// literal running to the end of the file.
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
