package anka

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/hashicorp/packer/packer"
	"github.com/lox/packer-builder-veertu-anka/client"
	"github.com/mitchellh/multistep"
)

var random *rand.Rand

func init() {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type StepCreateVM struct {
	client *client.Client
	vmName string
}

func (s *StepCreateVM) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	s.client = state.Get("client").(*client.Client)
	sourceVM := config.SourceVMName

	onError := func(err error) multistep.StepAction {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if sourceVM == "" {
		ui.Say("Creating a new disk from installer, this will take a while")
		imageID, err := s.client.CreateDisk(client.CreateDiskParams{
			DiskSize:     config.DiskSize,
			InstallerApp: config.InstallerApp,
		})
		if err != nil {
			return onError(err)
		}

		cpuCount, err := strconv.ParseInt(config.CPUCount, 10, 32)
		if err != nil {
			return onError(err)
		}

		sourceVM = fmt.Sprintf("anka-disk-base-%s", randSeq(10))

		ui.Say("Creating a new virtual machine for disk")
		_, err = s.client.Create(client.CreateParams{
			ImageID:  imageID,
			RamSize:  config.RamSize,
			CPUCount: int(cpuCount),
			Name:     sourceVM,
		})
		if err != nil {
			return onError(fmt.Errorf("Error creating VM: %v", err))
		}

		ui.Say(fmt.Sprintf("VM %s was created", sourceVM))
	}

	show, err := s.client.Show(sourceVM)
	if err != nil {
		return onError(err)
	}

	if show.IsRunning() {
		ui.Say(fmt.Sprintf("Suspending VM %s", sourceVM))
		err := s.client.Suspend(client.SuspendParams{
			VMName: sourceVM,
		})
		if err != nil {
			return onError(err)
		}
	}

	vmName := config.VMName
	if vmName == "" {
		vmName = fmt.Sprintf("anka-packer-%s", randSeq(10))
	}

	exists, _ := s.client.Exists(vmName)
	if exists && config.PackerConfig.PackerForce {
		ui.Say(fmt.Sprintf("Deleting existing virtual machine %s", vmName))
		err = s.client.Delete(client.DeleteParams{
			VMName: vmName,
		})
		if err != nil {
			return onError(err)
		}
	}

	ui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine %s", sourceVM, vmName))
	err = s.client.Clone(client.CloneParams{
		VMName:     vmName,
		SourceUUID: show.UUID,
	})
	if err != nil {
		return onError(err)
	}

	state.Put("vm_name", vmName)
	s.vmName = vmName

	return multistep.ActionContinue
}

func (s *StepCreateVM) Cleanup(state multistep.StateBag) {
	if _, ok := state.GetOk(multistep.StateCancelled); ok {
		err := s.client.Delete(client.DeleteParams{
			VMName: s.vmName,
			Force:  true,
		})
		if err != nil {
			log.Println(err)
		}
		return
	}

	err := s.client.Suspend(client.SuspendParams{
		VMName: s.vmName,
	})
	if err != nil {
		log.Println(err)
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[random.Intn(len(letters))]
	}
	return string(b)
}
