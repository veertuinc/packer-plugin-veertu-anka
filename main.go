package main

import (
	"github.com/buildkite/packer-builder-veertu-anka/builder/anka"
	"github.com/mitchellh/packer/packer/plugin"
)

func main() {
	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}
	server.RegisterBuilder(new(anka.Builder))
	server.Serve()
}
