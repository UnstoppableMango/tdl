WORKING_DIR := $(shell pwd)
_ := $(shell mkdir -p .make bin)

export GOWORK :=

VERSION ?= $(shell dotnet minver --tag-prefix v --verbosity warn)
export MINVERVERSIONOVERRIDE = ${VERSION}

GO_MODULES = cli gen pkg

CFG ?=
ifeq ($(CFG),)
CFG := Debug
endif

export BIN_DIR := $(WORKING_DIR)/bin

DOTNET_NS := UnMango.Tdl
TFM       := net9.0
BIN_PATH  := bin/$(CFG)/$(TFM)

RUNNER_TEST := src/RunnerTest/$(BIN_PATH)/$(DOTNET_NS).RunnerTest.dll

SRC := $(shell git ls-files)
CS_SRC := $(filter %.cs,$(SRC))
FS_SRC := $(filter %.fs,$(SRC))
GO_SRC := $(filter %.go,$(SRC))
TS_SRC := $(filter %.ts,$(SRC))
JS_SRC := $(filter %.js,$(SRC))
PROTO_SRC := $(filter %.proto,$(SRC))

build: build_dotnet
build_dotnet: .make/build_dotnet

test: test_dotnet test_packages $(GO_MODULES:%=.make/test_%)
test_dotnet: .make/test_dotnet
test_packages: .make/test_packages

echo_test: go_echo_test ts_echo_test
go_echo_test: .make/go_echo_test
ts_echo_test: .make/ts_echo_test

e2e: .make/e2e

.PHONY: gen
gen: gen_proto

lint: .make/lint_proto lint_go lint_dotnet lint_packages
lint_go: $(GO_MODULES:%=.make/%_lint_go)
lint_dotnet: .make/lint_cs .make/lint_fs
lint_packages: .make/lint_packages

clean: clean_gen clean_dist
clean_gen:
	@find gen -mindepth 3 \
	-not -name 'package.json' \
	-not -name 'index.ts' \
	-ls -delete
clean_dist:
	@find . -type d -name dist \
	-not -path '*node_modules*' \
	-exec rm -rf '{}' + \
	-ls

tidy: $(GO_MODULES:%=.make/%_go_mod_tidy)
vet: $(GO_MODULES:%=.make/%_go_vet)

release:
	goreleaser release --snapshot --clean

.PHONY: proto
proto: build_proto gen_proto
gen_proto: .make/gen_proto
build_proto: .make/build_proto

version: .make/tool_restore
	@echo '${VERSION}'

dev: work .envrc
	@echo 'Ensuring development environment'
undev:
	@echo 'Tearing down dev environment'
	rm -f .envrc .make/regen_envrc

# The naming is kinda silly but its short
.PHONY: work
work: go.work go.work.sum

###################### Real targets ######################

bin/uml2ts: .make/gen_proto $(filter packages/uml2ts/%,$(TS_SRC))
	bun build packages/uml2ts/index.ts --compile --outfile $@

bin/um: .make/gen_proto $(filter cli/um/% cli/internal/%,$(GO_SRC))
	go build -C cli/um -o ${WORKING_DIR}/$@

bin/go_echo: $(GO_SRC)
	go -C cli/echo build -o ${WORKING_DIR}/$@

bin/ts_echo: $(TS_SRC)
	bun build packages/echo/index.ts --compile --outfile $@

$(RUNNER_TEST): $(filter src/RunnerTest,$(FS_SRC))
	dotnet build src/RunnerTest

go.work: GOWORK :=
go.work: $(GO_MODULES:%=%/go.mod)
	go work init
	go work use $(GO_MODULES)

go.work.sum: GOWORK :=
go.work.sum: go.work
	go work sync

.envrc: .make/regen_envrc
	echo '#!/bin/bash\nexport TDL_DEV=true' > .envrc

###################### Sentinal targets ######################

.make/build_proto: buf.work.yaml proto/buf.yaml $(PROTO_SRC)
	buf build
	@touch $@
.make/gen_proto: .make/build_proto buf.gen.yaml $(PROTO_SRC)
	buf generate
	@touch $@
.make/lint_proto: proto/buf.yaml $(PROTO_SRC)
	buf lint proto
	@touch $@

.make/tool_restore: .config/dotnet-tools.json
	dotnet tool restore
	@touch $@
.make/restore_dotnet: .make/tool_restore $(filter %.csproj %.fsproj,$(SRC))
	dotnet restore
	@touch $@
.make/lint_cs: .make/restore_dotnet $(CS_SRC)
	dotnet format analyzers \
	--no-restore \
	--verbosity diag \
	--exclude .make \
	--include $?
	@touch $@
.make/lint_fs: .make/tool_restore $(FS_SRC)
	dotnet fantomas $(filter-out .make/%,$?)
	@touch $@
.make/build_dotnet: .make/restore_dotnet .make/build_proto $(CS_SRC) $(FS_SRC)
	dotnet build
	@touch $@
.make/test_dotnet: $(CS_SRC) $(FS_SRC)
	dotnet test --no-build
	@touch $@

.make/lint_packages: $(TS_SRC) $(JS_SRC)
	bun eslint --no-warn-ignored $?
	@touch $@
.make/test_packages: bin/uml2ts $(TS_SRC) $(JS_SRC)
	bun test --timeout 10000 $?
	@touch $@

.make/test_cli: $(filter cli/%,$(GO_SRC))
.make/test_gen: $(filter gen/%,$(GO_SRC))
.make/test_pkg: $(filter pkg/%,$(GO_SRC))
$(GO_MODULES:%=.make/test_%): .make/test_%:
	go -C $* test ./... -ginkgo.timeout=5s
	@touch $@

.make/cli_lint_go: $(filter cli/%,$(GO_SRC))
.make/gen_lint_go: $(filter gen/%,$(GO_SRC))
.make/pkg_lint_go: $(filter pkg/%,$(GO_SRC))
$(GO_MODULES:%=.make/%_lint_go): .make/%_lint_go:
	golangci-lint run ./$*/...
	@touch $@

.make/cli_go_vet: $(filter cli/%,$(GO_SRC))
.make/gen_go_vet: $(filter gen/%,$(GO_SRC))
.make/pkg_go_vet: $(filter pkg/%,$(GO_SRC))
$(GO_MODULES:%=.make/%_go_vet): .make/%_go_vet:
	go -C $* vet ./...
	@touch $@

.make/cli_go_mod_tidy: $(filter cli/%,$(GO_SRC))
.make/gen_go_mod_tidy: $(filter gen/%,$(GO_SRC))
.make/pkg_go_mod_tidy: $(filter pkg/%,$(GO_SRC))
$(GO_MODULES:%=.make/%_go_mod_tidy): .make/%_go_mod_tidy: %/go.mod %/go.sum
	go -C $* mod tidy
	@touch $@

.make/go_echo_test: bin/go_echo $(RUNNER_TEST)
	@dotnet ${RUNNER_TEST} bin/go_echo
	@touch $@

.make/ts_echo_test: bin/ts_echo $(RUNNER_TEST)
	@dotnet ${RUNNER_TEST} bin/ts_echo
	@touch $@

.make/e2e: bin/um bin/go_echo bin/ts_echo bin/uml2ts
	go -C cli/um test --ginkgo.focus 'End to end'

.make/regen_envrc:
	@touch $@
