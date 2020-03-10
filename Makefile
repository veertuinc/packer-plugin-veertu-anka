PREFIX := github.com/veertuinc/packer-builder-veertu-anka
VERSION := $(shell git describe --tags --candidates=1 --dirty 2>/dev/null || echo "dev")
FLAGS := -X main.Version=$(VERSION)
BIN := packer-builder-veertu-anka
SOURCES := $(shell find . -name '*.go')

.PHONY: test packer-test clean clean-images

test:
	govendor test +l

build: $(BIN)

$(BIN): $(SOURCES)
	GOOS=darwin GOBIN=$(shell pwd) go install github.com/hashicorp/packer/cmd/mapstructure-to-hcl2
	GOOS=darwin PATH=$(shell pwd):${PATH} go generate builder/anka/config.go
	GOOS=darwin go build -ldflags="$(FLAGS)" -o $(BIN) $(PREFIX)

install: $(BIN)
	mkdir -p ~/.packer.d/plugins/
	cp $(BIN) ~/.packer.d/plugins/

packer-test: install
	PACKER_LOG=1 packer build -var "source_vm_name=$(SOURCE_VM_NAME)" examples/macos-sierra.json

clean:
	rm -f $(BIN)

clean-images:
	anka --machine-readable list | jq '.body[].name' | grep anka-packer | xargs -n1 anka delete --force

wipe-anka:
	-rm -rf ~/Library/Application\ Support/Veertu
	-rm -rf ~/.anka
