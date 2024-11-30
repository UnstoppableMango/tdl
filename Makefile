_ := $(shell mkdir -p .make)
WORKING_DIR := $(shell git rev-parse --show-toplevel)

PROJECT := tdl
MODULE  := github.com/unstoppablemango/${PROJECT}

FIND := find
ifeq ($(shell uname),Darwin)
FIND := gfind
endif

LOCALBIN := ${WORKING_DIR}/bin
DEVOPS   := ${LOCALBIN}/devops
BUF      := ${LOCALBIN}/buf
GINKGO   := ${LOCALBIN}/ginkgo
GOLANGCI := ${LOCALBIN}/golangci-lint

export PATH := ${LOCALBIN}:${PATH}

GO_SRC    := $(shell $(DEVOPS) list --go)
TS_SRC    := $(shell $(DEVOPS) list --ts)
PROTO_SRC := $(shell $(DEVOPS) list --proto)
GO_PB_SRC ?= ${PROTO_SRC:proto/%.proto=pkg/%.pb.go}

GO_SUITES  ?= $(filter %_suite_test.go,${GO_SRC})
GO_REPORTS ?= $(addsuffix report.json,$(dir ${GO_SUITES}))

ifeq ($(CI),)
TEST_FLAGS := --label-filter '!E2E'
else
TEST_FLAGS := --github-output --race --trace
endif

build: generate .make/buf_build build_go build_ts

build_go: bin/ux bin/devops
build_ts: bin/uml2ts bin/zod2uml packages/tdl/dist packages/ts/dist

test: .make/go_test .make/ts_test
generate: ${GO_PB_SRC}
docker: .make/docker_ux .make/docker_uml2ts .make/docker_zod2uml
format: .make/dprint .make/go_fmt .make/buf_format
lint: .make/buf_lint .make/go_lint
tidy: go.sum

clean:
	rm -f bin/ux bin/uml2ts
	find . -type f -name 'report.json' -delete
	bun run --cwd packages/tdl clean
	bun run --cwd packages/ts clean

test_all: bin/ux bin/uml2ts bin/uml2uml
	$(GINKGO) run -r ./

e2e: .make/go_e2e_test

${GO_PB_SRC}: buf.gen.yaml ${PROTO_SRC} | bin/buf
	$(BUF) generate

packages/%/dist:
	bun run --cwd $(dir $@) build

%_suite_test.go: | bin/ginkgo
	cd $(dir $@) && $(GINKGO) bootstrap

$(GO_SRC:%.go=%_test.go): %_test.go: | bin/ginkgo
	cd $(dir $@) && $(GINKGO) generate $(notdir $*)

bin/ux: $(shell $(DEVOPS) list --go --exclude-tests)
	go -C cmd/ux build -o ${WORKING_DIR}/$@

bin/uml2uml: cmd/uml2uml/main.go
	go -C cmd/uml2uml build -o ${WORKING_DIR}/$@

bin/uml2ts: $(shell $(DEVOPS) list --ts --exclude-tests)
	bun build --cwd packages/uml2ts index.ts --compile --outfile ${WORKING_DIR}/$@

bin/zod2uml: $(shell $(DEVOPS) list --ts --exclude-tests)
	bun build --cwd packages/zod2uml index.ts --compile --outfile ${WORKING_DIR}/$@

bin/devops: $(shell $(DEVOPS) list --go --exclude-tests)
	go -C cmd/devops build -o ${WORKING_DIR}/$@

bin/buf: .versions/buf
	GOBIN=${LOCALBIN} go install github.com/bufbuild/buf/cmd/buf@v$(shell cat $<)

bin/ginkgo: go.mod
	GOBIN=${LOCALBIN} go install github.com/onsi/ginkgo/v2/ginkgo

bin/golangci-lint: .versions/golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${LOCALBIN} v$(shell cat $<)

.envrc: hack/example.envrc
	cp $< $@

buf.yaml: | bin/buf
	$(BUF) config init

buf.lock: | bin/buf
	$(BUF) dep update

go.mod:
	go mod init ${MODULE}

go.sum: go.mod ${GO_SRC}
	go mod tidy && touch $@

.make/docker_ux: ${GO_SRC} $(wildcard docker/ux/*)
	docker build -f docker/ux/Dockerfile -t ux ${WORKING_DIR}
	@touch $@

.make/docker_uml2ts: ${TS_SRC} $(wildcard docker/uml2ts/*)
	docker build -f docker/uml2ts/Dockerfile -t uml2ts ${WORKING_DIR}
	@touch $@

.make/docker_zod2uml: ${TS_SRC} $(wildcard docker/zod2uml/*)
	docker build -f docker/zod2uml/Dockerfile -t zod2uml ${WORKING_DIR}
	@touch $@

.make/go_test: ${GO_SRC} | bin/ginkgo bin/ux bin/uml2ts bin/uml2uml
	$(GINKGO) run ${TEST_FLAGS} $(sort $(dir $?))
	@touch $@

.make/go_e2e_test: ${GO_SRC} | bin/ginkgo bin/ux bin/uml2ts bin/uml2uml
	$(GINKGO) run --label-filter 'E2E' $(sort $(dir $?))
	@touch $@

.make/go_lint: ${GO_SRC} | bin/golangci-lint
	$(GOLANGCI) run $(sort $(dir $(filter %.go,$?)))
	@touch $@

.make/ts_test: ${TS_SRC}
	bun test --cwd packages/ts
	@touch $@

.make/buf_build: buf.yaml ${PROTO_SRC} | bin/buf
	$(BUF) build
	@touch $@

.make/buf_lint: buf.yaml ${PROTO_SRC} | bin/buf
	$(BUF) lint
	@touch $@

.make/buf_format: buf.yaml ${PROTO_SRC} | bin/buf
	$(BUF) format --write
	@touch $@

.make/dprint:
	dprint fmt

.make/go_fmt: ${GO_SRC}
	go fmt $(addprefix ./,$(sort $(dir $?)))
	@touch $@
