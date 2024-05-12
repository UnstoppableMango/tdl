WORKING_DIR := $(shell pwd)
_ := $(shell mkdir -p .make)

export GOWORK := off

VERSION := $(shell dotnet minver --tag-prefix v --verbosity warn)
export MINVERVERSIONOVERRIDE = ${VERSION}

CFG ?=
ifeq ($(CFG),)
CFG := Debug
endif

NS       := UnMango.Tdl
TFM      := net9.0
BIN_PATH := bin/$(CFG)/$(TFM)

BROKER_DIR := src/Broker
BROKER_SRC := $(shell find $(BROKER_DIR) -name '*.cs' -not -path '*obj*' -type f)
BROKER_BIN := $(BROKER_DIR)/$(BIN_PATH)/$(NS).Broker.dll

CLI_DIR := src/Cli
CLI_SRC := $(shell find $(CLI_DIR) -name '*.cs' -not -path '*obj*' -type f)
CLI_BIN := $(CLI_DIR)/$(BIN_PATH)/um.dll

LANG_DIR := src/Language
LANG_SRC := $(shell find $(LANG_DIR) -name '*.fs' -not -path '*obj*' -type f)

.PHONY: build build_dotnet
build: build_dotnet
	@touch .make/build_lang
build_dotnet: .make/build_dotnet

.PHONY: test test_dotnet
test: test_dotnet test_packages
test_dotnet: build_dotnet
	dotnet test --no-build
test_packages:
	@$(MAKE) -C packages test

.PHONY: gen
gen: gen_proto

.PHONY: lint
lint: .make/lint_proto .make/lint_lang

.PHONY: clean clean_gen clean_src clean_dist
clean: clean_gen clean_src clean_dist
	rm -rf .make
clean_cli:
	@$(MAKE) -C cli clean
clean_gen:
	@$(MAKE) -C gen clean
clean_src:
	@$(MAKE) -C src clean
clean_dist:
	@find . -type d -name dist \
		-not -path '*node_modules*' \
		-exec rm -rf '{}' + \
		-ls

.PHONY: tidy
tidy: gen
	@$(MAKE) -C cli tidy
	@$(MAKE) -C gen tidy
	@$(MAKE) -C pkg tidy

.PHONY: release
release:
	goreleaser release --snapshot --clean

.PHONY: proto gen_proto build_proto
proto: build_proto gen_proto
gen_proto: .make/gen_proto
build_proto: .make/build_proto

.PHONY: docker
docker:
	@$(MAKE) -C docker all

version:
	@echo '${VERSION}'

.PHONY: dev undev
dev: work .envrc
	@echo 'Ensuring development environment'
undev:
	@echo 'Tearing down dev environment'
	rm -f .envrc .make/regen_envrc

# The naming is kinda silly but its short
.PHONY: work
work: go.work go.work.sum

go.work: GOWORK :=
go.work:
	go work init
	go work use cli
	go work use gen
	go work use pkg

go.work.sum: GOWORK :=
go.work.sum: go.work
	go work sync

.envrc: .make/regen_envrc
	echo 'export TDL_DEV=true' > .envrc

.make/tool_restore: .config/dotnet-tools.json
	dotnet tool restore
	@touch $@

PROTO_SRC := $(shell find proto -type f -name '*.proto')
.make/gen_proto: .make/build_proto $(PROTO_SRC)
	buf generate
	@touch $@
.make/build_proto: $(PROTO_SRC)
	buf build
	@touch $@
.make/lint_proto: $(PROTO_SRC)
	buf lint proto
	@touch $@

.make/build_dotnet: $(LANG_SRC) $(BROKER_SRC) $(CLI_SRC) .make/gen_proto
	dotnet build
	@touch $@ .make/build_lang .make/build_broker .make/build_cli
.make/build_lang: $(LANG_SRC)
	dotnet build ${LANG_DIR}
	@touch $@
.make/build_broker: $(BROKER_SRC) .make/gen_proto
	dotnet build ${BROKER_DIR}
	@touch $@
.make/build_cli: $(CLI_SRC) .make/gen_proto
	dotnet build ${CLI_DIR}
	@touch $@

.make/lint_lang: .make/tool_restore $(LANG_SRC)
	dotnet fantomas ${LANG_DIR}
	@touch $@

.make/regen_envrc:
	@touch $@
