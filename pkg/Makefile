WORKING_DIR ?= $(shell git rev-parse --show-toplevel)

export GOWORK  :=
export BIN_DIR := $(WORKING_DIR)/bin

.PHONY: build test
build:
	go build ./...

test: echo_cli
	go test ./... -ginkgo.timeout=5s

tidy:
	go mod tidy

lint:
	golangci-lint run

vet:
	go vet ./...

# Don't like this, but can't think of a better way right now
echo_cli:
	@$(MAKE) -C ${WORKING_DIR}/cli/echo build --no-print-directory
