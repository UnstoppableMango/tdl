_ := $(shell mkdir -p .make)

FIND := find
ifeq ($(shell uname),Darwin)
FIND := gfind
endif

WORKING_DIR := $(shell git rev-parse --show-toplevel)

PROJECT := tdl
MODULE  := github.com/unstoppablemango/${PROJECT}

LOCALBIN := ${WORKING_DIR}/bin
DEVOPS   := ${LOCALBIN}/devops
BUF      := ${LOCALBIN}/buf
GINKGO   := ${LOCALBIN}/ginkgo

export PATH := ${LOCALBIN}:${PATH}

GO_SRC    := $(shell $(DEVOPS) list --go)
TS_SRC    := $(shell find packages -name '*.ts' -not -path '*/node_modules/*')
PROTO_SRC := $(shell $(DEVOPS) list --proto)
GO_PB_SRC ?= ${PROTO_SRC:proto/%.proto=pkg/%.pb.go}

# Temporarily focusing on cmd/ux
GO_SUITES  := $(filter cmd/ux/%_suite_test.go,${GO_SRC})
GO_REPORTS := $(addsuffix report.json,$(dir ${GO_SUITES}))

ifeq ($(CI),)
TEST_FLAGS := --json-report report.json --keep-separate-reports
else
TEST_FLAGS := --github-output --race --trace
endif

build: generate bin/ux bin/devops .make/buf_build packages/tdl/dist packages/ts/dist
test: ${GO_REPORTS}
generate: ${GO_PB_SRC}
format: .make/dprint
lint: .make/buf_lint
tidy: go.sum

clean:
	rm -f bin/ux bin/uml2ts
	find . -type f -name 'report.json' -delete
	bun run --cwd packages/tdl clean
	bun run --cwd packages/ts clean

${GO_PB_SRC}: buf.gen.yaml ${PROTO_SRC} | bin/buf
	$(BUF) generate

packages/%/dist:
	bun run --cwd $(dir $@) build

cmd/ux/report.json: ${TS_SRC} | bin/ux bin/uml2ts
${GO_REPORTS} &: ${GO_SRC} | bin/ginkgo
	$(GINKGO) run ${TEST_FLAGS} $(dir $@)

%_suite_test.go: | bin/ginkgo
	cd $(dir $@) && $(GINKGO) bootstrap

$(GO_SRC:%.go=%_test.go): %_test.go: | bin/ginkgo
	cd $(dir $@) && $(GINKGO) generate $(notdir $*)

bin/ux: ${GO_SRC}
	go -C cmd/ux build -o ${WORKING_DIR}/$@

bin/uml2ts: ${TS_SRC}
	bun build --cwd packages/uml2ts index.ts --compile --outfile ${WORKING_DIR}/$@

bin/devops: ${GO_SRC}
	go -C cmd/devops build -o ${WORKING_DIR}/$@

bin/buf: .versions/buf
	GOBIN=${LOCALBIN} go install github.com/bufbuild/buf/cmd/buf@v$(shell cat $<)

bin/ginkgo: go.mod
	GOBIN=${LOCALBIN} go install github.com/onsi/ginkgo/v2/ginkgo

.envrc: hack/example.envrc
	cp $< $@

buf.yaml: | bin/buf
	$(BUF) config init

buf.lock: | bin/buf
	$(BUF) dep update

go.mod:
	go mod init ${MODULE}

%/go.mod:
	go -C $(dir $@) mod init ${MODULE}/$*

go.sum: go.mod ${GO_SRC}
	go mod tidy && touch $@

.make/buf_build: buf.yaml ${PROTO_SRC} | bin/buf
	$(BUF) build
	@touch $@

.make/buf_lint: buf.yaml ${PROTO_SRC} | bin/buf
	$(BUF) lint
	@touch $@

.make/dprint:
	dprint fmt
