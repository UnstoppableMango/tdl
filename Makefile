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

build: build_dotnet cli docker pkg
build_dotnet: .make/build_dotnet

test: test_dotnet test_pkg test_packages
test_dotnet: build_dotnet
	dotnet test --no-build
test_pkg:
	@$(MAKE) -C pkg test
test_packages:
	@$(MAKE) -C packages test
echo_test: go_echo_test ts_echo_test
go_echo_test: bin/go_echo $(RUNNER_TEST)
	@dotnet ${RUNNER_TEST} bin/go_echo
ts_echo_test: bin/ts_echo $(RUNNER_TEST)
	@dotnet ${RUNNER_TEST} bin/ts_echo
e2e: export BIN_DIR := $(WORKING_DIR)/bin
e2e: bin/um bin/go_echo bin/ts_echo bin/uml2ts
	@$(MAKE) -C cli/um e2e

.PHONY: gen
gen: gen_proto

lint: .make/lint_proto .make/lint_dotnet

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

tidy: $(GO_MODULES:%=.make/%_go_mod_tidy)

release:
	goreleaser release --snapshot --clean

.PHONY: proto
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

version:
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

bin/go_echo:
	@$(MAKE) -C cli/echo build --no-print-directory

bin/ts_echo: $(filter packages/echo/%,$(TS_SRC))
	@$(MAKE) -C packages/echo --no-print-directory

$(RUNNER_TEST): $(filter src/RunnerTest,$(FS_SRC))
	dotnet build src/RunnerTest

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
	echo '#!/bin/bash\nexport TDL_DEV=true' > .envrc

###################### Sentinal targets ######################

.make/tool_restore: .config/dotnet-tools.json
	dotnet tool restore
	@touch $@

.make/gen_proto: .make/build_proto $(PROTO_SRC)
	buf generate
	@touch $@
.make/build_proto: $(PROTO_SRC)
	buf build
	@touch $@
.make/lint_proto: $(PROTO_SRC)
	buf lint proto
	@touch $@

.make/build_dotnet: .make/gen_proto $(CS_SRC) $(FS_SRC)
	dotnet build
	@touch $@ .make/build_cli
.make/build_cli: $(filter src/Cli/%,$(CS_SRC)) .make/gen_proto
	dotnet build src/Cli
	@touch $@

.make/regen_envrc:
	@touch $@

.make/cli_go_mod_tidy: $(filter cli/%,$(GO_SRC))
.make/gen_go_mod_tidy: $(filter gen/%,$(GO_SRC))
.make/pkg_go_mod_tidy: $(filter pkg/%,$(GO_SRC))
$(GO_MODULES:%=.make/%_go_mod_tidy): .make/%_go_mod_tidy: %/go.mod %/go.sum
	go -C $* mod tidy
	@touch $@
