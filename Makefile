WORKING_DIR := $(shell pwd)
_ := $(shell mkdir -p .make)

export GOWORK := off

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

.PHONY: build test gen lint clean
build: $(LANG_SRC) $(CLI_SRC) $(BROKER_SRC) build_proto
	@touch .make/build_lang
	dotnet build

test: build
	dotnet test --no-build

gen: clean_gen build_proto
	buf generate

lint: .make/lint_proto .make/lint_lang

clean: clean_gen clean_dist
	rm -rf .make
	@$(MAKE) -C src clean

clean_gen:
	@$(MAKE) -C gen clean

clean_dist:
	@find . -type d -name dist \
		-not -path '*node_modules*' \
		-exec echo 'Removing: {}' \; \
		-exec rm -rf '{}' +

.PHONY: tidy
tidy: gen
	@$(MAKE) -C cli tidy
	@$(MAKE) -C gen tidy
	@$(MAKE) -C pkg tidy

.PHONY: build_proto
build_proto:
	buf build

$(BROKER_BIN): $(BROKER_SRC)
	dotnet build ${BROKER_DIR}

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

.make/tool_restore: .config/dotnet-tools.json
	dotnet tool restore
	@touch $@

.make/build_lang: $(LANG_SRC)
	dotnet build ${LANG_DIR}
	@touch $@

.make/build_cli: $(CLI_SRC)
	dotnet build ${CLI_DIR}
	@touch $@

.make/lint_proto:
	buf lint proto
	@touch $@

.make/lint_lang: .make/tool_restore $(LANG_SRC)
	dotnet fantomas ${LANG_DIR}
	@touch $@

.make/build_plugin_gen_ts:
	cd plugin/gen/ts && bun run build
	@touch $@
