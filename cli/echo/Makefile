WORKING_DIR ?= $(shell git rev-parse --show-toplevel)
SRC         := $(shell find . -type f -path '*.go')
INTERNAL    := $(shell find ../internal -type f -path '*.go')
BIN         := $(WORKING_DIR)/bin/go_echo

.PHONY: build test lint tidy run clean vet

build: $(BIN)

test:
	go test

lint:
	golangci-lint run

tidy:
	go mod tidy

run:
	go run main.go

clean:
	rm -rf bin

vet:
	go vet

$(BIN): $(SRC) $(INTERNAL)
	go build -o $@
