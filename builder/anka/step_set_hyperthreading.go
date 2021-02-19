package anka

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
)

type StepSetHyperThreading struct{}

func (s *StepSetHyperThreading) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	onError := func(err error) multistep.StepAction {
		return stepError(ui, state, err)
	}

	if !config.EnableHtt && !config.DisableHtt { // configuration did not set enable or disable hyper threading
		log.Printf("enable/disable htt not specified. moving on.")
		return multistep.ActionContinue
	}
	if config.EnableHtt && config.DisableHtt { // just doesn't make any sense
		return onError(fmt.Errorf("Conflicting setting enable_htt and disable_htt both true"))
	}

	cmdClient := state.Get("client").(client.Client)
	vmName := state.Get("vm_name").(string)

	stopParams := client.StopParams{VMName: vmName, Force: true}

	describeResponse, err := cmdClient.Describe(vmName)
	if err != nil {
		return onError(err)
	}
	if describeResponse.CPU.Threads > 0 && config.EnableHtt {
		log.Print("Htt already on")
		return multistep.ActionContinue
	}
	if describeResponse.CPU.Threads == 0 && config.DisableHtt {
		log.Print("Htt already off")
		return multistep.ActionContinue
	}

	rerun := false
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

func (s *StepSetHyperThreading) Cleanup(state multistep.StateBag) {
	// nothing to do here!
}
