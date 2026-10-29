PROTO_GO  := ir/ir.pb.go plugin/plugin.pb.go
PROTO_CS  := $(addprefix dotnet/Tdl.Plugin/Generated/,Ir.g.cs Plugin.g.cs)
PROTO_SRC ?= $(shell git ls-files proto)
GO_SRC    ?= $(filter-out ${PROTO_GO},$(shell git ls-files '*.go'))
TEST_DATA ?= $(shell git ls-files testdata prelude)

# The non-test Go files in the given package directories.
gosrc = $(filter-out %_test.go,$(wildcard $(addsuffix /*.go,$(1))))

VSCODE      := editors/vscode
TMLANGUAGE  := ${VSCODE}/syntaxes/tdl.tmLanguage.json
TS_GRAMMAR  := tree-sitter/grammar.js
FEATURES    := $(shell git grep -l '<!-- tdl:features' -- '*.md')

.PHONY: build test cover play demo generate treesitter textmate features vscode-install \
	vscode-check test-treesitter check-treesitter dotnet-test dotnet-deps mutate update lint check fmt tidy

build:
	nix build .#

test:
	go test ./...

cover: cover.profile
	go tool cover -func=$<

cover.profile: ${GO_SRC} ${PROTO_GO} ${TEST_DATA}
	go test -coverpkg=./... -coverprofile=$@ ./...

FILE ?= examples/nested.tdl
VIEWS ?= fmt,ast,stats
play:
	go run ./cmd/tdl play ${FILE} --views ${VIEWS}

demo:
	nix build .#demo -o result-demo
	install -m 644 result-demo/demo.gif docs/demo/demo.gif

generate: ${PROTO_GO} ${PROTO_CS}

${PROTO_GO} ${PROTO_CS} &: ${PROTO_SRC} buf.gen.yaml buf.yaml
	buf generate

treesitter: ${TS_GRAMMAR}
	${MAKE} -C tree-sitter generate

${TS_GRAMMAR}: docs/grammar.ebnf $(call gosrc,lex internal/ebnf internal/treesitter tools/treesitter)
	go run ./tools/treesitter

textmate: ${TMLANGUAGE}

${TMLANGUAGE}: docs/grammar.ebnf $(call gosrc,lex internal/ebnf internal/textmate tools/textmate)
	go run ./tools/textmate

features: ${FEATURES}

# The documents are inputs too: a cell the measurement skips keeps its text.
${FEATURES} &: ${GO_SRC} ${TEST_DATA} $(wildcard backend/*/features.toml dotnet/*/features.toml)
	go run ./tools/features

vscode-install: ${TMLANGUAGE}
	${MAKE} -C ${VSCODE} install

vscode-check:
	${MAKE} -C ${VSCODE} check

dotnet-test:
	dotnet test dotnet/Tdl.slnx

# Restores from nuget.org with dotnet, so it needs the network but not Nix;
# Renovate's NuGet updates run the same command.
dotnet-deps:
	go run ./tools/nugetdeps

# Every package in scope, or the ones PKG names.
PKG ?=
mutate:
	go run ./tools/mutate ${PKG}

test-treesitter:
	${MAKE} -C tree-sitter test

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
