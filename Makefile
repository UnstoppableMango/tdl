GO_SRC ?= $(shell find . -name '*.go')

build:
	nix build .#

test:
	go test ./...

cover: cover.profile
	go tool cover -func=$<

cover.profile: ${GO_SRC}
	go test -race -coverprofile=$@ ./...

validate_codecov: codecov.yml
	curl -X POST --data-binary @codecov.yml https://codecov.io/validate

# Watch a TDL file and re-render it on every save.
# Override the target: make play FILE=examples/nested.tdl VIEWS=all
FILE ?= examples/nested.tdl
VIEWS ?= fmt,ast,stats
play:
	go run ./cmd/tdl play ${FILE} --views ${VIEWS}

# Regenerate ir/ir.pb.go from proto/. The generated file is committed, so
# this only runs when the schema changes.
generate:
	buf generate

# Regenerate tree-sitter/grammar.js from docs/grammar.ebnf and the parser
# from grammar.js. Both are committed, so this only runs when the grammar
# changes; read the diff rather than trusting it.
treesitter:
	go test ./internal/treesitter -update
	cd tree-sitter && tree-sitter generate

# Regenerate the VS Code TextMate grammar from docs/grammar.ebnf. Committed
# like grammar.js, so this only runs when the grammar or the lexer changes.
textmate:
	go test ./internal/textmate -update

# Package editors/vscode and install it into a running VS Code. The
# grammar it carries is whatever `make textmate` last wrote.
vscode-install:
	./editors/vscode/install.sh

# The conformance corpus, run by tree-sitter rather than by Go.
test-treesitter:
	./tree-sitter/corpus.sh

update:
	nix flake update

check lint:
	nix flake check
	golangci-lint run ./...
	buf lint
	buf format --diff --exit-code
	markdownlint-cli2

format fmt:
	nix fmt
	buf format -w

tidy: go.sum nix/gomod2nix.toml

go.sum: go.mod ${GO_SRC}
	go mod tidy

nix/gomod2nix.toml: go.sum ${GO_SRC}
	gomod2nix --dir ${CURDIR} --outdir ${@D}
