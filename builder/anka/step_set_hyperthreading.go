package anka

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
	"github.com/veertuinc/packer-builder-veertu-anka/util"
)

// StepSetHyperThreading will be used to enable or disbale hyperthreading
type StepSetHyperThreading struct{}

// Run will configure the hyperthreaded settings on the virtuals machines
func (s *StepSetHyperThreading) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	util := state.Get("util").(util.Util)
	onError := func(err error) multistep.StepAction {
		return util.StepError(ui, state, err)
	}
	cmdClient := state.Get("client").(client.Client)
	vmName := state.Get("vm_name").(string)
	rerun := false

	if !config.EnableHtt && !config.DisableHtt {
		log.Printf("enable/disable htt not specified. moving on.")
		return multistep.ActionContinue
	}

	if config.EnableHtt && config.DisableHtt {
		return onError(fmt.Errorf("Conflicting setting enable_htt and disable_htt both true"))
	}

	stopParams := client.StopParams{VMName: vmName, Force: true}

	describeResponse, err := cmdClient.Describe(vmName)
	if err != nil {
		return onError(err)
	}
	if describeResponse.VCPU.Threads > 0 && config.EnableHtt {
		log.Print("Htt already on")
		return multistep.ActionContinue
	}
	if describeResponse.VCPU.Threads == 0 && config.DisableHtt {
		log.Print("Htt already off")
		return multistep.ActionContinue
	}

	showResponse, err := cmdClient.Show(vmName)
	if err != nil {
		return onError(err)
	}
	if showResponse.IsRunning() {
		rerun = true
	}
	if !showResponse.IsStopped() {
		err := cmdClient.Stop(stopParams)
		if err != nil {
			return onError(err)
		}
	}

	if config.EnableHtt {
		err = cmdClient.Modify(vmName, "set", "cpu", "--htt")
	}

	if config.DisableHtt {
		err = cmdClient.Modify(vmName, "set", "cpu", "--no-htt")
	}

	if err != nil {
		return onError(err)
	}

	if rerun {
		err := cmdClient.Start(client.StartParams{VMName: vmName})
		if err != nil {
			return onError(err)
		}
	}

	return multistep.ActionContinue
}

// Cleanup cleans up the step when errors happen
// nothing to do here since the step will just modify an existing vm
func (s *StepSetHyperThreading) Cleanup(state multistep.StateBag) {
}
