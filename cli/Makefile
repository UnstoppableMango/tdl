_ := $(shell mkdir -p .make)

export GOWORK :=

.PHONY: all uml2go uml2pcl
all: uml2go uml2pcl

uml2go:
	@$(MAKE) -C uml2go build

uml2pcl:
	@$(MAKE) -C uml2pcl build

clean:
	@$(MAKE) -C uml2go clean
	@$(MAKE) -C uml2pcl clean

tidy: .make/go_mod_tidy

.make/go_mod_tidy: go.mod
	go mod tidy
	@touch $@
