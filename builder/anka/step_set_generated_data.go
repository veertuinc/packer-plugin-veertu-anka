package anka

import (
	"bytes"
	"context"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
)

type StepSetGeneratedData struct {
	client        client.Client
	vmName        string
	GeneratedData *packerbuilderdata.GeneratedData
}

func (s *StepSetGeneratedData) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	log.Printf("Exposing build contextual variables...")

	s.client = state.Get("client").(client.Client)
	s.vmName = state.Get("vm_name").(string)
	darwinVersion := client.RunParams{
		Command: []string{"/usr/bin/uname", "-r"},
		VMName:  s.vmName,
		Stdout:  &bytes.Buffer{},
	}
	osVersion := client.RunParams{
		Command: []string{"/usr/bin/sw_vers", "-productVersion"},
		VMName:  s.vmName,
		Stdout:  &bytes.Buffer{},
	}

	_, err := s.client.Run(darwinVersion)
	if err != nil {
		return multistep.ActionHalt
	}

	_, osErr := s.client.Run(osVersion)
	if osErr != nil {
		return multistep.ActionHalt
	}

	s.GeneratedData.Put("VMName", s.vmName)
	s.GeneratedData.Put("OSVersion", osVersion.Stdout)
	s.GeneratedData.Put("DarwinVersion", darwinVersion.Stdout)

	return multistep.ActionContinue
}

// Cleanup will run whenever there are any errors.
// No cleanup needs to happen here
func (s *StepSetGeneratedData) Cleanup(_ multistep.StateBag) {
}
