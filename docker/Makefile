CONTEXT := $(shell git rev-parse --show-toplevel)

.PHONY: uml2ts
uml2ts:
	docker build \
		-t unstoppablemango/uml2ts \
		-f uml2ts/Dockerfile \
		${CONTEXT}
