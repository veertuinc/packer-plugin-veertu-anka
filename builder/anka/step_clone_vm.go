package anka

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
	"github.com/veertuinc/packer-builder-veertu-anka/common"
	"github.com/veertuinc/packer-builder-veertu-anka/util"
)

// StepCloneVM will be used to run the clone step for any 'vm-clone' builder types
type StepCloneVM struct {
	client client.Client
	vmName string
}

// Run clones a vm from a source vm either from an anka registry or locally
func (s *StepCloneVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	ankaUtil := state.Get("util").(util.Util)
	onError := func(err error) multistep.StepAction {
		return ankaUtil.StepError(ui, state, err)
	}
	sourceVMTag := "latest"
	doPull := config.AlwaysFetch

	if config.SourceVMTag != "" {
		sourceVMTag = config.SourceVMTag
	}

	s.client = state.Get("client").(client.Client)
	s.vmName = config.VMName

	if s.vmName == "" {
		s.vmName = fmt.Sprintf("%s-%s", config.SourceVMName, ankaUtil.RandSeq(10))
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

	if !config.AlwaysFetch {
		log.Printf("Searching for %s locally...", config.SourceVMName)

		sourceExists, err := s.client.Exists(config.SourceVMName)
		if err != nil {
			return onError(err)
		}
		if !sourceExists {
			log.Printf("Could not find %s locally, looking in anka registry...", config.SourceVMName)

			doPull = true
		}
	}

	if doPull {
		ui.Say(fmt.Sprintf("Pulling source VM %s with tag %s from Anka Registry", config.SourceVMName, sourceVMTag))

		registryParams := client.RegistryParams{
			RegistryName: config.RegistryName,
			RegistryURL:  config.RegistryURL,
			NodeCertPath: config.NodeCertPath,
			NodeKeyPath:  config.NodeKeyPath,
			CaRootPath:   config.CaRootPath,
			IsInsecure:   config.IsInsecure,
		}

		registryPullParams := client.RegistryPullParams{
			VMID:   config.SourceVMName,
			Tag:    sourceVMTag,
			Local:  false,
			Shrink: false,
		}

		err := s.client.RegistryPull(registryParams, registryPullParams)
		if err != nil {
			return onError(fmt.Errorf("failed to pull vm %v with tag %v from registry", config.SourceVMName, sourceVMTag))
		}
	}

	sourceShow, err := s.client.Show(config.SourceVMName)
	if err != nil {
		return onError(err)
	}

	ui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", sourceShow.Name, s.vmName))

	err = s.client.Clone(client.CloneParams{VMName: s.vmName, SourceUUID: sourceShow.UUID})
	if err != nil {
		return onError(err)
	}

	clonedShow, err := s.client.Show(s.vmName)
	if err != nil {
		return onError(err)
	}

	err = s.modifyVMResources(clonedShow, config, ui, ankaUtil)
	if err != nil {
		return onError(err)
	}

	err = s.modifyVMProperties(clonedShow, config, ui)
	if err != nil {
		return onError(err)
	}

	if config.UpdateAddons {
		ui.Say(fmt.Sprintf("Updating guest addons for %s", s.vmName))

		err := s.client.UpdateAddons(s.vmName)
		if err != nil {
			return onError(err)
		}
	}

	return multistep.ActionContinue
}

// Cleanup will delete the vm if there happens to be an error and handle anything failed states
func (s *StepCloneVM) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packer.Ui)

	log.Println("Cleaning up clone VM step")
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

