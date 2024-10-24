_ := $(shell mkdir -p .make)
WORKING_DIR := $(shell git rev-parse --show-toplevel)

PROJECT := tdl
MODULE  := github.com/unstoppablemango/${PROJECT}

LOCALBIN := ${WORKING_DIR}/bin
BUF      := ${LOCALBIN}/buf

GO_SRC    := $(shell find . -name '*.go')
PROTO_SRC := $(shell find . -name '*.proto')

build: bin/ux .make/buf_build

tidy: go.sum gen/go.sum go.work.sum

clean:
	rm bin/ux

bin/ux: $(filter cmd/ux/%,${GO_SRC})
	go -C cmd/ux build -o ${WORKING_DIR}/$@

bin/buf: .versions/buf
	GOBIN=${LOCALBIN} go install github.com/bufbuild/buf/cmd/buf@v$(shell cat $<)

.envrc: hack/example.envrc
	cp $< $@

buf.yaml: | bin/buf
	$(BUF) config init

go.mod:
	go mod init ${MODULE}

%/go.mod:
	go -C $(dir $@) mod init ${MODULE}/$*

go.sum: go.mod
	go mod tidy && touch $@

%/go.sum: %/go.mod ${GO_SRC}
	go -C $(dir $@) mod tidy && touch $@

go.work: go.mod gen/go.mod pkg/go.mod
	rm $@ && go work init $(dir $^)

go.work.sum: go.work go.mod gen/go.mod pkg/go.mod
	go work sync && touch $@

.make/buf_build: ${PROTO_SRC} | bin/buf
	$(BUF) build
	@touch $@
