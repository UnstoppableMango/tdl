WORKING_DIR := $(shell pwd)
_ := $(shell mkdir -p .make)

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
build: $(LANG_SRC) $(BROKER_SRC)
	@touch .make/build_lang
	dotnet build

test: build
	dotnet test --no-build

gen: clean_gen
	buf generate proto

lint: .make/lint_proto .make/lint_lang

clean:
	rm -rf .make
	@find ${WORKING_DIR} -depth \( -name 'bin' -o -name 'obj' \) -type d \
		-exec echo 'Removing: {}' \; \
		-exec rm -rf '{}' \;

.PHONY: tidy
tidy:
	cd gen && go mod tidy

clean_gen:
	find gen -mindepth 2 -not -name package.json -delete

$(BROKER_BIN): $(BROKER_SRC)
	dotnet build ${BROKER_DIR}

.make/tool_restore: .config/dotnet-tools.json
	dotnet tool restore
	@touch $@

.make/build_lang: $(LANG_SRC)
	dotnet build ${LANG_DIR}
	@touch $@

.make/lint_proto:
	buf lint proto
	@touch $@

.make/lint_lang: .make/tool_restore $(LANG_SRC)
	dotnet fantomas ${LANG_DIR}
	@touch $@
