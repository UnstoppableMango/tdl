GO_SRC ?= $(shell find . -name '*.go')

build:
	nix build .#

test:
	go test ./...

update:
	nix flake update

check lint:
	nix flake check

format fmt:
	nix fmt

tidy: go.sum nix/gomod2nix.toml

go.sum: go.mod ${GO_SRC}
	go mod tidy

nix/gomod2nix.toml: go.sum ${GO_SRC}
	go generate --dir ${CURDIR} --outdir ${@D}
