package anka

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-plugin-veertu-anka/client"
)

// HostDirectoryMount defines a persistent host directory mount for the VM template.
type HostDirectoryMount struct {
	HostPath        string `mapstructure:"host_path"`
	GuestFolderName string `mapstructure:"guest_folder_name,omitempty"`
}

func buildHostDirectoryMountArgument(hostDirectoryMount HostDirectoryMount) string {
	if hostDirectoryMount.GuestFolderName != "" {
		return hostDirectoryMount.HostPath + ":" + hostDirectoryMount.GuestFolderName
	}

	return hostDirectoryMount.HostPath
}

func guestFolderNameForHostDirectoryMount(hostDirectoryMount HostDirectoryMount) string {
	if hostDirectoryMount.GuestFolderName != "" {
		return hostDirectoryMount.GuestFolderName
	}

	return filepath.Base(strings.TrimRight(hostDirectoryMount.HostPath, string(filepath.Separator)))
}

func applyHostDirectoryMounts(
	ankaClient client.Client,
	stopParams client.StopParams,
	vmName string,
	hostDirectoryMounts []HostDirectoryMount,
	packerForce bool,
	ui packer.Ui,
) error {
	for _, hostDirectoryMount := range hostDirectoryMounts {
		mountArgument := buildHostDirectoryMountArgument(hostDirectoryMount)
		guestFolderName := guestFolderNameForHostDirectoryMount(hostDirectoryMount)

		ui.Say(fmt.Sprintf(
			"Ensuring %s host directory mount (Host Folder: %s, Guest Folder: %s)",
			vmName,
			hostDirectoryMount.HostPath,
			guestFolderName,
		))

		err := ankaClient.Stop(stopParams)
		if err != nil {
			return err
		}

		err = ankaClient.Modify(vmName, "mount", mountArgument)
		if !packerForce && err != nil {
			return err
		}
	}

	return nil
}
