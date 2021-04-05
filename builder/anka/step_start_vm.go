package anka

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
	"github.com/veertuinc/packer-builder-veertu-anka/util"
)

// StepStartVM will start the created/cloned vms
type StepStartVM struct{}

// Run will call the necessary client commands to start the vm
func (s *StepStartVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	util := state.Get("util").(util.Util)
	onError := func(err error) multistep.StepAction {
		return util.StepError(ui, state, err)
	}
	cmdClient := state.Get("client").(client.Client)
	vmName := state.Get("vm_name").(string)

	err := cmdClient.Start(client.StartParams{
		VMName: vmName,
	})
	if err != nil {
		return onError(err)
	}

	if config.BootDelay != "" {
		d, err := time.ParseDuration(config.BootDelay)
		if err != nil {
			return onError(err)
		}

		ui.Say(fmt.Sprintf("Waiting for %s for clone to boot", d))
		time.Sleep(d)
	}

	return multistep.ActionContinue
}

// Cleanup will run when errors occur
// Nothing to do here since this just this step just starts vms
func (s *StepStartVM) Cleanup(state multistep.StateBag) {
}
