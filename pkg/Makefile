.PHONY: build test
build: build_go
test: test_go

build_go:
	go build ./...

test_go:
	go test ./...

tidy:
	go mod tidy
