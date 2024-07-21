WORKING_DIR := $(shell pwd)
_ := $(shell mkdir -p .make bin)

export GOWORK :=

VERSION ?= $(shell dotnet minver --tag-prefix v --verbosity warn)
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

RUNNER_TEST_DIR := src/RunnerTest
RUNNER_TEST_SRC := $(shell find $(RUNNER_TEST_DIR) -name '*.fs' -not -path '*obj*' -type f)
RUNNER_TEST_BIN := $(RUNNER_TEST_DIR)/$(BIN_PATH)/$(NS).RunnerTest.dll

GO_ECHO_SRC := $(shell find cli/echo -type f -name '*.go')
GO_ECHO_CLI := bin/go_echo

TS_ECHO_SRC := $(shell find packages/echo -type f -name '*.ts')
TS_ECHO_CLI := bin/ts_echo

.PHONY: build build_dotnet
build: build_dotnet cli docker pkg
	@touch .make/build_lang
build_dotnet: .make/build_dotnet

.PHONY: test test_dotnet test_pkg test_packages e2e
test: test_dotnet test_pkg test_packages
test_dotnet: build_dotnet
	dotnet test --no-build
test_pkg:
	@$(MAKE) -C pkg test
test_packages:
	@$(MAKE) -C packages test
echo_test: go_echo_test ts_echo_test
go_echo_test: $(GO_ECHO_CLI) $(RUNNER_TEST_BIN)
	@dotnet ${RUNNER_TEST_BIN} ${GO_ECHO_CLI}
ts_echo_test: $(TS_ECHO_CLI) $(RUNNER_TEST_BIN)
	@dotnet ${RUNNER_TEST_BIN} ${TS_ECHO_CLI}
e2e: export BIN_DIR := $(WORKING_DIR)/bin
e2e: bin/um $(GO_ECHO_CLI) $(TS_ECHO_CLI) bin/uml2ts
	go run -C e2e/cmd ./...

.PHONY: gen
gen: gen_proto

.PHONY: lint
lint: .make/lint_proto .make/lint_dotnet

.PHONY: clean clean_gen clean_src clean_dist
clean: clean_gen clean_src clean_dist
	rm -rf .make bin
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
	@$(MAKE) -C cli tidy --no-print-directory
	@$(MAKE) -C e2e tidy --no-print-directory
	@$(MAKE) -C gen tidy --no-print-directory
	@$(MAKE) -C pkg tidy --no-print-directory

.PHONY: release
release:
	goreleaser release --snapshot --clean

.PHONY: proto gen_proto build_proto
proto: build_proto gen_proto
gen_proto: .make/gen_proto
build_proto: .make/build_proto

.PHONY: cli docker pkg src
cli:
	@$(MAKE) -C cli
docker:
	@$(MAKE) -C docker
pkg:
	@$(MAKE) -C pkg
src:
	@$(MAKE) -C src

.PHONY: version
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

###################### Real targets ######################

bin/uml2ts: .make/gen_proto $(shell find packages/uml2ts -type f -path '*.ts')
	bun build packages/uml2ts/index.ts --compile --outfile $@

bin/um: .make/gen_proto $(shell find cli/um cli/internal -type f -path '*.go')
	go build -C cli/um -o ${WORKING_DIR}/$@

$(GO_ECHO_CLI):
	@$(MAKE) -C cli/echo build --no-print-directory

$(TS_ECHO_CLI): $(TS_ECHO_SRC)
	@$(MAKE) -C packages/echo --no-print-directory

$(RUNNER_TEST_BIN): $(RUNNER_TEST_SRC)
	dotnet build ${RUNNER_TEST_DIR}

go.work: GOWORK :=
go.work:
	go work init
	go work use cli
	go work use e2e
	go work use gen
	go work use pkg

go.work.sum: GOWORK :=
go.work.sum: go.work
	go work sync

.envrc: .make/regen_envrc
	echo '#!/bin/bash\nexport TDL_DEV=true' > .envrc

###################### Sentinal targets ######################

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

.make/lint_dotnet: .make/lint_lang .make/lint_broker
.make/lint_lang: .make/tool_restore $(LANG_SRC)
	dotnet fantomas ${LANG_DIR}
	@touch $@
.make/lint_broker: $(BROKER_SRC)
	dotnet format --include ${BROKER_SRC} --verify-no-changes
	@touch $@

.make/regen_envrc:
	@touch $@
