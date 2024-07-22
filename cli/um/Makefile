ROOT := $(shell git rev-parse --show-toplevel)

export BIN_DIR := $(ROOT)/bin/

build:
	go build -o ${BIN_DIR}

test:
	go test

e2e:
	go test --ginkgo.focus 'End to end'

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
