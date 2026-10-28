package anka

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-plugin-veertu-anka/client"
	"github.com/veertuinc/packer-plugin-veertu-anka/common"
	"github.com/veertuinc/packer-plugin-veertu-anka/util"
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

	doPull := config.AlwaysFetch

	sourceVMTag := fmt.Sprintf("%s tag", config.SourceVMTag)
	if config.SourceVMTag == "" {
		sourceVMTag = "latest tag"
	}

	s.client = state.Get("client").(client.Client)
	s.vmName = resolveWorkingVMName(config, ankaUtil)

	state.Put("vm_name", s.vmName)

	if config.PackerForce && !config.SkipClone {
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
		if err := s.pullSourceVM(config, sourceVMTag, ui); err != nil {
			return onError(err)
		}
	}

	sourceShow, err := s.client.Show(config.SourceVMName)
	if err != nil {
		return onError(err)
	}

	sourceVMTagLog := config.SourceVMTag
	if sourceVMTagLog == "" {
		sourceVMTagLog = "latest"
	}
	ui.Say(fmt.Sprintf("Source TEMPLATE_NAME: %s, TEMPLATE_ID: %s, TAG_NAME: %s", sourceShow.Name, sourceShow.UUID, sourceVMTagLog))

	// Check arch of host
	// // If arm64, ensure the source has a local tag
	if !doPull && !config.SkipClone {
		if sourceShow.Version == "" {
			if config.HostArch == "arm64" {
				ui.Say("Preparing source VM by creating a local tag (necessary in Anka 3 to optimize disk usage of clones)")
				pushParams := client.RegistryPushParams{
					Tag:      fmt.Sprintf("local-tag-%s", ankaUtil.RandSeq(10)),
					RemoteVM: "",
					Local:    true,
					Force:    false,
					VMID:     config.SourceVMName,
				}
				s.client.RegistryPush(client.RegistryParams{HostArch: config.HostArch}, pushParams, nil)
			}
		}
	}

	if err := applyThinOCIPulls(s.client, config.SourceVMName, config.ThinOCIPulls, ui); err != nil {
		return onError(err)
	}

	workingVMShowLabel := "post-clone"
	if config.SkipClone {
		workingVMShowLabel = "skip-clone"
		ui.Say(fmt.Sprintf("Skipping clone; building on source VM %s in place", s.vmName))
	} else {
		ui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", sourceShow.Name, s.vmName))

		err = s.client.Clone(client.CloneParams{VMName: s.vmName, SourceUUID: sourceShow.UUID})
		if err != nil {
			return onError(err)
		}
	}

	workingVMShow, err := s.client.Show(s.vmName)
	if err != nil {
		return onError(err)
	}

	client.LogShowResponse(ui, workingVMShowLabel, workingVMShow)

	err = s.modifyVMResources(workingVMShow, config, ui, ankaUtil)
	if err != nil {
		return onError(err)
	}

	err = s.modifyVMProperties(workingVMShow, config, ui)
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

	if config := state.Get("config").(*Config); config.SkipClone {
		return
	}

	errorObj := state.Get("error")
	switch errorObj.(type) {
	case *common.VMAlreadyExistsError:
		return
	case *common.VMNotFoundException:
		return
	default:
		if !shouldDeleteAnkaTemplateAfterFailedBuild(state) {
			return
		}
		ui.Say(fmt.Sprintf("Deleting VM %s", s.vmName))

		err := s.client.Delete(client.DeleteParams{VMName: s.vmName})
		if err != nil {
			ui.Error(fmt.Sprint(err))
		}
	}
}

func resolveWorkingVMName(config *Config, ankaUtil util.Util) string {
	if config.SkipClone {
		return config.SourceVMName
	}
	if config.VMName != "" {
		return config.VMName
	}
	return fmt.Sprintf("%s-%s", config.SourceVMName, ankaUtil.RandSeq(10))
}

// pullSourceVM pulls the full source tag: `anka pull` for an OCI remote, `anka registry pull` for an Anka Registry.
func (s *StepCloneVM) pullSourceVM(config *Config, sourceVMTagDescription string, ui packer.Ui) error {
	if client.IsOCIRemote(config.Remote) {
		ui.Say(fmt.Sprintf("Pulling source VM %s with %s from OCI registry %s", config.SourceVMName, sourceVMTagDescription, config.Remote))
		ociPullParams := client.OCIPullParams{
			VMID:   config.SourceVMName,
			Remote: config.Remote,
			Tag:    config.SourceVMTag,
		}
		err := client.RunWithOutputStreamToUI(ui, func(outputStream chan string) error {
			return s.client.OCIPull(ociPullParams, outputStream)
		})
		if err != nil {
			return fmt.Errorf("failed to pull vm %s with %s from OCI registry %s: %w", config.SourceVMName, sourceVMTagDescription, config.Remote, err)
		}
		return nil
	}

	ui.Say(fmt.Sprintf("Pulling source VM %s with %s from Anka Registry", config.SourceVMName, sourceVMTagDescription))
	registryPullParams := client.RegistryPullParams{
		VMID:   config.SourceVMName,
		Tag:    config.SourceVMTag,
		Local:  false,
		Shrink: false,
	}
	err := client.RunWithOutputStreamToUI(ui, func(outputStream chan string) error {
		return s.client.RegistryPull(config.registryParams(), registryPullParams, outputStream)
	})
	if err != nil {
		return fmt.Errorf("failed to pull vm %s with %s from registry (make sure to add it as the default: https://docs.veertu.com/anka/intel/command-line-reference/#registry-add)", config.SourceVMName, sourceVMTagDescription)
	}
	return nil
}

func (s *StepCloneVM) modifyVMResources(showResponse client.ShowResponse, config *Config, ui packer.Ui, util util.Util) error {
	stopParams := client.StopParams{
		VMName: showResponse.Name,
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
				Command: []string{guestAPFSResizeContainerShellCommand},
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
				if wantedPortForwardingRule.PortForwardingHostPort > 0 {
					ui.Error(fmt.Sprintf("Found an existing host port rule (%s)! Skipping without setting...", strconv.Itoa(wantedPortForwardingRule.PortForwardingHostPort)))
					continue
				}
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
		ui.Say(fmt.Sprintf("Modifying VM custom-variable hw.uuid to %s", config.HWUUID))
		err = s.client.Modify(showResponse.Name, "set", "custom-variable", "hw.uuid", config.HWUUID)
		if err != nil {
			return err
		}
	}

	if config.DisplayController != "" {
		err := s.client.Stop(stopParams)
		if err != nil {
			return err
		}
		ui.Say(fmt.Sprintf("Modifying VM display controller to %s", config.DisplayController))
		err = s.client.Modify(showResponse.Name, "set", "display", "-c", config.DisplayController)
		if err != nil {
			return err
		}
	}

	if len(config.HostDirectoryMounts) > 0 {
		err := applyHostDirectoryMounts(
			s.client,
			stopParams,
			showResponse.Name,
			config.HostDirectoryMounts,
			config.PackerConfig.PackerForce,
			ui,
		)
		if err != nil {
			return err
		}
	}

	if len(config.VMLabels) > 0 {
		err := applyVMLabels(
			s.client,
			stopParams,
			showResponse.Name,
			config.VMLabels,
			ui,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
