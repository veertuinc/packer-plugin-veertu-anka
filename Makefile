LATEST-GIT-SHA := $(shell git rev-parse HEAD)
VERSION := $(shell cat VERSION)
FLAGS := -X main.commit=$(LATEST-GIT-SHA) -X main.version=$(VERSION)
BIN := packer-plugin-veertu-anka
ARCH := amd64
OS_TYPE ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
BIN_FULL := bin/$(BIN)_$(OS_TYPE)_$(ARCH)

.PHONY: go.lint lint go.test test clean anka.clean-images

.DEFAULT_GOAL := help

all: lint go.lint go.hcl2spec clean go.build go.test anka.clean-images anka.clean-clones

#help:	@ List available tasks on this project
help:
	@grep -h -E '[a-zA-Z\.\-]+:.*?@ .*$$' $(MAKEFILE_LIST) | sort | tr -d '#' | awk 'BEGIN {FS = ":.*?@ "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

#go.lint:		@ Run `golangci-lint run` against the current code
go.lint:
	golangci-lint run --fast

#go.test:		@ Run `go test` against the current tests
go.test:
	go test -v builder/anka/*.go
	go test -v post-processor/ankaregistry/*.go

#go.hcl2spec:		@ Run `go generate` to generate hcl2 config specs
go.hcl2spec:
	GOOS=$(OS_TYPE) go install github.com/hashicorp/packer/cmd/mapstructure-to-hcl2
	GOOS=$(OS_TYPE) PATH="$(shell pwd):${PATH}" go generate builder/anka/config.go
	GOOS=$(OS_TYPE) PATH="$(shell pwd):${PATH}" go generate post-processor/ankaregistry/post-processor.go

#go.build:		@ Run `go build` to generate the binary
go.build: $(BIN)
$(BIN): go.hcl2spec
	GOARCH=$(ARCH) go build $(RACE) -ldflags "$(FLAGS)" -o $(BIN_FULL)
	chmod +x $(BIN_FULL)

#lint:  @ Run `packer validate` against packer definitions
lint:
	packer validate examples/create-from-installer.pkr.hcl
	packer validate examples/create-from-installer-with-post-processing.pkr.hcl
	packer validate examples/clone-existing.pkr.hcl
	packer validate examples/clone-existing-with-post-processing.pkr.hcl
	packer validate examples/clone-existing-with-port-forwarding-rules.pkr.hcl
	packer validate examples/clone-existing-with-hwuuid.pkr.hcl
	packer validate examples/clone-existing-with-expect-disconnect.pkr.hcl
	packer validate examples/clone-existing-with-use-anka-cp.pkr.hcl

#install:		@ Copy the binary to the packer plugins folder
install:
	mkdir -p ~/.packer.d/plugins/
	cp -f $(BIN_FULL) ~/.packer.d/plugins/$(BIN)

#build-and-install:		@ Run make targets to setup the initialize the binary
build-and-install:
	$(MAKE) clean
	$(MAKE) go.build
	$(MAKE) install

#build-linux:		@ Run go.build for Linux
build-linux:
	GOOS=linux OS_TYPE=linux $(MAKE) go.build

#build-mac:		@ Run go.build for macOS
build-mac:
	GOOS=darwin OS_TYPE=darwin RACE="-race" $(MAKE) go.build

#create-test:		@ Run `packer build` with the default .pkr.hcl file
create-test: lint install
	PACKER_LOG=1 packer build examples/create-from-installer.pkr.hcl

#clean:		@ Remove the plugin binary
clean:
	rm -f $(BIN_FULL)

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