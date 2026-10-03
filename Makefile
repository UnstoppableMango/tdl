PROTO_GO  := ir/ir.pb.go plugin/plugin.pb.go
PROTO_SRC ?= $(shell git ls-files proto)
GO_SRC    ?= $(filter-out ${PROTO_GO},$(shell git ls-files '*.go'))
TEST_DATA ?= $(shell git ls-files testdata prelude)

# The non-test Go files in the given package directories.
gosrc = $(filter-out %_test.go,$(wildcard $(addsuffix /*.go,$(1))))

VSCODE      := editors/vscode
TMLANGUAGE  := ${VSCODE}/syntaxes/tdl.tmLanguage.json
TS_GRAMMAR  := tree-sitter/grammar.js
TS_PARSER   := tree-sitter/src/parser.c tree-sitter/src/grammar.json tree-sitter/src/node-types.json

.PHONY: build test cover play generate treesitter textmate vscode-install \
	vscode-check test-treesitter check-treesitter update lint check fmt tidy

build:
	nix build .#

test:
	go test ./...

cover: cover.profile
	go tool cover -func=$<

cover.profile: ${GO_SRC} ${TEST_DATA}
	go test -race -coverprofile=$@ ./...

FILE ?= examples/nested.tdl
VIEWS ?= fmt,ast,stats
play:
	go run ./cmd/tdl play ${FILE} --views ${VIEWS}

generate: ${PROTO_GO}

${PROTO_GO} &: ${PROTO_SRC} buf.gen.yaml buf.yaml
	buf generate

treesitter: tree-sitter/src/parser.c

${TS_GRAMMAR}: docs/grammar.ebnf $(call gosrc,lex internal/ebnf internal/treesitter tools/treesitter)
	go run ./tools/treesitter

${TS_PARSER} &: ${TS_GRAMMAR} tree-sitter/tree-sitter.json
	cd tree-sitter && tree-sitter generate

textmate: ${TMLANGUAGE}

${TMLANGUAGE}: docs/grammar.ebnf $(call gosrc,lex internal/ebnf internal/textmate tools/textmate)
	go run ./tools/textmate

vscode-install: ${TMLANGUAGE}
	${MAKE} -C ${VSCODE} install

vscode-check:
	${MAKE} -C ${VSCODE} check

test-treesitter:
	./tree-sitter/corpus.sh

check-treesitter:
	${MAKE} -B treesitter
	git diff --exit-code -- tree-sitter
	${MAKE} test-treesitter

update:
	nix flake update

lint:
	nix flake check
	golangci-lint run ./...
	${MAKE} vscode-check
	buf lint
	buf format --diff --exit-code
	markdownlint-cli2

check:
	nix flake check

fmt:
	nix fmt
	buf format -w

tidy: go.sum nix/gomod2nix.toml

go.sum: go.mod ${GO_SRC}
	go mod tidy

nix/gomod2nix.toml: go.sum ${GO_SRC}
	gomod2nix --dir ${CURDIR} --outdir ${@D}
