_ := $(shell mkdir -p .make)
WORKING_DIR := $(shell git rev-parse --show-toplevel)

PROJECT := tdl
MODULE  := github.com/unstoppablemango/${PROJECT}

FIND := find
ifeq ($(shell uname),Darwin)
FIND := gfind
endif

LOCALBIN := ${WORKING_DIR}/bin
DEVCTL   := go tool devctl
BUF      := go tool buf
GINKGO   := go tool ginkgo
GOLANGCI := ${LOCALBIN}/golangci-lint

export PATH := ${LOCALBIN}:${PATH}

GO_SRC    := $(shell $(DEVCTL) list --go)
TS_SRC    := $(shell $(DEVCTL) list --ts)
PROTO_SRC := $(shell $(DEVCTL) list --proto)
GO_PB_SRC ?= ${PROTO_SRC:proto/%.proto=pkg/%.pb.go}

GO_SUITES  ?= $(filter %_suite_test.go,${GO_SRC})
GO_REPORTS ?= $(addsuffix report.json,$(dir ${GO_SUITES}))

DOTNET_CONF := Debug
DOTNET_TFM  := net9.0

ifeq ($(CI),)
TEST_FLAGS := --label-filter '!E2E'
else
TEST_FLAGS := --github-output --race --trace
endif

build: generate .make/buf_build build_go build_ts build_dotnet

build_go: bin/ux
build_ts: bin/uml2ts bin/zod2uml packages/tdl/dist packages/ts/dist
build_dotnet: bin/lang-host

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

${GO_PB_SRC}: buf.gen.yaml ${PROTO_SRC}
	$(BUF) generate

packages/%/dist:
	bun run --cwd $(dir $@) build

%_suite_test.go:
	cd $(dir $@) && $(GINKGO) bootstrap

$(GO_SRC:%.go=%_test.go): %_test.go:
	cd $(dir $@) && $(GINKGO) generate $(notdir $*)

bin/ux: $(shell $(DEVCTL) list --go --exclude-tests)
	go -C cmd/ux build -o ${WORKING_DIR}/$@

bin/uml2uml: cmd/uml2uml/main.go
	go -C cmd/uml2uml build -o ${WORKING_DIR}/$@

bin/uml2ts: $(shell $(DEVCTL) list --ts --exclude-tests)
	bun build --cwd packages/uml2ts index.ts --compile --outfile ${WORKING_DIR}/$@

bin/zod2uml: $(shell $(DEVCTL) list --ts --exclude-tests)
	bun build --cwd packages/zod2uml index.ts --compile --outfile ${WORKING_DIR}/$@

bin/golangci-lint: .versions/golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${LOCALBIN} v$(shell cat $<)

bin/crd2pulumi: .versions/crd2pulumi
	curl -sSL https://github.com/pulumi/crd2pulumi/releases/download/v$(shell cat $<)/crd2pulumi-v$(shell cat $<)-$(shell go env GOOS)-$(shell go env GOARCH).tar.gz | tar -zxv -C bin crd2pulumi

bin/lang-host: | src/Lang.Host/bin/${DOTNET_CONF}/${DOTNET_TFM}/UnMango.Tdl.Lang.Host
	ln -s ${CURDIR}/$< ${CURDIR}/$@

src/Lang.Host/bin/${DOTNET_CONF}/${DOTNET_TFM}/UnMango.Tdl.Lang.Host: $(shell $(DEVCTL) list --cs) | bin/devctl
	dotnet build --configuration ${DOTNET_CONF}

.envrc: hack/example.envrc
	cp $< $@

buf.yaml:
	$(BUF) config init

buf.lock:
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

.make/go_test: ${GO_SRC} | bin/ux bin/uml2ts bin/uml2uml
	$(GINKGO) run ${TEST_FLAGS} $(sort $(dir $?))
	@touch $@

.make/go_e2e_test: ${GO_SRC} |  bin/ux bin/uml2ts bin/uml2uml
	$(GINKGO) run --label-filter 'E2E' $(sort $(dir $?))
	@touch $@

.make/go_lint: ${GO_SRC} | bin/golangci-lint
	$(GOLANGCI) run --config .golangci.yml $(sort $(dir $(filter %.go,$?)))
	@touch $@

.make/ts_test: ${TS_SRC}
	bun test --cwd packages/ts
	@touch $@

.make/buf_build: buf.yaml ${PROTO_SRC}
	$(BUF) build
	@touch $@

.make/buf_lint: buf.yaml ${PROTO_SRC}
	$(BUF) lint
	@touch $@

.make/buf_format: buf.yaml ${PROTO_SRC}
	$(BUF) format --write
	@touch $@

.make/dprint:
	dprint fmt

.make/go_fmt: ${GO_SRC}
	go fmt $(addprefix ./,$(sort $(dir $?)))
	@touch $@
