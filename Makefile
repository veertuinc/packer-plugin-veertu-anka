
LATEST-GIT-SHA := $(shell git rev-parse HEAD)
FLAGS := -X main.Version=$(LATEST-GIT-SHA)
BIN := packer-builder-veertu-anka
SOURCES := $(shell find . -name '*.go')

.PHONY: test packer-test clean clean-images

test:
	go test -v builder/anka/*.go

build: $(BIN)
$(BIN):
	GOOS=darwin GOBIN=$(shell pwd) go install github.com/hashicorp/packer/cmd/mapstructure-to-hcl2
	GOOS=darwin PATH="$(shell pwd):${PATH}" go generate builder/anka/config.go
	GOOS=darwin go build -ldflags="$(FLAGS)" -o $(BIN)

install: $(BIN)
	mkdir -p ~/.packer.d/plugins/
	cp $(BIN) ~/.packer.d/plugins/

build-and-install: $(BIN)
	$(MAKE) clean
	$(MAKE) build
	$(MAKE) install

packer-test: install
	PACKER_LOG=1 packer build -var "source_vm_name=$(SOURCE_VM_NAME)" examples/macos-sierra.json

packer-test2: build
	PACKER_LOG=1 packer build examples/macos-catalina.json

big-sur: install
	PACKER_LOG=1 packer build examples/macos-bigsur.json

clean:
	rm -f $(BIN)

clean-images:
	anka --machine-readable list | jq '.body[].name' | grep anka-packer | xargs -n1 anka delete --force

wipe-anka:
	-rm -rf ~/Library/Application\ Support/Veertu
	-rm -rf ~/.anka

release-dry-run: build test
	goreleaser --snapshot --skip-sign --skip-publish --rm-dist