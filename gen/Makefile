export GOWORK :=

.PHONY: go proto_go tidy
go: proto_go

proto_go:
	cd proto/go && go build ./...

tidy:
	go mod tidy

clean:
	@echo 'Cleaning sources...'
	@find . -mindepth 3 \
		-not -name 'package.json' \
		-not -name 'index.ts' \
		-ls -delete
