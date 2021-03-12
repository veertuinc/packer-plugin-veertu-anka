package anka

import (
	"context"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
)

// StepConnectAnka attaches the anka builder to the communicator
type StepConnectAnka struct{}

// Run will add the ank client to the communicator and expose that via the state bag
func (s *StepConnectAnka) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	client := state.Get("client").(client.Client)
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

// Cleanup will run when any error happens
// nothing to do here since it just exposes a communicator
func (s *StepConnectAnka) Cleanup(state multistep.StateBag) {
}
