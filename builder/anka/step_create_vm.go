package anka

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-plugin-veertu-anka/client"
	"github.com/veertuinc/packer-plugin-veertu-anka/common"
	"github.com/veertuinc/packer-plugin-veertu-anka/util"
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

	s.client = state.Get("client").(client.Client)
	s.vmName = config.VMName

	if s.vmName == "" {
		matchInstaller, err := regexp.Match(".app(/?)$|.ipsw(/?)$", []byte(config.Installer))
		if err != nil {
			return onError(err)
		}
		if matchInstaller {
			if config.HostArch == "arm64" {
				installerData := util.InstallerIPSWPlist{}
				installerData, err = ankaUtil.ObtainMacOSVersionFromInstallerIPSW(config.Installer)
				if err != nil {
					return onError(err)
				}
				s.vmName = fmt.Sprintf("anka-packer-base-%s-%s", installerData.ProductVersion, installerData.ProductBuildVersion)
			} else {
				installerData := util.InstallerAppPlist{}
				installerData, err = ankaUtil.ObtainMacOSVersionFromInstallerApp(config.Installer)
				if err != nil {
					return onError(err)
				}
				s.vmName = fmt.Sprintf("anka-packer-base-%s-%s", installerData.OSVersion, installerData.BundlerVersion)
			}
		} else {
			s.vmName = fmt.Sprintf("anka-packer-base-%s", config.Installer)
		}
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

	err = s.createFromInstaller(ui, config)
	if err != nil {
		return onError(err)
	}

	createdShow, err := s.client.Show(s.vmName)
	if err != nil {
		return onError(err)
	}

	err = s.modifyVMProperties(createdShow, config, ui)
	if err != nil {
		return onError(err)
	}

	return multistep.ActionContinue
}

func (s *StepCreateVM) createFromInstaller(ui packer.Ui, config *Config) error {
	ui.Say(fmt.Sprintf("Creating a new VM Template (%s) from installer, this will take a while", s.vmName))

	outputStream := make(chan string)

	go func() {
		for msg := range outputStream {
			ui.Say(msg)
		}
	}()

	createParams := client.CreateParams{
		Installer: config.Installer,
		Name:      s.vmName,
		DiskSize:  config.DiskSize,
		VCPUCount: config.VCPUCount,
		RAMSize:   config.RAMSize,
	}

	createdVMUUID, err := s.client.Create(createParams, outputStream)
	if err != nil {
		return err
	}

	ui.Say(fmt.Sprintf("VM %s was created (%s)", s.vmName, createdVMUUID))

	close(outputStream)

	return nil
}

func (s *StepCreateVM) modifyVMProperties(showResponse client.ShowResponse, config *Config, ui packer.Ui) error {
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

	if config.DisplayResolution != "" {
		err := s.client.Stop(stopParams)
		if err != nil {
			return err
		}
		ui.Say(fmt.Sprintf("Modifying VM display resolution to %s", config.DisplayResolution))
		err = s.client.Modify(showResponse.Name, "set", "display", "-r", config.DisplayResolution)
		if err != nil {
			return err
		}
	}

	return nil
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
