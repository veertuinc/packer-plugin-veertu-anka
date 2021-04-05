package anka

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
	"github.com/veertuinc/packer-builder-veertu-anka/common"
	"github.com/veertuinc/packer-builder-veertu-anka/util"
)

const (
	defaultDiskSize  = "40G"
	defaultRAMSize   = "4G"
	defaultVCPUCount = "2"
)

// StepCreateVM will be used to run the create step for an 'vm-create' builder types
type StepCreateVM struct {
	client client.Client
	vmName string
}

// Run creates a new vm from a local installer app
func (s *StepCreateVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	ankaUtil := state.Get("util").(util.Util)
	onError := func(err error) multistep.StepAction {
		return ankaUtil.StepError(ui, state, err)
	}
	installerAppData := util.InstallAppPlist{}

	s.client = state.Get("client").(client.Client)
	s.vmName = config.VMName

	if s.vmName == "" {
		installerAppData, err = ankaUtil.ObtainMacOSVersionFromInstallerApp(config.InstallerApp)
		if err != nil {
			return onError(err)
		}

		s.vmName = fmt.Sprintf("anka-packer-base-%s-%s", installerAppData.OSVersion, installerAppData.BundlerVersion)
	}

	state.Put("vm_name", s.vmName)

	if config.PackerForce {
		exists, err := s.client.Exists(s.vmName)
		if err != nil {
			return onError(err)
		}
		if exists {
			ui.Say(fmt.Sprintf("Deleting existing virtual machine %s", s.vmName))

			err = s.client.Delete(client.DeleteParams{VMName: s.vmName})
			if err != nil {
				return onError(err)
			}
		}
	}

	err = s.createFromInstallerApp(ui, config)
	if err != nil {
		return onError(err)
	}

	return multistep.ActionContinue
}

// Cleanup will delete the vm if there happens to be an error and handle anything failed states
func (s *StepCreateVM) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packer.Ui)

	log.Println("Cleaning up create VM step")
	if s.vmName == "" {
		return
	}

	_, halted := state.GetOk(multistep.StateHalted)
	_, canceled := state.GetOk(multistep.StateCancelled)
	errorObj := state.Get("error")
	switch errorObj.(type) {
	case *common.VMAlreadyExistsError:
		return
	case *common.VMNotFoundException:
		return
	default:
		if halted || canceled {
			ui.Say(fmt.Sprintf("Deleting VM %s", s.vmName))

			err := s.client.Delete(client.DeleteParams{VMName: s.vmName})
			if err != nil {
				ui.Error(fmt.Sprint(err))
			}
			return
		}
	}
}

func (s *StepCreateVM) createFromInstallerApp(ui packer.Ui, config *Config) error {
	ui.Say(fmt.Sprintf("Creating a new VM Template (%s) from installer, this will take a while", s.vmName))

	outputStream := make(chan string)

	go func() {
		for msg := range outputStream {
			ui.Say(msg)
		}
	}()

	createParams := client.CreateParams{
		InstallerApp: config.InstallerApp,
		Name:         s.vmName,
		DiskSize:     config.DiskSize,
		VCPUCount:    config.VCPUCount,
		RAMSize:      config.RAMSize,
	}

	if createParams.DiskSize == "" {
		createParams.DiskSize = defaultDiskSize
	}

	if createParams.VCPUCount == "" {
		createParams.VCPUCount = defaultVCPUCount
	}

	if createParams.RAMSize == "" {
		createParams.RAMSize = defaultRAMSize
	}

	resp, err := s.client.Create(createParams, outputStream)
	if err != nil {
		return err
	}

	ui.Say(fmt.Sprintf("VM %s was created (%s)", s.vmName, resp.UUID))

	close(outputStream)

	return nil
}
