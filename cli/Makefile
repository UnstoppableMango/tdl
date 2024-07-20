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

tidy:
	go mod tidy
