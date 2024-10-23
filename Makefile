WORKING_DIR := $(shell git rev-parse --show-toplevel)

PROJECT := tdl
MODULE  := github.com/unstoppablemango/${PROJECT}

GO_SRC := $(shell find . -name '*.go')

build: bin/ux

tidy: go.sum gen/go.sum go.work.sum

clean:
	rm bin/ux

bin/ux: $(filter cmd/ux/%,${GO_SRC})
	go -C cmd/ux build -o ${WORKING_DIR}/$@

.envrc: hack/example.envrc
	cp $< $@

go.mod:
	go mod init ${MODULE}

%/go.mod:
	go -C $(dir $@) mod init ${MODULE}/$*

go.sum: go.mod
	go mod tidy && touch $@

%/go.sum: %/go.mod $(shell find $* -name '*.go')
	go -C $(dir $@) mod tidy && touch $@

go.work: go.mod gen/go.mod pkg/go.mod
	rm $@ && go work init $(dir $^)

go.work.sum: go.work go.mod gen/go.mod pkg/go.mod
	go work sync && touch $@
