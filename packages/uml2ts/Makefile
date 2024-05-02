ROOT := $(shell git rev-parse --show-toplevel)

.PHONY: build test clean docker

build:
	bun run build

test:
	bun test

clean:
	rm -rf dist

docker:
	@$(MAKE) -C ${ROOT}/docker uml2ts
