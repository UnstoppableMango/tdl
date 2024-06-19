ROOT    := $(shell git rev-parse --show-toplevel)
BIN_DIR := $(ROOT)/bin

.PHONY: build test clean docker

build:
	bun build index.ts --compile --outfile ${BIN_DIR}/ts_echo

test:
	bun test

clean:
	rm -rf ${BIN_DIR}/ts_echo

docker:
	@$(MAKE) -C ${ROOT}/docker uml2ts

test_image:
	@$(MAKE) -C ${ROOT}/docker uml2ts_test
