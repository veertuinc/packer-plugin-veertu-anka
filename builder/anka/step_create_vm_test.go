package anka

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
	mocks "github.com/veertuinc/packer-builder-veertu-anka/mocks"
	"github.com/veertuinc/packer-builder-veertu-anka/util"
	"gotest.tools/assert"
)

var createResponse client.CreateResponse

func TestCreateVMRun(t *testing.T) {
	err := json.Unmarshal(json.RawMessage(`{"UUID": "abcd-efgh-1234-5678"}`), &createResponse)
	if err != nil {
		t.Fail()
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ankaClient := mocks.NewMockClient(mockCtrl)
	ankaUtil := mocks.NewMockUtil(mockCtrl)

	step := StepCreateVM{}
	ui := packer.TestUi(t)
	ctx := context.Background()
	state := new(multistep.BasicStateBag)
	installerAppInfo := util.InstallAppPlist{
		OSVersion:      "11.2",
		BundlerVersion: "16.4.06",
	}

	state.Put("ui", ui)
	state.Put("client", ankaClient)
	state.Put("util", ankaUtil)

	t.Run("create vm", func(t *testing.T) {
		config := &Config{
			DiskSize:     "500G",
			VCPUCount:    "32G",
			RAMSize:      "16G",
			InstallerApp: "/fake/InstallApp.app/",
			VMName:       "foo",
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-create",
			},
		}

		state.Put("config", config)

		step.vmName = config.VMName
		state.Put("vm_name", step.vmName)

		createParams := client.CreateParams{
			InstallerApp: config.InstallerApp,
			Name:         step.vmName,
			DiskSize:     config.DiskSize,
			VCPUCount:    config.VCPUCount,
			RAMSize:      config.RAMSize,
		}

		ankaClient.EXPECT().Create(createParams, gomock.Any()).Return(createResponse, nil).Times(1)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Creating a new VM Template (%s) from installer, this will take a while", step.vmName))
		mockui.Say(fmt.Sprintf("VM %s was created (%s)", step.vmName, createResponse.UUID))

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Creating a new VM Template (foo) from installer, this will take a while")
		assert.Equal(t, mockui.SayMessages[1].Message, "VM foo was created (abcd-efgh-1234-5678)")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("create vm with packer force", func(t *testing.T) {
		config := &Config{
			DiskSize:     "500G",
			VCPUCount:    "32G",
			RAMSize:      "16G",
			InstallerApp: "/fake/InstallApp.app/",
			VMName:       "foo",
			PackerConfig: common.PackerConfig{
				PackerForce:       true,
				PackerBuilderType: "veertu-anka-vm-create",
			},
		}

		state.Put("config", config)

		step.vmName = config.VMName
		state.Put("vm_name", step.vmName)

		createParams := client.CreateParams{
			InstallerApp: config.InstallerApp,
			Name:         step.vmName,
			DiskSize:     config.DiskSize,
			VCPUCount:    config.VCPUCount,
			RAMSize:      config.RAMSize,
		}

		gomock.InOrder(
			ankaClient.EXPECT().Exists(step.vmName).Return(true, nil).Times(1),
			ankaClient.EXPECT().Delete(client.DeleteParams{VMName: step.vmName}).Return(nil).Times(1),
			ankaClient.EXPECT().Create(createParams, gomock.Any()).Return(createResponse, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Deleting existing virtual machine %s", step.vmName))
		mockui.Say(fmt.Sprintf("Creating a new VM Template (%s) from installer, this will take a while", step.vmName))
		mockui.Say(fmt.Sprintf("VM %s was created (%s)", step.vmName, createResponse.UUID))

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Deleting existing virtual machine foo")
		assert.Equal(t, mockui.SayMessages[1].Message, "Creating a new VM Template (foo) from installer, this will take a while")
		assert.Equal(t, mockui.SayMessages[2].Message, "VM foo was created (abcd-efgh-1234-5678)")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("create vm and installer app does not exist", func(t *testing.T) {
		config := &Config{
			DiskSize:     "500G",
			VCPUCount:    "32G",
			RAMSize:      "16G",
			InstallerApp: "/does/not/exist/InstallApp.app/",
			VMName:       "foo",
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-create",
			},
		}

		state.Put("config", config)

		step.vmName = config.VMName
		state.Put("vm_name", step.vmName)

		createParams := client.CreateParams{
			InstallerApp: config.InstallerApp,
			Name:         step.vmName,
			DiskSize:     config.DiskSize,
			VCPUCount:    config.VCPUCount,
			RAMSize:      config.RAMSize,
		}

		gomock.InOrder(
			ankaClient.EXPECT().
				Create(createParams, gomock.Any()).
				Return(client.CreateResponse{}, fmt.Errorf("installer app does not exist at %q", config.InstallerApp)).
				Times(1),
			ankaUtil.EXPECT().
				StepError(ui, state, fmt.Errorf("installer app does not exist at %q", config.InstallerApp)).
				Return(multistep.ActionHalt).
				Times(1),
		)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, multistep.ActionHalt, stepAction)
	})

	t.Run("create vm when no vm_name is provided in config", func(t *testing.T) {
		config := &Config{
			DiskSize:     "500G",
			VCPUCount:    "32G",
			RAMSize:      "16G",
			InstallerApp: "/fake/InstallApp.app/",
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-create",
			},
		}

		state.Put("config", config)

		step.vmName = fmt.Sprintf("anka-packer-base-%s-%s", installerAppInfo.OSVersion, installerAppInfo.BundlerVersion)
		state.Put("vm_name", step.vmName)

		createParams := client.CreateParams{
			InstallerApp: config.InstallerApp,
			Name:         step.vmName,
			DiskSize:     config.DiskSize,
			VCPUCount:    config.VCPUCount,
			RAMSize:      config.RAMSize,
		}

		gomock.InOrder(
			ankaUtil.EXPECT().ObtainMacOSVersionFromInstallerApp(config.InstallerApp).Return(installerAppInfo, nil).Times(1),
			ankaClient.EXPECT().Create(createParams, gomock.Any()).Return(createResponse, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Creating a new VM Template (%s) from installer, this will take a while", step.vmName))
		mockui.Say(fmt.Sprintf("VM %s was created (%s)", step.vmName, createResponse.UUID))

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Creating a new VM Template (anka-packer-base-11.2-16.4.06) from installer, this will take a while")
		assert.Equal(t, mockui.SayMessages[1].Message, "VM anka-packer-base-11.2-16.4.06 was created (abcd-efgh-1234-5678)")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})
}
