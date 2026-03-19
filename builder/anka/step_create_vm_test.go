package anka

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-plugin-veertu-anka/client"
	mocks "github.com/veertuinc/packer-plugin-veertu-anka/mocks"
	"github.com/veertuinc/packer-plugin-veertu-anka/util"
	"gotest.tools/v3/assert"
)

var (
	createdShowResponse     client.ShowResponse
	createdDescribeResponse client.DescribeResponse
)

func TestCreateVMRun(t *testing.T) {

	createdVMUUID := "abcd-efgh-1234-5678"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ankaClient := mocks.NewMockClient(mockCtrl)
	ankaUtil := mocks.NewMockUtil(mockCtrl)

	step := StepCreateVM{}
	ui := packer.TestUi(t)
	ctx := context.Background()
	state := new(multistep.BasicStateBag)
	InstallerInfo := util.InstallerAppPlist{
		OSVersion:      "11.2",
		BundlerVersion: "16.4.06",
	}

	state.Put("ui", ui)
	state.Put("client", ankaClient)
	state.Put("util", ankaUtil)

	err = json.Unmarshal(json.RawMessage(`{ "Name": "anka-packer-base-11.2-16.4.06", "UUID": "1234-hijk-abcdef-5678" }`), &createdShowResponse)
	if err != nil {
		t.Fail()
	}

	err = json.Unmarshal(json.RawMessage(`{  }`), &createdDescribeResponse)
	if err != nil {
		t.Fail()
	}

	t.Run("create vm", func(t *testing.T) {
		config := &Config{
			DiskSize:  "500G",
			VCPUCount: "32G",
			RAMSize:   "16G",
			Installer: "/fake/InstallApp.app/",
			VMName:    "foo",
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-create",
			},
		}

		state.Put("config", config)

		step.vmName = config.VMName
		state.Put("vm_name", step.vmName)

		createParams := client.CreateParams{
			Installer: config.Installer,
			Name:      step.vmName,
			DiskSize:  config.DiskSize,
			VCPUCount: config.VCPUCount,
			RAMSize:   config.RAMSize,
		}

		gomock.InOrder(
			ankaClient.EXPECT().Create(createParams, gomock.Any()).Return(createdVMUUID, nil).Times(1),
			ankaClient.EXPECT().Show(step.vmName).Return(createdShowResponse, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Creating a new VM Template (%s) from installer, this will take a while", step.vmName))
		mockui.Say(fmt.Sprintf("VM %s was created (%s)", step.vmName, createdVMUUID))

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Creating a new VM Template (foo) from installer, this will take a while")
		assert.Equal(t, mockui.SayMessages[1].Message, "VM foo was created (abcd-efgh-1234-5678)")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("create vm without .app or ipsw", func(t *testing.T) {
		config := &Config{
			DiskSize:  "500G",
			VCPUCount: "32G",
			RAMSize:   "16G",
			Installer: "13.5",
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-create",
			},
		}

		state.Put("config", config)

		step.vmName = "anka-packer-base-13.5"
		state.Put("vm_name", step.vmName)

		createParams := client.CreateParams{
			Installer: config.Installer,
			Name:      step.vmName,
			DiskSize:  config.DiskSize,
			VCPUCount: config.VCPUCount,
			RAMSize:   config.RAMSize,
		}

		gomock.InOrder(
			ankaClient.EXPECT().Create(createParams, gomock.Any()).Return(createdVMUUID, nil).Times(1),
			ankaClient.EXPECT().Show(step.vmName).Return(createdShowResponse, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Creating a new VM Template (%s) from installer, this will take a while", step.vmName))
		mockui.Say(fmt.Sprintf("VM %s was created (%s)", step.vmName, createdVMUUID))

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, fmt.Sprintf("Creating a new VM Template (%s) from installer, this will take a while", step.vmName))
		assert.Equal(t, mockui.SayMessages[1].Message, fmt.Sprintf("VM %s was created (abcd-efgh-1234-5678)", step.vmName))
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("create vm with packer force", func(t *testing.T) {
		config := &Config{
			DiskSize:  "500G",
			VCPUCount: "32G",
			RAMSize:   "16G",
			Installer: "/fake/InstallApp.app/",
			VMName:    "foo",
			PackerConfig: common.PackerConfig{
				PackerForce:       true,
				PackerBuilderType: "veertu-anka-vm-create",
			},
		}

		state.Put("config", config)

		step.vmName = config.VMName
		state.Put("vm_name", step.vmName)

		createParams := client.CreateParams{
			Installer: config.Installer,
			Name:      step.vmName,
			DiskSize:  config.DiskSize,
			VCPUCount: config.VCPUCount,
			RAMSize:   config.RAMSize,
		}

		gomock.InOrder(
			ankaClient.EXPECT().Exists(step.vmName).Return(true, nil).Times(1),
			ankaClient.EXPECT().Delete(client.DeleteParams{VMName: step.vmName}).Return(nil).Times(1),
			ankaClient.EXPECT().Create(createParams, gomock.Any()).Return(createdVMUUID, nil).Times(1),
			ankaClient.EXPECT().Show(step.vmName).Return(createdShowResponse, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Deleting existing virtual machine %s", step.vmName))
		mockui.Say(fmt.Sprintf("Creating a new VM Template (%s) from installer, this will take a while", step.vmName))
		mockui.Say(fmt.Sprintf("VM %s was created (%s)", step.vmName, createdVMUUID))

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Deleting existing virtual machine foo")
		assert.Equal(t, mockui.SayMessages[1].Message, "Creating a new VM Template (foo) from installer, this will take a while")
		assert.Equal(t, mockui.SayMessages[2].Message, "VM foo was created (abcd-efgh-1234-5678)")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("create vm and installer app does not exist", func(t *testing.T) {
		config := &Config{
			DiskSize:  "500G",
			VCPUCount: "32G",
			RAMSize:   "16G",
			Installer: "/does/not/exist/InstallApp.app/",
			VMName:    "foo",
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-create",
			},
		}

		state.Put("config", config)

		step.vmName = config.VMName
		state.Put("vm_name", step.vmName)

		createParams := client.CreateParams{
			Installer: config.Installer,
			Name:      step.vmName,
			DiskSize:  config.DiskSize,
			VCPUCount: config.VCPUCount,
			RAMSize:   config.RAMSize,
		}

		gomock.InOrder(
			ankaClient.EXPECT().
				Create(createParams, gomock.Any()).
				Return(createdVMUUID, fmt.Errorf("installer app does not exist at %q", config.Installer)).
				Times(1),
			ankaUtil.EXPECT().
				StepError(ui, state, fmt.Errorf("installer app does not exist at %q", config.Installer)).
				Return(multistep.ActionHalt).
				Times(1),
		)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, multistep.ActionHalt, stepAction)
	})

	t.Run("create vm when no vm_name is provided in config", func(t *testing.T) {
		config := &Config{
			DiskSize:  "500G",
			VCPUCount: "32G",
			RAMSize:   "16G",
			Installer: "/fake/InstallApp.app/",
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-create",
			},
		}

		state.Put("config", config)

		step.vmName = fmt.Sprintf("anka-packer-base-%s-%s", InstallerInfo.OSVersion, InstallerInfo.BundlerVersion)
		state.Put("vm_name", step.vmName)

		createParams := client.CreateParams{
			Installer: config.Installer,
			Name:      step.vmName,
			DiskSize:  config.DiskSize,
			VCPUCount: config.VCPUCount,
			RAMSize:   config.RAMSize,
		}

		gomock.InOrder(
			ankaUtil.EXPECT().ObtainMacOSVersionFromInstallerApp(config.Installer).Return(InstallerInfo, nil).Times(1),
			ankaClient.EXPECT().Create(createParams, gomock.Any()).Return(createdVMUUID, nil).Times(1),
			ankaClient.EXPECT().Show(step.vmName).Return(createdShowResponse, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Creating a new VM Template (%s) from installer, this will take a while", step.vmName))
		mockui.Say(fmt.Sprintf("VM %s was created (%s)", step.vmName, createdVMUUID))

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Creating a new VM Template (anka-packer-base-11.2-16.4.06) from installer, this will take a while")
		assert.Equal(t, mockui.SayMessages[1].Message, "VM anka-packer-base-11.2-16.4.06 was created (abcd-efgh-1234-5678)")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("create vm with modify vm properties", func(t *testing.T) {

		err = json.Unmarshal(json.RawMessage(`{ "Name": "anka-packer-base-latest", "UUID": "1234-hijk-abcdef-5678" }`), &createdShowResponse)
		if err != nil {
			t.Fail()
		}

		var config Config
		err = json.Unmarshal(json.RawMessage(`
			{
				"PortForwardingRules": [
					{
						"PortForwardingGuestPort": 8080,
						"PortForwardingHostPort": 80,
						"PortForwardingRuleName": "rule1"
					}
				],
				"DiskSize":  "500G",
				"VCPUCount": "32G",
				"RAMSize":   "16G",
				"Installer": "latest",
				"HWUUID": "abcdefgh",
				"DisplayController": "pg",
				"DisplayResolution": "1920x1080"
			}
		`), &config)
		if err != nil {
			t.Fail()
		}

		config.PackerConfig = common.PackerConfig{
			PackerBuilderType: "veertu-anka-vm-create",
		}

		state.Put("config", config)

		step.vmName = fmt.Sprintf("anka-packer-base-%s", config.Installer)
		state.Put("vm_name", step.vmName)

		createParams := client.CreateParams{
			Installer: config.Installer,
			Name:      step.vmName,
			DiskSize:  config.DiskSize,
			VCPUCount: config.VCPUCount,
			RAMSize:   config.RAMSize,
		}

		stopParams := client.StopParams{
			VMName: createdShowResponse.Name,
		}

		state.Put("config", &config)

		gomock.InOrder(
			ankaClient.EXPECT().Create(createParams, gomock.Any()).Return(createdVMUUID, nil).Times(1),
			ankaClient.EXPECT().Show(step.vmName).Return(createdShowResponse, nil).Times(1),
			ankaClient.EXPECT().Describe(step.vmName).Return(client.DescribeResponse{}, nil).Times(1),
			ankaClient.EXPECT().Stop(stopParams).Return(nil).Times(1),
			ankaClient.EXPECT().
				Modify(createdShowResponse.Name, "add", "port-forwarding", "--host-port", strconv.Itoa(config.PortForwardingRules[0].PortForwardingHostPort), "--guest-port", strconv.Itoa(config.PortForwardingRules[0].PortForwardingGuestPort), "rule1").
				Return(nil).
				Times(1),
			ankaClient.EXPECT().Stop(stopParams).Return(nil).Times(1),
			ankaClient.EXPECT().Modify(createdShowResponse.Name, "set", "custom-variable", "hw.uuid", config.HWUUID).Return(nil).Times(1),
			ankaClient.EXPECT().Stop(stopParams).Return(nil).Times(1),
			ankaClient.EXPECT().Modify(createdShowResponse.Name, "set", "display", "-c", config.DisplayController).Return(nil).Times(1),
			ankaClient.EXPECT().Stop(stopParams).Return(nil).Times(1),
			ankaClient.EXPECT().Modify(createdShowResponse.Name, "set", "display", "-r", config.DisplayResolution).Return(nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Creating a new VM Template (%s) from installer, this will take a while", step.vmName))
		mockui.Say(fmt.Sprintf("VM %s was created (%s)", step.vmName, createdVMUUID))
		mockui.Say(fmt.Sprintf("Ensuring %s port-forwarding (Guest Port: %s, Host Port: %s, Rule Name: %s)", createdShowResponse.Name, strconv.Itoa(config.PortForwardingRules[0].PortForwardingGuestPort), strconv.Itoa(config.PortForwardingRules[0].PortForwardingHostPort), config.PortForwardingRules[0].PortForwardingRuleName))
		mockui.Say(fmt.Sprintf("Modifying VM custom-variable hw.uuid to %s", config.HWUUID))
		mockui.Say(fmt.Sprintf("Modifying VM display controller to %s", config.DisplayController))
		mockui.Say(fmt.Sprintf("Modifying VM display resolution to %s", config.DisplayResolution))

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Creating a new VM Template (anka-packer-base-latest) from installer, this will take a while")
		assert.Equal(t, mockui.SayMessages[1].Message, "VM anka-packer-base-latest was created (abcd-efgh-1234-5678)")
		assert.Equal(t, multistep.ActionContinue, stepAction)

	})
}
