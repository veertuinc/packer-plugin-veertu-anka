package anka

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
)

type StepStartVM struct{}

func (s *StepStartVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	onError := func(err error) multistep.StepAction {
		return stepError(ui, state, err)
	}
	cmdClient := state.Get("client").(*client.Client)
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

func (s *StepStartVM) Cleanup(state multistep.StateBag) {
	log.Print("Cleaning up start vm")
}
