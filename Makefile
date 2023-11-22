WORKDIR := $(shell pwd)
LATEST-GIT-SHA := $(shell git rev-parse HEAD)
VERSION := $(shell cat VERSION)
FLAGS := -X main.commit=$(LATEST-GIT-SHA) -X main.version=$(VERSION)
BIN := packer-plugin-veertu-anka
ARCH := $(shell arch)
ifeq ($(ARCH), i386)
	ARCH = amd64
endif
PACKER_CI_PROJECT_API_VERSION?=$(shell go run . describe 2>/dev/null | jq -r '.api_version')
OS_TYPE ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
BIN_FULL ?= dist/$(BIN)_v$(VERSION)_$(PACKER_CI_PROJECT_API_VERSION)_$(OS_TYPE)_$(ARCH)
HASHICORP_PACKER_PLUGIN_SDK_VERSION?=$(shell go list -m github.com/hashicorp/packer-plugin-sdk | cut -d " " -f2)
export PATH := $(shell go env GOPATH)/bin:$(PATH)

.PHONY: go.lint validate-examples go.test test clean anka.clean-images

.DEFAULT_GOAL := help

all: clean go.releaser anka.clean-images anka.clean-clones generate-docs install

#help:	@ List available tasks on this project
help:
	@grep -h -E '[a-zA-Z\.\-]+:.*?@ .*$$' $(MAKEFILE_LIST) | sort | tr -d '#' | awk 'BEGIN {FS = ":.*?@ "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

#go.lint:		@ Run `golangci-lint run` against the current code
go.lint:
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b /usr/local/bin v1.40.1
	golangci-lint run --fast

#go.test:		@ Install modules, mockgen, generate mocks, and run `go test` against the current tests
# removed: go mod tidy -go=1.16 && go mod tidy -go=1.17
go.test:
	go mod tidy
	go install github.com/golang/mock/mockgen@v1.6.0
	mockgen -source=util/util.go -destination=mocks/util_mock.go -package=mocks
	mockgen -source=client/client.go -destination=mocks/client_mock.go -package=mocks
	go test -v builder/anka/*.go
	go test -v post-processor/ankaregistry/*.go

#go.build:		@ Run `go build` to generate the binary
go.build:
	GOARCH=$(ARCH) go build $(RACE) -ldflags "$(FLAGS)" -o $(BIN_FULL)
	chmod +x dist/$(BIN)*

#go.releaser 	@ Run goreleaser release --clean for current version
go.releaser:
	git tag -d "$(VERSION)" 2>/dev/null || true
	git tag -a "$(VERSION)" -m "Version $(VERSION)"
	echo "LATEST TAG: $$(git describe --tags --abbrev=0)"
	PACKER_CI_PROJECT_API_VERSION=$(PACKER_CI_PROJECT_API_VERSION) goreleaser release --clean

#validate-examples:  @ Run `packer validate` against example packer definitions using the built package
validate-examples:
	cp -rfp $(WORKDIR)/examples /tmp/
	for file in $$(ls $(WORKDIR)/examples/ | grep hcl); do echo $$file; packer validate $(WORKDIR)/examples/$$file; done

#install:		@ Copy the binary to the packer plugins folder
install:
	mkdir -p ~/.packer.d/plugins/
	cp -f $(BIN_FULL) ~/.packer.d/plugins/$(BIN)

#uninstall:		@ Delete the binary from the packer plugins folder
uninstall:
	packer plugins remove github.com/veertuinc/veertu-anka
	rm -f ~/.packer.d/plugins/$(BIN)*

#build-and-install:		@ Run make targets to setup the initialize the binary
build-and-install:
	$(MAKE) clean
	$(MAKE) go.build
	$(MAKE) go.hcl2spec
	$(MAKE) install

#build-linux:		@ Run go.build for Linux
build-linux:
	GOOS=linux OS_TYPE=linux $(MAKE) go.build

#build-mac:		@ Run go.build for macOS
build-mac:
	GOOS=darwin OS_TYPE=darwin $(MAKE) go.build

#create-test:		@ Run `packer build` with the default .pkr.hcl file
create-test: lint install
	PACKER_LOG=1 packer build $(WORKDIR)/examples/create-from-installer.pkr.hcl

#clean:		@ Remove the plugin binary
clean:
	$(MAKE) uninstall
	rm -f docs.zip
	rm -rf dist

#anka.clean-images:		@ Remove all anka images with `anka delete`
anka.clean-images:
	anka --machine-readable list | jq -r '.body[].name' | grep anka-packer | xargs -n1 anka delete --yes

#anka.clean-clones:		@ Remove all anka clones with `anka delete`
anka.clean-clones:
	anka --machine-readable list | jq -r '.body[].name' | grep anka-packer | grep -v base | xargs -n1 anka delete --yes

#anka.wipe-anka:		@ Remove all anka images from the local library
anka.wipe-anka:
	-rm -rf ~/Library/Application\ Support/Veertu
	-rm -rf ~/.anka

#install-packer-sdc:	@ Install the hashicorp packer sdc
install-packer-sdc: ## Install packer sofware development command
	@go install github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc@${HASHICORP_PACKER_PLUGIN_SDK_VERSION}

#go.hcl2spec:		@ Run `go generate` to generate hcl2 config specs
go.hcl2spec: install-packer-sdc
	GOOS=$(OS_TYPE) go generate builder/anka/config.go
	GOOS=$(OS_TYPE) go generate post-processor/ankaregistry/post-processor.go

#generate-docs:		@ Generate packer docs
generate-docs: install-packer-sdc
	@pushd dist/; packer-sdc renderdocs -src ../docs -partials docs-partials/ -dst docs/ && /bin/sh -c "[ -d docs ] && zip -r docs.zip docs/"

build-docs: install-packer-sdc
	@if [ -d ".docs" ]; then rm -r ".docs"; fi
	@packer-sdc renderdocs -src "docs" -partials docs-partials/ -dst ".docs/"
	@./.web-docs/scripts/compile-to-webdocs.sh "." ".docs" ".web-docs" "veertuinc"
	@rm -r ".docs"