func (s *StepCloneVM) modifyVMResources(showResponse client.ShowResponse, config *Config, ui packer.Ui, util util.Util) error {
	stopParams := client.StopParams{
		VMName: showResponse.Name,
		Force:  true,
	}

	if config.DiskSize != "" {
		diskSizeBytes, err := util.ConvertDiskSizeToBytes(config.DiskSize)
		if err != nil {
			return err
		}

		if diskSizeBytes > showResponse.HardDrive {
			err := s.client.Stop(stopParams)
			if err != nil {
				return err
			}

			ui.Say(fmt.Sprintf("Modifying VM %s disk size to %s", showResponse.Name, config.DiskSize))

			err = s.client.Modify(showResponse.Name, "set", "hard-drive", "-s", config.DiskSize)
			if err != nil {
				return err
			}

			// Resize the inner VM disk too with diskutil
			_, err = s.client.Run(client.RunParams{
				VMName:  showResponse.Name,
				Command: []string{"diskutil", "apfs", "resizeContainer", "disk1", "0"},
			})
			if err != nil {
				return err
			}

			// Prevent 'VM is already running' error
			err = s.client.Stop(stopParams)
			if err != nil {
				return err
			}
		}

		if diskSizeBytes < showResponse.HardDrive {
			return fmt.Errorf("Shrinking VM disks is not allowed! Source VM Disk Size (bytes): %v", showResponse.HardDrive)
		}
	}

	if config.RAMSize != "" && config.RAMSize != showResponse.RAM {
		err := s.client.Stop(stopParams)
		if err != nil {
			return err
		}

		ui.Say(fmt.Sprintf("Modifying VM %s RAM to %s", showResponse.Name, config.RAMSize))

		err = s.client.Modify(showResponse.Name, "set", "ram", config.RAMSize)
		if err != nil {
			return err
		}
	}

	if config.VCPUCount != "" {
		stringVCPUCount, err := strconv.ParseInt(config.VCPUCount, 10, 32)
		if err != nil {
			return err
		}

		if int(stringVCPUCount) != showResponse.VCPUCores {
			err := s.client.Stop(stopParams)
			if err != nil {
				return err
			}

			ui.Say(fmt.Sprintf("Modifying VM %s VCPU core count to %v", showResponse.Name, stringVCPUCount))

			err = s.client.Modify(showResponse.Name, "set", "cpu", "-c", strconv.Itoa(int(stringVCPUCount)))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *StepCloneVM) modifyVMProperties(showResponse client.ShowResponse, config *Config, ui packer.Ui) error {
	stopParams := client.StopParams{
		VMName: showResponse.Name,
		Force:  true,
	}

	if len(config.PortForwardingRules) > 0 {
		describeResponse, err := s.client.Describe(showResponse.Name)
		if err != nil {
			return err
		}

		existingForwardedPorts := make(map[int]struct{})
		for _, existingNetworkCard := range describeResponse.NetworkCards {
			for _, existingPortForwardingRule := range existingNetworkCard.PortForwardingRules {
				existingForwardedPorts[existingPortForwardingRule.HostPort] = struct{}{}
			}
		}

		for _, wantedPortForwardingRule := range config.PortForwardingRules {
			ui.Say(fmt.Sprintf("Ensuring %s port-forwarding (Guest Port: %s, Host Port: %s, Rule Name: %s)", showResponse.Name, strconv.Itoa(wantedPortForwardingRule.PortForwardingGuestPort), strconv.Itoa(wantedPortForwardingRule.PortForwardingHostPort), wantedPortForwardingRule.PortForwardingRuleName))
			if _, ok := existingForwardedPorts[wantedPortForwardingRule.PortForwardingHostPort]; ok {
				ui.Error(fmt.Sprintf("Found an existing host port rule (%s)! Skipping without setting...", strconv.Itoa(wantedPortForwardingRule.PortForwardingHostPort)))
				continue
			}

			err := s.client.Stop(stopParams)
			if err != nil {
				return err
			}

			err = s.client.Modify(showResponse.Name, "add", "port-forwarding", "--host-port", strconv.Itoa(wantedPortForwardingRule.PortForwardingHostPort), "--guest-port", strconv.Itoa(wantedPortForwardingRule.PortForwardingGuestPort), wantedPortForwardingRule.PortForwardingRuleName)
			if !config.PackerConfig.PackerForce {
				if err != nil {
					return err
				}
			}
		}
	}

	if config.HWUUID != "" {
		err := s.client.Stop(stopParams)
		if err != nil {
			return err
		}

		ui.Say(fmt.Sprintf("Modifying VM custom-variable hw.UUID to %s", config.HWUUID))

		err = s.client.Modify(showResponse.Name, "set", "custom-variable", "hw.UUID", config.HWUUID)
		if err != nil {
			return err
		}
	}

	return nil
}
