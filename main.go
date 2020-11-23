package main

import (
	"log"

	"github.com/hashicorp/packer/packer/plugin"
	"github.com/veertuinc/packer-builder-veertu-anka/builder/anka"
)

var Version = "SNAPSHOT"

func main() {
	log.Printf("packer-builder-veertu-anka version: %q", Version)
	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}
	server.RegisterBuilder(new(anka.Builder))
	server.Serve()
}
