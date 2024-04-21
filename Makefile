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

LANG_DIR := src/Language
LANG_SRC := $(shell find $(LANG_DIR) -name '*.fs' -not -path '*obj*' -type f)

.PHONY: build test clean
build: $(LANG_SRC) $(BROKER_SRC)
	@touch .make/build_lang
	dotnet build

test: build
	dotnet test --no-build

clean:
	rm -rf .make
	@find ${WORKING_DIR} -depth \( -name 'bin' -o -name 'obj' \) -type d \
		-exec echo 'Removing: {}' \; \
		-exec rm -rf '{}' \;

$(BROKER_BIN): $(BROKER_SRC)
	dotnet build ${BROKER_DIR}

.make/build_lang: $(LANG_SRC)
	dotnet build ${LANG_DIR}
	@touch $@

.make/tool_restore: .config/dotnet-tools.json
	dotnet tool restore
	@touch $@
