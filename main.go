package main

import (
	"github.com/hashicorp/packer/packer/plugin"
	"github.com/veertuinc/packer-builder-veertu-anka/builder/anka"
)

func main() {
	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}
	server.RegisterBuilder(new(anka.Builder))
	server.Serve()
}
