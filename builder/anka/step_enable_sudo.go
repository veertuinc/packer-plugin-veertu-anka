package anka

import (
	"github.com/hashicorp/packer/packer"
	"github.com/lox/packer-builder-veertu-anka/client"
	"github.com/mitchellh/multistep"
)

type StepEnableSudo struct {
	client *client.Client
	vmName string
}

func (s *StepEnableSudo) Run(state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)

	s.vmName = state.Get("vm_name").(string)
	s.client = state.Get("client").(*client.Client)

	ui.Say("Enabling sudo access for anka user in VM")
	err, _ := s.client.Run(client.RunParams{
		VMName:  s.vmName,
		Command: []string{"sh", "-c", `echo 'anka  ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers`},
		User:    "root",
	})
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *StepEnableSudo) Cleanup(state multistep.StateBag) {}
