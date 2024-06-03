CONTEXT    := $(shell git rev-parse --show-toplevel)
BROKER_TAG ?= unstoppablemango/tdl-broker:local

.PHONY: all
all: broker uml2go uml2pcl uml2ts

.PHONY: broker
broker:
	docker build \
		-f broker/Dockerfile \
		-t ${BROKER_TAG} \
		${CONTEXT}
# broker:
# 	@$(MAKE) -C ${CONTEXT}/src brokerc

.PHONY: uml2go uml2go_test
uml2go:
	docker build \
		-f uml2go/Dockerfile \
		-t unstoppablemango/uml2go:local \
		${CONTEXT}
uml2go_test:
	docker build \
		-f uml2go/Dockerfile \
		-t unstoppablemango/uml2go:test \
		${CONTEXT}

.PHONY: uml2pcl uml2pcl_test
uml2pcl:
	docker build \
		-f uml2pcl/Dockerfile \
		-t unstoppablemango/uml2pcl:local \
		${CONTEXT}
uml2pcl_test:
	docker build \
		-f uml2pcl/Dockerfile \
		-t unstoppablemango/uml2pcl:test \
		${CONTEXT}

.PHONY: uml2ts uml2ts_test
uml2ts:
	docker build \
		-f uml2ts/Dockerfile \
		-t unstoppablemango/uml2ts:local \
		${CONTEXT}
uml2ts_test:
	docker build \
		-f uml2ts/Dockerfile \
		-t unstoppablemango/uml2ts:test \
		--target test \
		${CONTEXT}
