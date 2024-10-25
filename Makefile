_ := $(shell mkdir -p .make)

FIND := find
ifeq ($(shell uname),Darwin)
FIND := gfind
endif

WORKING_DIR := $(shell git rev-parse --show-toplevel)

PROJECT := tdl
MODULE  := github.com/unstoppablemango/${PROJECT}

LOCALBIN := ${WORKING_DIR}/bin
BUF      := ${LOCALBIN}/buf

GO_SRC    := $(shell $(FIND) . -name '*.go' -not -path '*/node_modules/*')
PROTO_SRC := $(shell $(FIND) . -name '*.proto' -not -path '*/node_modules/*' -printf '%P\n')
GO_PB_SRC := ${PROTO_SRC:proto/%.proto=pkg/%.pb.go}

build: bin/ux .make/buf_build
generate: ${GO_PB_SRC}
lint: .make/buf_lint
tidy: go.sum

clean:
	rm bin/ux

${GO_PB_SRC}: buf.gen.yaml ${PROTO_SRC} | bin/buf
	$(BUF) generate

bin/ux: $(filter cmd/ux/%,${GO_SRC})
	go -C cmd/ux build -o ${WORKING_DIR}/$@

bin/buf: .versions/buf
	GOBIN=${LOCALBIN} go install github.com/bufbuild/buf/cmd/buf@v$(shell cat $<)

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

go.sum: go.mod
	go mod tidy && touch $@

.make/buf_build: buf.yaml ${PROTO_SRC} | bin/buf
	$(BUF) build
	@touch $@

.make/buf_lint: buf.yaml ${PROTO_SRC} | bin/buf
	$(BUF) lint
	@touch $@
