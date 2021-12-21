LATEST-GIT-SHA := $(shell git rev-parse HEAD)
VERSION := $(shell cat VERSION)
FLAGS := -X main.commit=$(LATEST-GIT-SHA) -X main.version=$(VERSION)
BIN := packer-plugin-veertu-anka
ARCH := amd64
OS_TYPE ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
BIN_FULL ?= bin/$(BIN)_$(OS_TYPE)_$(ARCH)

.PHONY: go.lint validate-examples go.test test clean anka.clean-images

.DEFAULT_GOAL := help

all: go.lint clean go.hcl2spec go.build go.test install validate-examples anka.clean-images anka.clean-clones uninstall

#help:	@ List available tasks on this project
help:
	@grep -h -E '[a-zA-Z\.\-]+:.*?@ .*$$' $(MAKEFILE_LIST) | sort | tr -d '#' | awk 'BEGIN {FS = ":.*?@ "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

#go.lint:		@ Run `golangci-lint run` against the current code
go.lint:
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b /usr/local/bin v1.40.1
	golangci-lint run --fast

#go.test:		@ Run `go test` against the current tests
go.test:
	go mod tidy -go=1.16 && go mod tidy -go=1.17
	go install github.com/golang/mock/mockgen@v1.6.0
	mockgen -source=client/client.go -destination=mocks/client_mock.go -package=mocks
	go test -v builder/anka/*.go
	go test -v post-processor/ankaregistry/*.go

#go.hcl2spec:		@ Run `go generate` to generate hcl2 config specs
go.hcl2spec:
	go install github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc@latest
	GOOS=$(OS_TYPE) go generate builder/anka/config.go
	GOOS=$(OS_TYPE) go generate post-processor/ankaregistry/post-processor.go

#go.build:		@ Run `go build` to generate the binary
go.build:
	GOARCH=$(ARCH) go build $(RACE) -ldflags "$(FLAGS)" -o $(BIN_FULL)
	chmod +x $(BIN_FULL)

#validate-examples:  @ Run `packer validate` against example packer definitions using the built package
validate-examples:
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

#uninstall:		@ Delete the binary from the packer plugins folder
uninstall:
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

#generate-docs:		@ Generate packer docs
generate-docs:
	@go install github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc@latest
	@pushd dist/; packer-sdc renderdocs -src ../docs -partials docs-partials/ -dst docs/ && /bin/sh -c "[ -d docs ] && zip -r docs.zip docs/"