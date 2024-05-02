CONTEXT := $(shell git rev-parse --show-toplevel)

.PHONY: uml2ts uml2ts_test
uml2ts:
	docker build \
		-f uml2ts/Dockerfile \
		-t unstoppablemango/uml2ts \
		${CONTEXT}

uml2ts_test:
	docker build \
		-f uml2ts/Dockerfile \
		-t unstoppablemango/uml2ts_test \
		--target test \
		${CONTEXT}
