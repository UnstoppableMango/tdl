ROOT := $(shell git rev-parse --show-toplevel)

.PHONY: build test clean docker

build:
	bun run build

test:
	bun test

clean:
	rm -rf dist

lint:
	bun lint

docker:
	@$(MAKE) -C ${ROOT}/docker uml2ts

test_image:
	@$(MAKE) -C ${ROOT}/docker uml2ts_test
