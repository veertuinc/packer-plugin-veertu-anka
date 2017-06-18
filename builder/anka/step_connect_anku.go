package anka

import (
	"github.com/mitchellh/multistep"
)

// Borrowed from docker communicator, needs to be adapted still

type StepConnectAnka struct{}

func (s *StepConnectAnka) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)

	// containerId := state.Get("container_id").(string)
	// driver := state.Get("driver").(Driver)
	// tempDir := state.Get("temp_dir").(string)

	// // Get the version so we can pass it to the communicator
	// version, err := driver.Version()
	// if err != nil {
	// 	state.Put("error", err)
	// 	return multistep.ActionHalt
	// }

	// Create the communicator that talks to anka via os/exec tricks.
	comm := &Communicator{
		Config: config,
	}

	state.Put("communicator", comm)
	return multistep.ActionContinue
}

func (s *StepConnectAnka) Cleanup(state multistep.StateBag) {}
