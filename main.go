package main

import (
	"log"

	"github.com/hashicorp/packer-plugin-sdk/plugin"
	"github.com/veertuinc/packer-builder-veertu-anka/builder/anka"
)

var version = "SNAPSHOT"
var commit = ""

func main() {
	if commit == "" {
		log.Printf("packer-builder-veertu-anka version: %s", version)
	} else {
		log.Printf("packer-builder-veertu-anka version: %s+%s", version, commit)
	}
	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}
	server.RegisterBuilder(new(anka.Builder))
	server.Serve()
}
