.PHONY: generate test

generate: src/parser.c

src/parser.c src/grammar.json src/node-types.json &: grammar.js tree-sitter.json
	tree-sitter generate

test:
	./corpus.sh
