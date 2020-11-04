package anka

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/groob/plist"
	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
)

var random *rand.Rand

func init() {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type StepCreateVM struct {
	client *client.Client
	vmName string
}

func (s *StepCreateVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	s.client = state.Get("client").(*client.Client)

	onError := func(err error) multistep.StepAction {
		return stepError(ui, state, err)
	}

	// By default, do not create a new sourceVM
	doCreateSourceVM := false
	installerAppFullName := "anka-packer-base"

	if config.InstallerApp != "" { // If users specifies an InstallerApp and sourceVM doesn't exist, assume they want to build a new VM template and use the macOS installer version
		doCreateSourceVM = true
		ui.Say(fmt.Sprintf("Extracting version from installer app: %q", config.InstallerApp))
		macOSVersionFromInstallerApp, err := obtainMacOSVersionFromInstallerApp(config.InstallerApp) // Grab the version details from the Info.plist inside of the Installer package
		if err != nil {
			return onError(err)
		}
		installerAppFullName = fmt.Sprintf("%s-%s", installerAppFullName, macOSVersionFromInstallerApp) // We need to set the SourceVMName since the user didn't and the logic below creates a VM using it
	}

	if config.SourceVMName == "" {
		config.SourceVMName = installerAppFullName
	}

	if strings.ContainsAny(config.SourceVMName, " \n") {
		return onError(fmt.Errorf("VM name contains spaces %q", config.SourceVMName))
	}

	if sourceVMExists, _ := s.client.Exists(config.SourceVMName, ui); sourceVMExists { // Reuse the base VM template if it matches the one from the installer
		doCreateSourceVM = false
	}

	s.vmName = config.SourceVMName // Used for cleanup BEFORE THE CLONE

	if config.VMName == "" { // If user doesn't give a vm_name, generate one
		config.VMName = fmt.Sprintf("anka-packer-%s", randSeq(10))
	}

	// If we need to create the base/source VM
	if doCreateSourceVM {
		cpuCount, err := strconv.ParseInt(config.CPUCount, 10, 32)
		if err != nil {
			return onError(err)
		}

		ui.Say(fmt.Sprintf("Creating a new vm (%s) from installer, this will take a while", config.SourceVMName))

		outputStream := make(chan string)
		go func() {
			for msg := range outputStream {
				ui.Say(msg)
			}
		}()

		resp, err := s.client.Create(client.CreateParams{
			DiskSize:     config.DiskSize,
			InstallerApp: config.InstallerApp,
			RAMSize:      config.RAMSize,
			CPUCount:     int(cpuCount),
			Name:         config.SourceVMName,
		}, outputStream)
		if err != nil {
			return onError(err)
		}

		close(outputStream)

		ui.Say(fmt.Sprintf("VM %s was created (%s)", config.SourceVMName, resp.UUID))
	} // doCreateSourceVM

	show, err := s.client.Show(config.SourceVMName)
	if err != nil {
		return onError(err)
	}

	if show.IsRunning() {
		ui.Say(fmt.Sprintf("Suspending VM %s", config.SourceVMName))
		err := s.client.Suspend(client.SuspendParams{
			VMName: config.SourceVMName,
		})
		if err != nil {
			return onError(err)
		}
	}

	s.vmName = config.VMName // Used for cleanup

	// If the user forces the build (packer build --force), delete the existing VM that would fail the build
	exists, _ := s.client.Exists(config.VMName, ui)
	if exists && config.PackerConfig.PackerForce {
		ui.Say(fmt.Sprintf("Deleting existing virtual machine %s", config.VMName))
		err = s.client.Delete(client.DeleteParams{
			VMName: config.VMName,
		})
		if err != nil {
			return onError(err)
		}
	}

	ui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine %s", config.SourceVMName, config.VMName))
	err = s.client.Clone(client.CloneParams{
		VMName:     config.VMName,
		SourceUUID: show.UUID,
	})

	if err != nil {
		return onError(err)
	}

	// If cloned from an existing VM, check if modification is required
	if !doCreateSourceVM {

		showResponse, err := s.client.Show(config.VMName)
		if err != nil {
			return onError(err)
		}

		clonedVMDescribeResponse, err := s.client.Describe(config.VMName)
		if err != nil {
			return onError(err)
		}

		stopParams := client.StopParams{
			VMName: showResponse.Name,
			Force:  true,
		}

		s.vmName = showResponse.Name // needed for cleanup; prevent "No VM name - skipping this part"

		// Disk Size
		err, diskSizeBytes := convertDiskSizeToBytes(config.DiskSize)
		if err != nil {
			return onError(err)
		}
		if diskSizeBytes != showResponse.HardDrive {
			ui.Say(fmt.Sprintf("Modifying VM %s disk size to %s", showResponse.Name, config.DiskSize))

			if diskSizeBytes < showResponse.HardDrive {
				return onError(fmt.Errorf("Can not set disk size to smaller than source VM"))
			}

			if err := s.client.Stop(stopParams); err != nil {
				return onError(err)
			}

			err = s.client.Modify(showResponse.Name, "set", "hard-drive", "-s", config.DiskSize)
			if err != nil {
				return onError(err)
			}
		}

		// Port Forwarding
		if len(config.PortForwardingRules) > 0 {
			if err := s.client.Stop(stopParams); err != nil {
				return onError(err)
			}
			// Check if the rule already exists
			for _, wantedPortForwardingRule := range config.PortForwardingRules {
				ui.Say(fmt.Sprintf("Ensuring %s port-forwarding (Guest Port: %s, Host Port: %s, Rule Name: %s)", showResponse.Name, wantedPortForwardingRule.PortForwardingGuestPort, wantedPortForwardingRule.PortForwardingHostPort, wantedPortForwardingRule.PortForwardingRuleName))
				for _, existingNetworkCard := range clonedVMDescribeResponse.NetworkCards {
					for _, existingPortForwardingRule := range existingNetworkCard.PortForwardingRules {
						// Check if host port is set already and warn the user
						if wantedPortForwardingRule.PortForwardingHostPort == fmt.Sprint(existingPortForwardingRule.HostPort) {
							ui.Error(fmt.Sprintf("Found an already existing rule using %s! This can cause VMs to not start!", wantedPortForwardingRule.PortForwardingHostPort))
						}
					}
				}
				err = s.client.Modify(showResponse.Name, "add", "port-forwarding", "--host-port", wantedPortForwardingRule.PortForwardingHostPort, "--guest-port", wantedPortForwardingRule.PortForwardingGuestPort, wantedPortForwardingRule.PortForwardingRuleName)
				if config.PackerConfig.PackerForce == false { // If force is enabled, just skip
					if err != nil {
						return onError(err)
					}
				}
			}
		}

		// Custom Variables
		if config.HWUUID != "" {
			ui.Say(fmt.Sprintf("Modifying VM custom-variable hw.UUID to %s", config.HWUUID))

			if err := s.client.Stop(stopParams); err != nil {
				return onError(err)
			}

			err = s.client.Modify(showResponse.Name, "set", "custom-variable", "hw.UUID", config.HWUUID)
			if err != nil {
				return onError(err)
			}
		}

		// RAM
		if config.RAMSize != showResponse.RAM {
			ui.Say(fmt.Sprintf("Modifying VM %s RAM to %s", showResponse.Name, config.RAMSize))
			if err := s.client.Stop(stopParams); err != nil {
				return onError(err)
			}

			err = s.client.Modify(showResponse.Name, "set", "ram", config.RAMSize)
			if err != nil {
				return onError(err)
			}
		}

		// CPU Core Count
		if config.CPUCount != strconv.Itoa(showResponse.CPUCores) {
			ui.Say(fmt.Sprintf("Modifying VM %s CPU core count to %s", showResponse.Name, config.CPUCount))

			if err := s.client.Stop(stopParams); err != nil {
				return onError(err)
			}

			err = s.client.Modify(showResponse.Name, "set", "cpu", "-c", config.CPUCount)
			if err != nil {
				return onError(err)
			}
		}

	}

	state.Put("vm_name", config.VMName)

	return multistep.ActionContinue
}

func (s *StepCreateVM) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packer.Ui)

	log.Println("Cleaning up create VM step")
	if s.vmName == "" {
		return
	}
	_, halted := state.GetOk(multistep.StateHalted)
	_, canceled := state.GetOk(multistep.StateCancelled)

	errorMessage := state.Get("error")
	switch errorMessage.(type) {
	case nil:
		errorMessage = ""
	case client.MachineReadableError:
		errorMessage = errorMessage.(client.MachineReadableError)
	default:
		errorMessage = ""
	}

	if fmt.Sprintf("%s", errorMessage) != fmt.Sprintf("%s: already exists", s.vmName) { // Skip delete if the VM already exists...
		if halted || canceled {
			ui.Say(fmt.Sprintf("Deleting VM %s", s.vmName))
			err := s.client.Delete(client.DeleteParams{VMName: s.vmName})
			if err != nil {
				ui.Error(fmt.Sprint(err))
			}
			return
		}
	}

	err := s.client.Suspend(client.SuspendParams{
		VMName: s.vmName,
	})
	if err != nil {
		ui.Error(fmt.Sprint(err))
		s.client.Delete(client.DeleteParams{VMName: s.vmName})
		panic(err)
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

func obtainMacOSVersionFromInstallerApp(path string) (string, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("installer app does not exist at %q: %w", path, err)
	}
	if err != nil {
		return "", fmt.Errorf("failed to stat installer at %q: %w", path, err)
	}

	plistPath := filepath.Join(path, "Contents", "Info.plist")
	_, err = os.Stat(plistPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("installer app info plist did not exist at %q: %w", plistPath, err)
	}
	if err != nil {
		return "", fmt.Errorf("failed to stat installer app info plist at %q: %w", plistPath, err)
	}
	plistContent, err := os.Open(plistPath)

	var installAppPlist struct {
		PlatformVersion string `plist:"DTPlatformVersion"`
		ShortVersion    string `plist:"CFBundleShortVersionString"`
	}
	plist.NewXMLDecoder(plistContent).Decode(&installAppPlist)

	return fmt.Sprintf("%s-%s", installAppPlist.PlatformVersion, installAppPlist.ShortVersion), nil
}
