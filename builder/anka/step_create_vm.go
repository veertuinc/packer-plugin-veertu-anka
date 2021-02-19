package anka

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/groob/plist"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
	"github.com/veertuinc/packer-builder-veertu-anka/common"
)

var random *rand.Rand

func init() {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type StepCreateVM struct {
	client client.Client
	vmName string
}

const (
	DEFAULT_DISK_SIZE = "40G"
	DEFAULT_RAM_SIZE  = "4G"
	DEFAULT_CPU_COUNT = "2"
)

func (s *StepCreateVM) modifyVMResources(showResponse client.ShowResponse, config *Config, ui packer.Ui) error {
	stopParams := client.StopParams{
		VMName: showResponse.Name,
		Force:  true,
	}

	if config.DiskSize != "" {
		err, diskSizeBytes := convertDiskSizeToBytes(config.DiskSize)
		if err != nil {
			return err
		}
		if diskSizeBytes > showResponse.HardDrive {
			if err := s.client.Stop(stopParams); err != nil {
				return err
			}
			ui.Say(fmt.Sprintf("Modifying VM %s disk size to %s", showResponse.Name, config.DiskSize))
			err = s.client.Modify(showResponse.Name, "set", "hard-drive", "-s", config.DiskSize)
			if err != nil {
				return err
			}
			// Resize the inner VM disk too with diskutil
			err, _ = s.client.Run(client.RunParams{
				VMName:  showResponse.Name,
				Command: []string{"diskutil", "apfs", "resizeContainer", "disk1", "0"},
			})
			if err != nil {
				return err
			}
			if err := s.client.Stop(stopParams); err != nil { // Prevent 'VM is already running' error
				return err
			}
		}
		if diskSizeBytes < showResponse.HardDrive {
			return fmt.Errorf("Shrinking VM disks is not allowed! Source VM Disk Size (bytes): %v", showResponse.HardDrive)
		}
	}

	if config.RAMSize != "" && config.RAMSize != showResponse.RAM {
		if err := s.client.Stop(stopParams); err != nil {
			return err
		}
		ui.Say(fmt.Sprintf("Modifying VM %s RAM to %s", showResponse.Name, config.RAMSize))
		err := s.client.Modify(showResponse.Name, "set", "ram", config.RAMSize)
		if err != nil {
			return err
		}
	}

	if config.CPUCount != "" {
		stringCPUCount, err := strconv.ParseInt(config.CPUCount, 10, 32)
		if err != nil {
			return err
		}
		if int(stringCPUCount) != showResponse.CPUCores {
			if err := s.client.Stop(stopParams); err != nil {
				return err
			}
			ui.Say(fmt.Sprintf("Modifying VM %s CPU core count to %v", showResponse.Name, stringCPUCount))
			err = s.client.Modify(showResponse.Name, "set", "cpu", "-c", strconv.Itoa(int(stringCPUCount)))
			if err != nil {
				return err
			}
		}
	}

	return nil

}

func (s *StepCreateVM) modifyVMProperties(describeResponse client.DescribeResponse, showResponse client.ShowResponse, config *Config, ui packer.Ui) error {

	stopParams := client.StopParams{
		VMName: showResponse.Name,
		Force:  true,
	}

	if len(config.PortForwardingRules) > 0 {
		// Check if the rule already exists
		existingForwardedPorts := make(map[int]struct{})
		for _, existingNetworkCard := range describeResponse.NetworkCards {
			for _, existingPortForwardingRule := range existingNetworkCard.PortForwardingRules {
				existingForwardedPorts[existingPortForwardingRule.HostPort] = struct{}{}
			}
		}
		for _, wantedPortForwardingRule := range config.PortForwardingRules {
			ui.Say(fmt.Sprintf("Ensuring %s port-forwarding (Guest Port: %s, Host Port: %s, Rule Name: %s)", showResponse.Name, strconv.Itoa(wantedPortForwardingRule.PortForwardingGuestPort), strconv.Itoa(wantedPortForwardingRule.PortForwardingHostPort), wantedPortForwardingRule.PortForwardingRuleName))
			// Check if host port is set already and warn the user
			if _, ok := existingForwardedPorts[wantedPortForwardingRule.PortForwardingHostPort]; ok {
				ui.Error(fmt.Sprintf("Found an existing host port rule (%s)! Skipping without setting...", strconv.Itoa(wantedPortForwardingRule.PortForwardingHostPort)))
				continue
			}
			if err := s.client.Stop(stopParams); err != nil {
				return err
			}
			err := s.client.Modify(showResponse.Name, "add", "port-forwarding", "--host-port", strconv.Itoa(wantedPortForwardingRule.PortForwardingHostPort), "--guest-port", strconv.Itoa(wantedPortForwardingRule.PortForwardingGuestPort), wantedPortForwardingRule.PortForwardingRuleName)
			if !config.PackerConfig.PackerForce { // If force is enabled, just skip
				if err != nil {
					return err
				}
			}
		}
	}

	if config.HWUUID != "" {
		if err := s.client.Stop(stopParams); err != nil {
			return err
		}
		ui.Say(fmt.Sprintf("Modifying VM custom-variable hw.UUID to %s", config.HWUUID))
		err := s.client.Modify(showResponse.Name, "set", "custom-variable", "hw.UUID", config.HWUUID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *StepCreateVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	s.client = state.Get("client").(client.Client)
	sourceVMName := config.SourceVMName

	onError := func(err error) multistep.StepAction {
		return stepError(ui, state, err)
	}

	createSourceVM := false
	installerAppFullName := "anka-packer-base"

	if config.InstallerApp != "" { // If users specifies an InstallerApp and sourceVMName doesn't exist, assume they want to build a new VM template and use the macOS installer version
		createSourceVM = true
		ui.Say(fmt.Sprintf("Extracting version from installer app: %q", config.InstallerApp))
		macOSVersionFromInstallerApp, err := obtainMacOSVersionFromInstallerApp(config.InstallerApp) // Grab the version details from the Info.plist inside of the Installer package
		if err != nil {
			return onError(err)
		}
		installerAppFullName = fmt.Sprintf("%s-%s", installerAppFullName, macOSVersionFromInstallerApp) // We need to set the SourceVMName since the user didn't and the logic below creates a VM using it
	}

	if sourceVMName == "" {
		sourceVMName = installerAppFullName
	}

	// Reuse the base VM template if it matches the one from the installer
	if sourceVMExists, err := s.client.Exists(sourceVMName); err != nil {
		return onError(err)
	} else {
		if sourceVMExists {
			createSourceVM = false
		}
	}

	s.vmName = sourceVMName // Used for cleanup BEFORE THE CLONE

	clonedVMName := config.VMName
	if clonedVMName == "" { // If user doesn't give a vm_name, generate one
		clonedVMName = fmt.Sprintf("anka-packer-%s", randSeq(10))
	}

	if createSourceVM {
		ui.Say(fmt.Sprintf("Creating a new base VM Template (%s) from installer, this will take a while", sourceVMName))
		outputStream := make(chan string)
		go func() {
			for msg := range outputStream {
				ui.Say(msg)
			}
		}()
		createParams := client.CreateParams{
			InstallerApp: config.InstallerApp,
			Name:         sourceVMName,
			DiskSize:     config.DiskSize,
			CPUCount:     config.CPUCount,
			RAMSize:      config.RAMSize,
		}

		if createParams.DiskSize == "" {
			createParams.DiskSize = DEFAULT_DISK_SIZE
		}

		if createParams.CPUCount == "" {
			createParams.CPUCount = DEFAULT_CPU_COUNT
		}

		if createParams.RAMSize == "" {
			createParams.RAMSize = DEFAULT_RAM_SIZE
		}

		if resp, err := s.client.Create(createParams, outputStream); err != nil {
			return onError(err)
		} else {
			ui.Say(fmt.Sprintf("VM %s was created (%s)", sourceVMName, resp.UUID))
		}
		close(outputStream)
	}

	show, err := s.client.Show(sourceVMName)
	if err != nil {
		return onError(err)
	}

	if show.IsRunning() {
		ui.Say(fmt.Sprintf("Suspending VM %s", sourceVMName))
		if err := s.client.Suspend(client.SuspendParams{VMName: sourceVMName}); err != nil {
			return onError(err)
		}
	}

	s.vmName = clonedVMName // Used for cleanup of clone if a failure happens

	// If the user forces the build (packer build --force), delete the existing VM that would fail the build
	exists, err := s.client.Exists(clonedVMName)
	if exists && config.PackerConfig.PackerForce {
		ui.Say(fmt.Sprintf("Deleting existing virtual machine %s", clonedVMName))
		if err = s.client.Delete(client.DeleteParams{VMName: clonedVMName}); err != nil {
			return onError(err)
		}
	}
	if err != nil {
		return onError(err)
	}

	ui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", sourceVMName, clonedVMName))
	if err = s.client.Clone(client.CloneParams{VMName: clonedVMName, SourceUUID: show.UUID}); err != nil {
		return onError(err)
	}

	showResponse, err := s.client.Show(clonedVMName)
	if err != nil {
		return onError(err)
	}

	if err := s.modifyVMResources(showResponse, config, ui); err != nil {
		return onError(err)
	}

	describeResponse, err := s.client.Describe(clonedVMName)
	if err != nil {
		return onError(err)
	}

	if err := s.modifyVMProperties(describeResponse, showResponse, config, ui); err != nil {
		return onError(err)
	}

	state.Put("vm_name", clonedVMName)

	return multistep.ActionContinue
}

func (s *StepCreateVM) Cleanup(state multistep.StateBag) {
	var err error

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
			err = s.client.Delete(client.DeleteParams{VMName: s.vmName})
			if err != nil {
				ui.Error(fmt.Sprint(err))
			}
			return
		}
	}

	err = s.client.Suspend(client.SuspendParams{
		VMName: s.vmName,
	})
	if err != nil {
		ui.Error(fmt.Sprint(err))
		if deleteErr := s.client.Delete(client.DeleteParams{VMName: s.vmName}); err != nil {
			panic(deleteErr)
		}
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
	plistContent, _ := os.Open(plistPath)

	var installAppPlist struct {
		PlatformVersion string `plist:"DTPlatformVersion"`
		ShortVersion    string `plist:"CFBundleShortVersionString"`
	}
	if err = plist.NewXMLDecoder(plistContent).Decode(&installAppPlist); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%s", installAppPlist.PlatformVersion, installAppPlist.ShortVersion), nil
}
