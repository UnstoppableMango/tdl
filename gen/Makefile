.PHONY: go proto_go tidy
go: proto_go

proto_go:
	cd proto/go && go build ./...

tidy:
	go mod tidy
