PROTO_GO  := ir/ir.pb.go plugin/plugin.pb.go
PROTO_SRC ?= $(shell git ls-files proto)
# The committed protobuf output is left out, so a test run never
# regenerates it.
GO_SRC    ?= $(filter-out ${PROTO_GO},$(shell git ls-files '*.go'))
TEST_DATA ?= $(shell git ls-files testdata prelude)

# The non-test Go files in the given package directories.
gosrc = $(filter-out %_test.go,$(wildcard $(addsuffix /*.go,$(1))))

VSCODE      := editors/vscode
NODE_MODS   := ${VSCODE}/node_modules/.package-lock.json
BUNDLE      := ${VSCODE}/dist/extension.js
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

# Regenerate tree-sitter/grammar.js from docs/grammar.ebnf and the parser
# from grammar.js. Both are committed, so this only runs when the grammar
# changes; read the diff rather than trusting it.
treesitter: tree-sitter/src/parser.c

${TS_GRAMMAR}: docs/grammar.ebnf $(call gosrc,lex internal/ebnf internal/treesitter tools/treesitter)
	go run ./tools/treesitter

${TS_PARSER} &: ${TS_GRAMMAR} tree-sitter/tree-sitter.json
	cd tree-sitter && tree-sitter generate

# Regenerate the VS Code TextMate grammar from docs/grammar.ebnf. Committed
# like grammar.js, so this only runs when the grammar or the lexer changes.
textmate: ${TMLANGUAGE}

${TMLANGUAGE}: docs/grammar.ebnf $(call gosrc,lex internal/ebnf internal/textmate tools/textmate)
	go run ./tools/textmate

# Package editors/vscode and install it into a running VS Code.
vscode-install: ${BUNDLE} ${TMLANGUAGE}
	./${VSCODE}/install.sh

# Typecheck and lint the extension's TypeScript with the versions its lock
# file pins. `nix fmt` formats it; this is what an editor reports.
vscode-check: ${NODE_MODS}
	cd ${VSCODE} && npm run typecheck && npm run check

# npm writes this hidden lockfile on every install.
${NODE_MODS}: ${VSCODE}/package.json ${VSCODE}/package-lock.json
	cd ${VSCODE} && npm ci --no-audit --no-fund

${BUNDLE}: ${VSCODE}/src/extension.ts ${NODE_MODS}
	cd ${VSCODE} && npm run bundle

# Drive the extension in a headless editor against a freshly built server.
# Run it in `nix develop .#vscode`, which sets VS_CODE and has xvfb-run.
vscode-test:
	go build -o bin/tdl ./cmd/tdl
	cd editors/vscode && npm ci --no-audit --no-fund && TDL=${CURDIR}/bin/tdl xvfb-run -a npm test

# The conformance corpus, run by tree-sitter rather than by Go.
test-treesitter:
	./tree-sitter/corpus.sh

# What CI holds the derived parser to: regenerate, fail on a diff, then run
# both corpora through it. A fresh checkout leaves every file with about the
# same mtime, so -B forces the regeneration a file target would skip.
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
