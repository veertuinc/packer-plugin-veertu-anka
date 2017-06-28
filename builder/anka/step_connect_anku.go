package anka

import (
	"github.com/lox/packer-builder-veertu-anka/client"
	"github.com/mitchellh/multistep"
)

type StepConnectAnka struct{}

func (s *StepConnectAnka) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	client := state.Get("client").(*client.Client)
	tempDir := state.Get("temp_dir").(string)
	vmName := state.Get("vm_name").(string)

	comm := &Communicator{
		Config:  config,
		Client:  client,
		HostDir: tempDir,
		VMDir:   "/packer-files",
		VMName:  vmName,
	}

	state.Put("communicator", comm)
	return multistep.ActionContinue
}

func (s *StepConnectAnka) Cleanup(state multistep.StateBag) {}
