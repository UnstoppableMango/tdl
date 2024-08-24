.PHONY: build build_ts build_uml build_uml2ts
build: build_ts build_uml build_uml2ts
build_ts:
	cd ts && bun run build
build_uml2ts:
	cd uml2ts && bun run build

.PHONY: restore
restore:
	bun install

.PHONY: test
test:
	bun test .

.PHONY: clean clean_ts clean_uml clean_uml2ts
clean: clean_ts clean_uml clean_uml2ts
clean_ts:
	cd ts && bun clean
clean_uml2ts:
	cd uml2ts && bun clean

.PHONY: lint
lint:
	bun eslint .
