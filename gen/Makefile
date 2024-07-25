_ := $(shell mkdir -p .make)

export GOWORK :=

.PHONY: go proto_go tidy
go: proto_go

proto_go:
	cd proto/go && go build ./...

tidy: .make/go_mod_tidy

clean:
	@echo 'Cleaning sources...'
	@find . -mindepth 3 \
		-not -name 'package.json' \
		-not -name 'index.ts' \
		-ls -delete

.make/go_mod_tidy: go.mod
	go mod tidy
	@touch $@
