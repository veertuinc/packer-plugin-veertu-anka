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
	sourceVM := config.SourceVMName

	onError := func(err error) multistep.StepAction {
		return stepError(ui, state, err)
	}

	// By default, do not create a new sourceVM
	doCreateSourceVM := false

	// If users specifies both a source name and installer app, assume they wish us to create the
	// source image using the installer app
	if config.InstallerApp != "" && sourceVM != "" {
		doCreateSourceVM = true
	}

	// If no source name was specified but an installer_app was, generate a source name
	// based on reading the version strings embedded into the installer app Contents/Info.plist
	// so that, if this baseVM already exists, we can skip creating it again.
	// This is beneficial for iteration time, because creating the baseVM takes tens of minutes
	// and once it exists, it doesn't change.
	// A user can clear this cached baseVM by `anka delete --yes packer-builder-base-10.15.6-15.7.02`
	if config.InstallerApp != "" && sourceVM == "" {
		ui.Say(fmt.Sprintf("Extracting version from installer app %q", config.InstallerApp))
		baseVMName, err := nameFromInstallerApp(config.InstallerApp)
		if err != nil {
			return onError(err)
		}
		sourceVM = fmt.Sprintf("packer-builder-base-%s", baseVMName)
		if strings.ContainsAny(sourceVM, " \n") {
			return onError(fmt.Errorf("illegal source VM generated, contained spaces - %q", sourceVM))
		}
		doCreateSourceVM = true
	}

	// Do not create the source vm if it already exists
	if sourceVMExists, _ := s.client.Exists(sourceVM); sourceVMExists {
		doCreateSourceVM = false
	}

	if doCreateSourceVM {
		cpuCount, err := strconv.ParseInt(config.CPUCount, 10, 32)
		if err != nil {
			return onError(err)
		}

		ui.Say(fmt.Sprintf("Creating a new vm (%s) from installer, this will take a while", sourceVM))

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
			Name:         sourceVM,
		}, outputStream)
		if err != nil {
			return onError(err)
		}

		close(outputStream)

		ui.Say(fmt.Sprintf("VM %s was created (%s)", sourceVM, resp.UUID))
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

	// If cloned from an existing VM, check if modification is required
	if !doCreateSourceVM {

		showResponse, err := s.client.Show(vmName)
		if err != nil {
			return onError(err)
		}

		stopParams := client.StopParams{
			VMName: showResponse.Name,
			Force:  true,
		}

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

	state.Put("vm_name", vmName)
	s.vmName = vmName

	return multistep.ActionContinue
}

func (s *StepCreateVM) Cleanup(state multistep.StateBag) {
	log.Println("Cleaning up create VM step")
	if s.vmName == "" {
		log.Println("No VM name - skipping this part")
		return
	}
	_, halted := state.GetOk(multistep.StateHalted)
	_, canceled := state.GetOk(multistep.StateCancelled)

	if halted || canceled {
		log.Println("Deleting VM ", s.vmName)
		err := s.client.Delete(client.DeleteParams{VMName: s.vmName})
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

func nameFromInstallerApp(path string) (string, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("installer app did not exist at %q: %w", path, err)
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
