package anka

import (
	"bytes"
	"context"
	"log"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
)

var (
	darwinBuffer, osBuffer bytes.Buffer
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
		Stdout:  &darwinBuffer,
	}
	osVersion := client.RunParams{
		Command: []string{"/usr/bin/sw_vers", "-productVersion"},
		VMName:  s.vmName,
		Stdout:  &osBuffer,
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
	s.GeneratedData.Put("OSVersion", strings.TrimSpace(osBuffer.String()))
	s.GeneratedData.Put("DarwinVersion", strings.TrimSpace(darwinBuffer.String()))

	return multistep.ActionContinue
}

// Cleanup will run whenever there are any errors.
// No cleanup needs to happen here
func (s *StepSetGeneratedData) Cleanup(_ multistep.StateBag) {
}
