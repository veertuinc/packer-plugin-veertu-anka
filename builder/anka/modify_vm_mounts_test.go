package anka

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestBuildHostDirectoryMountArgument(t *testing.T) {
	t.Run("uses guest folder name when provided", func(t *testing.T) {
		mountArgument := buildHostDirectoryMountArgument(HostDirectoryMount{
			HostPath:        "/tmp/packer-mount",
			GuestFolderName: "packer-mount",
		})

		assert.Equal(t, "/tmp/packer-mount:packer-mount", mountArgument)
	})

	t.Run("uses host path only when guest folder name is omitted", func(t *testing.T) {
		mountArgument := buildHostDirectoryMountArgument(HostDirectoryMount{
			HostPath: "/Users/nathanpierce",
		})

		assert.Equal(t, "/Users/nathanpierce", mountArgument)
	})
}

func TestGuestFolderNameForHostDirectoryMount(t *testing.T) {
	t.Run("returns configured guest folder name", func(t *testing.T) {
		guestFolderName := guestFolderNameForHostDirectoryMount(HostDirectoryMount{
			HostPath:        "/tmp/packer-mount",
			GuestFolderName: "shared-files",
		})

		assert.Equal(t, "shared-files", guestFolderName)
	})

	t.Run("defaults to host path basename", func(t *testing.T) {
		guestFolderName := guestFolderNameForHostDirectoryMount(HostDirectoryMount{
			HostPath: "/Users/nathanpierce",
		})

		assert.Equal(t, "nathanpierce", guestFolderName)
	})
}
