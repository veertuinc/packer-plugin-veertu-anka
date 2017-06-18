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
	imageID := config.ImageID

	if imageID == "" {
		ui.Say(fmt.Sprintf("Creating disk image from app: %s", config.InstallerApp))

		var err error
		imageID, err = client.CreateDisk(CreateDiskParams{
			DiskSize:     config.DiskSize,
			InstallerApp: config.InstallerApp,
		})
		if err != nil {
			return multistep.ActionHalt
		}

		ui.Say(fmt.Sprintf("Disk image %s was created", imageID))
	}

	vmId, err := client.CreateVM(CreateVMParams{
		ImageID:  imageID,
		RamSize:  "2G",
		CPUCount: 2,
		Name:     "anka-builder",
	})
	if err != nil {
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("VM %s was created", vmId))
	return multistep.ActionContinue
}

func (s *StepCreateDisk) Cleanup(state multistep.StateBag) {
}
