package main

import (
	"fmt"
	"log"
	"os"

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

	pps := plugin.NewSet()
	pps.RegisterBuilder(plugin.DEFAULT_NAME, new(anka.Builder))
	if err := pps.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
