package anka

import (
	"fmt"

	"github.com/hashicorp/packer/packer"
	"github.com/mitchellh/multistep"
)

type StepCreateDisk struct{}

func (s *StepCreateDisk) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	client := state.Get("client").(*Client)
	ui := state.Get("ui").(packer.Ui)

	ui.Say(fmt.Sprintf("Creating disk image from app: %s", config.InstallerApp))

	err := client.CreateDisk(config.DiskSize, config.InstallerApp)
	if err != nil {
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *StepCreateDisk) Cleanup(state multistep.StateBag) {
}
