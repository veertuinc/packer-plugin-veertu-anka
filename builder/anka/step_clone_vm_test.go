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
	"gotest.tools/v3/assert"
)

var (
	sourceShowResponse     client.ShowResponse
	clonedShowResponse     client.ShowResponse
	clonedDescribeResponse client.DescribeResponse
)

func TestCloneVMRun(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ankaClient := mocks.NewMockClient(mockCtrl)
	ankaUtil := mocks.NewMockUtil(mockCtrl)

	step := StepCloneVM{}
	ui := packer.TestUi(t)
	ctx := context.Background()
	state := new(multistep.BasicStateBag)

	state.Put("ui", ui)
	state.Put("client", ankaClient)
	state.Put("util", ankaUtil)

	err := json.Unmarshal(json.RawMessage(`{ "UUID": "1234-abcdef-hijk-5678", "Name": "source_foo" }`), &sourceShowResponse)
	if err != nil {
		t.Fail()
	}

	err = json.Unmarshal(json.RawMessage(`{ "Name": "foo", "UUID": "1234-hijk-abcdef-5678" }`), &clonedShowResponse)
	if err != nil {
		t.Fail()
	}

	t.Run("clone vm", func(t *testing.T) {
		config := &Config{
			VMName:       "foo",
			SourceVMName: "source_foo",
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-clone",
			},
		}

		step.vmName = config.VMName

		state.Put("vm_name", step.vmName)
		state.Put("config", config)

		gomock.InOrder(
			ankaClient.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			ankaClient.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			ankaClient.EXPECT().Clone(client.CloneParams{VMName: step.vmName, SourceUUID: sourceShowResponse.UUID}).Return(nil).Times(1),
			ankaClient.EXPECT().Show(step.vmName).Return(clonedShowResponse, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", sourceShowResponse.Name, step.vmName))

		stepAction := step.Run(ctx, state)

		assert.Equal(t, mockui.SayMessages[0].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("clone vm when no vm_name was provided in config", func(t *testing.T) {
		config := &Config{
			SourceVMName: "source_foo",
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-clone",
			},
		}

		step.vmName = fmt.Sprintf("%s-%s", config.SourceVMName, "ABCDEabcde")

		state.Put("vm_name", step.vmName)
		state.Put("config", config)

		gomock.InOrder(
			ankaUtil.EXPECT().RandSeq(10).Return("ABCDEabcde").Times(1),
			ankaClient.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			ankaClient.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			ankaClient.EXPECT().Clone(client.CloneParams{VMName: step.vmName, SourceUUID: sourceShowResponse.UUID}).Return(nil).Times(1),
			ankaClient.EXPECT().Show(step.vmName).Return(clonedShowResponse, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", sourceShowResponse.Name, step.vmName))

		stepAction := step.Run(ctx, state)

		assert.Equal(t, mockui.SayMessages[0].Message, "Cloning source VM source_foo into a new virtual machine: source_foo-ABCDEabcde")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("clone vm with packer force", func(t *testing.T) {
		config := &Config{
			VMName:       "foo",
			SourceVMName: "source_foo",
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-clone",
				PackerForce:       true,
			},
		}

		step.vmName = config.VMName

		state.Put("vm_name", step.vmName)
		state.Put("config", config)

		// force delete
		gomock.InOrder(
			ankaClient.EXPECT().Exists(step.vmName).Return(true, nil).Times(1),
			ankaClient.EXPECT().Delete(client.DeleteParams{VMName: step.vmName}).Return(nil).Times(1),
		)

		gomock.InOrder(
			ankaClient.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			ankaClient.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			ankaClient.EXPECT().Clone(client.CloneParams{VMName: step.vmName, SourceUUID: sourceShowResponse.UUID}).Return(nil).Times(1),
			ankaClient.EXPECT().Show(step.vmName).Return(clonedShowResponse, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", sourceShowResponse.Name, step.vmName))
		mockui.Say(fmt.Sprintf("Deleting existing virtual machine %s", step.vmName))

		stepAction := step.Run(ctx, state)

		assert.Equal(t, mockui.SayMessages[0].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, mockui.SayMessages[1].Message, "Deleting existing virtual machine foo")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("clone vm when source vm does not exist locally it should pull from registry", func(t *testing.T) {
		config := &Config{
			VMName:       "foo",
			SourceVMName: "source_foo",
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-clone",
			},
		}
		sourceVMTag := ""
		registryParams := client.RegistryParams{}
		registryPullParams := client.RegistryPullParams{
			VMID:   config.SourceVMName,
			Tag:    sourceVMTag,
			Local:  false,
			Shrink: false,
		}

		step.vmName = config.VMName

		state.Put("vm_name", step.vmName)
		state.Put("config", config)

		gomock.InOrder(
			ankaClient.EXPECT().Exists(config.SourceVMName).Return(false, nil).Times(1),
			ankaClient.EXPECT().RegistryPull(registryParams, registryPullParams).Return(nil).Times(1),
			ankaClient.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			ankaClient.EXPECT().Clone(client.CloneParams{VMName: step.vmName, SourceUUID: sourceShowResponse.UUID}).Return(nil).Times(1),
			ankaClient.EXPECT().Show(step.vmName).Return(clonedShowResponse, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", sourceShowResponse.Name, step.vmName))

		stepAction := step.Run(ctx, state)

		assert.Equal(t, mockui.SayMessages[0].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("clone vm when source vm does not exist at all it should throw error", func(t *testing.T) {
		config := &Config{
			VMName:       "foo",
			SourceVMName: "source_foo",
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-clone",
			},
		}
		sourceVMTag := ""
		registryParams := client.RegistryParams{}
		registryPullParams := client.RegistryPullParams{
			VMID:   config.SourceVMName,
			Tag:    sourceVMTag,
			Local:  false,
			Shrink: false,
		}

		step.vmName = config.VMName

		state.Put("vm_name", step.vmName)
		state.Put("config", config)

		gomock.InOrder(
			ankaClient.EXPECT().Exists(config.SourceVMName).Return(false, nil).Times(1),
			ankaClient.EXPECT().
				RegistryPull(registryParams, registryPullParams).
				Return(fmt.Errorf("failed to pull vm %v with latest tag from registry (make sure to add it as the default: https://docs.veertu.com/anka/intel/command-line-reference/#registry-add)", config.SourceVMName)).
				Times(1),
			ankaUtil.EXPECT().
				StepError(ui, state, fmt.Errorf("failed to pull vm %v with latest tag from registry (make sure to add it as the default: https://docs.veertu.com/anka/intel/command-line-reference/#registry-add)", config.SourceVMName)).
				Return(multistep.ActionHalt).
				Times(1),
		)

		stepAction := step.Run(ctx, state)

		assert.Equal(t, multistep.ActionHalt, stepAction)
	})

	t.Run("clone vm with always fetch flag should only pull from anka registry", func(t *testing.T) {
		config := &Config{
			AlwaysFetch:  true,
			VMName:       "foo",
			SourceVMName: "source_foo",
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-clone",
			},
		}
		sourceVMTag := ""
		registryParams := client.RegistryParams{}
		registryPullParams := client.RegistryPullParams{
			VMID:   config.SourceVMName,
			Tag:    sourceVMTag,
			Local:  false,
			Shrink: false,
		}

		step.vmName = config.VMName

		state.Put("vm_name", step.vmName)
		state.Put("config", config)

		gomock.InOrder(
			ankaClient.EXPECT().RegistryPull(registryParams, registryPullParams).Return(nil).Times(1),
			ankaClient.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			ankaClient.EXPECT().Clone(client.CloneParams{VMName: step.vmName, SourceUUID: sourceShowResponse.UUID}).Return(nil).Times(1),
			ankaClient.EXPECT().Show(step.vmName).Return(clonedShowResponse, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Pulling source VM %s with latest tag from Anka Registry", config.SourceVMName))
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", sourceShowResponse.Name, step.vmName))

		stepAction := step.Run(ctx, state)

		assert.Equal(t, mockui.SayMessages[0].Message, "Pulling source VM source_foo with latest tag from Anka Registry")
		assert.Equal(t, mockui.SayMessages[1].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("pull from anka registry with specific tag", func(t *testing.T) {
		sourceVMTag := "vanilla"
		config := &Config{
			AlwaysFetch:  true,
			VMName:       "foo",
			SourceVMName: "source_foo",
			SourceVMTag:  sourceVMTag,
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-clone",
			},
		}
		registryParams := client.RegistryParams{}
		registryPullParams := client.RegistryPullParams{
			VMID:   config.SourceVMName,
			Tag:    sourceVMTag,
			Local:  false,
			Shrink: false,
		}

		step.vmName = config.VMName

		state.Put("vm_name", step.vmName)
		state.Put("config", config)

		gomock.InOrder(
			ankaClient.EXPECT().RegistryPull(registryParams, registryPullParams).Return(nil).Times(1),
			ankaClient.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			ankaClient.EXPECT().Clone(client.CloneParams{VMName: step.vmName, SourceUUID: sourceShowResponse.UUID}).Return(nil).Times(1),
			ankaClient.EXPECT().Show(step.vmName).Return(clonedShowResponse, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Pulling source VM %s with vanilla tag from Anka Registry", config.SourceVMName))
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", sourceShowResponse.Name, step.vmName))

		stepAction := step.Run(ctx, state)

		assert.Equal(t, mockui.SayMessages[0].Message, "Pulling source VM source_foo with vanilla tag from Anka Registry")
		assert.Equal(t, mockui.SayMessages[1].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("clone vm with always fetch flag when source vm does not exist in anka registry should throw error", func(t *testing.T) {
		config := &Config{
			AlwaysFetch:  true,
			VMName:       "foo",
			SourceVMName: "source_foo",
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-clone",
			},
		}
		sourceVMTag := ""
		registryParams := client.RegistryParams{}
		registryPullParams := client.RegistryPullParams{
			VMID:   config.SourceVMName,
			Tag:    sourceVMTag,
			Local:  false,
			Shrink: false,
		}

		step.vmName = config.VMName

		state.Put("vm_name", step.vmName)
		state.Put("config", config)

		gomock.InOrder(
			ankaClient.EXPECT().
				RegistryPull(registryParams, registryPullParams).
				Return(fmt.Errorf("failed to pull vm %v with latest from registry (make sure to add it as the default: https://docs.veertu.com/anka/intel/command-line-reference/#registry-add)", config.SourceVMName)).
				Times(1),
			ankaUtil.EXPECT().
				StepError(ui, state, fmt.Errorf("failed to pull vm %v with latest tag from registry (make sure to add it as the default: https://docs.veertu.com/anka/intel/command-line-reference/#registry-add)", config.SourceVMName)).
				Return(multistep.ActionHalt).
				Times(1),
		)

		stepAction := step.Run(ctx, state)

		assert.Equal(t, multistep.ActionHalt, stepAction)
	})

	t.Run("clone vm and modify vm resources", func(t *testing.T) {
		err = json.Unmarshal(json.RawMessage(`{ "Name": "foo", "CPUCores": 8, "HardDrive": 40, "RAM": "8G", "UUID": "123456" }`), &clonedShowResponse)
		if err != nil {
			t.Fail()
		}

		config := &Config{
			VCPUCount:    "4",
			DiskSize:     "120G",
			RAMSize:      "16G",
			VMName:       "foo",
			SourceVMName: "source_foo",
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-clone",
			},
		}
		stopParams := client.StopParams{
			VMName: clonedShowResponse.Name,
		}
		runParams := client.RunParams{
			VMName:  clonedShowResponse.Name,
			Command: []string{"diskutil", "apfs", "resizeContainer", "disk0s2", "0"},
		}

		step.vmName = config.VMName

		state.Put("vm_name", step.vmName)
		state.Put("config", config)

		gomock.InOrder(
			ankaClient.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			ankaClient.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			ankaClient.EXPECT().Clone(client.CloneParams{VMName: step.vmName, SourceUUID: sourceShowResponse.UUID}).Return(nil).Times(1),
			ankaClient.EXPECT().Show(step.vmName).Return(clonedShowResponse, nil).Times(1),
		)

		// disksize
		gomock.InOrder(
			ankaUtil.EXPECT().ConvertDiskSizeToBytes(config.DiskSize).Return(uint64(120*1024*1024*1024), nil).Times(1),
			ankaClient.EXPECT().Stop(stopParams).Return(nil).Times(1),
			ankaClient.EXPECT().Modify(clonedShowResponse.Name, "set", "hard-drive", "-s", config.DiskSize).Return(nil).Times(1),
			ankaClient.EXPECT().Run(runParams).Return(0, nil).Times(1),
			ankaClient.EXPECT().Stop(stopParams).Return(nil).Times(1),
		)

		// ramsize
		gomock.InOrder(
			ankaClient.EXPECT().Modify(clonedShowResponse.Name, "set", "ram", config.RAMSize).Return(nil).Times(1),
			ankaClient.EXPECT().Stop(stopParams).Return(nil).Times(1),
		)

		// vcpucount
		gomock.InOrder(
			ankaClient.EXPECT().Stop(stopParams).Return(nil).Times(1),
			ankaClient.EXPECT().Modify(clonedShowResponse.Name, "set", "cpu", "-c", config.VCPUCount).Return(nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", sourceShowResponse.Name, step.vmName))
		mockui.Say(fmt.Sprintf("Modifying VM %s disk size to %s", clonedShowResponse.Name, config.DiskSize))
		mockui.Say(fmt.Sprintf("Modifying VM %s RAM to %s", clonedShowResponse.Name, config.RAMSize))
		mockui.Say(fmt.Sprintf("Modifying VM %s VCPU core count to %v", clonedShowResponse.Name, config.VCPUCount))

		stepAction := step.Run(ctx, state)

		assert.Equal(t, mockui.SayMessages[0].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, mockui.SayMessages[1].Message, "Modifying VM foo disk size to 120G")
		assert.Equal(t, mockui.SayMessages[2].Message, "Modifying VM foo RAM to 16G")
		assert.Equal(t, mockui.SayMessages[3].Message, "Modifying VM foo VCPU core count to 4")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("clone vm with modify vm properties", func(t *testing.T) {
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
				"SourceVMName": "source_foo",
				"VMName": "foo",
				"HWUUID": "abcdefgh",
				"DisplayController": "pg",
				"DisplayResolution": "1920x1080"
			}
		`), &config)
		if err != nil {
			t.Fail()
		}

		err = json.Unmarshal(json.RawMessage(`{  }`), &clonedDescribeResponse)
		if err != nil {
			t.Fail()
		}

		stopParams := client.StopParams{
			VMName: clonedShowResponse.Name,
		}

		step.vmName = config.VMName

		state.Put("vm_name", step.vmName)
		state.Put("config", &config)

		gomock.InOrder(
			ankaClient.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			ankaClient.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			ankaClient.EXPECT().Clone(client.CloneParams{VMName: step.vmName, SourceUUID: sourceShowResponse.UUID}).Return(nil).Times(1),
			ankaClient.EXPECT().Show(step.vmName).Return(clonedShowResponse, nil).Times(1),
		)

		// port forwarding rules
		gomock.InOrder(
			ankaClient.EXPECT().Describe(config.VMName).Return(client.DescribeResponse{}, nil).Times(1),
			ankaClient.EXPECT().Stop(stopParams).Return(nil).Times(1),
			ankaClient.EXPECT().
				Modify(clonedShowResponse.Name, "add", "port-forwarding", "--host-port", strconv.Itoa(config.PortForwardingRules[0].PortForwardingHostPort), "--guest-port", strconv.Itoa(config.PortForwardingRules[0].PortForwardingGuestPort), "rule1").
				Return(nil).
				Times(1),
		)

		// hwuuid
		gomock.InOrder(
			ankaClient.EXPECT().Stop(stopParams).Return(nil).Times(1),
			ankaClient.EXPECT().Modify(clonedShowResponse.Name, "set", "custom-variable", "hw.uuid", config.HWUUID).Return(nil).Times(1),
		)

		// display_controller
		gomock.InOrder(
			ankaClient.EXPECT().Stop(stopParams).Return(nil).Times(1),
			ankaClient.EXPECT().Modify(clonedShowResponse.Name, "set", "display", "-c", config.DisplayController).Return(nil).Times(1),
		)

		gomock.InOrder(
			ankaClient.EXPECT().Stop(stopParams).Return(nil).Times(1),
			ankaClient.EXPECT().Modify(clonedShowResponse.Name, "set", "display", "-r", config.DisplayResolution).Return(nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", config.SourceVMName, config.VMName))
		mockui.Say(fmt.Sprintf("Ensuring %s port-forwarding (Guest Port: %s, Host Port: %s, Rule Name: %s)", clonedShowResponse.Name, strconv.Itoa(config.PortForwardingRules[0].PortForwardingGuestPort), strconv.Itoa(config.PortForwardingRules[0].PortForwardingHostPort), config.PortForwardingRules[0].PortForwardingRuleName))
		mockui.Say(fmt.Sprintf("Modifying VM custom-variable hw.uuid to %s", config.HWUUID))
		mockui.Say(fmt.Sprintf("Modifying VM display controller to %s", config.DisplayController))
		mockui.Say(fmt.Sprintf("Modifying VM display resolution to %s", config.DisplayResolution))

		state.Put("vm_name", config.VMName)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, mockui.SayMessages[1].Message, "Ensuring foo port-forwarding (Guest Port: 8080, Host Port: 80, Rule Name: rule1)")
		assert.Equal(t, mockui.SayMessages[2].Message, "Modifying VM custom-variable hw.uuid to abcdefgh")
		assert.Equal(t, mockui.SayMessages[3].Message, "Modifying VM display controller to pg")
		assert.Equal(t, mockui.SayMessages[4].Message, "Modifying VM display resolution to 1920x1080")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("clone vm with modify vm properties with already enabled port forwarding rules should skip with error", func(t *testing.T) {
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
				"SourceVMName": "source_foo",
				"VMName": "foo"
			}
		`), &config)
		if err != nil {
			t.Fail()
		}

		err = json.Unmarshal(json.RawMessage(`
			{
				"network_cards": [
					{
						"port_forwarding_rules": [
							{
								"guest_port": 8080,
								"host_port": 80,
								"rule_name": "rule1"
							}
						]
					}
				]
			}
		`), &clonedDescribeResponse)
		if err != nil {
			t.Fail()
		}

		step.vmName = config.VMName

		state.Put("vm_name", step.vmName)
		state.Put("config", &config)

		gomock.InOrder(
			ankaClient.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			ankaClient.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			ankaClient.EXPECT().Clone(client.CloneParams{VMName: step.vmName, SourceUUID: sourceShowResponse.UUID}).Return(nil).Times(1),
			ankaClient.EXPECT().Show(step.vmName).Return(clonedShowResponse, nil).Times(1),
			ankaClient.EXPECT().Describe(config.VMName).Return(clonedDescribeResponse, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", config.SourceVMName, config.VMName))
		mockui.Say(fmt.Sprintf("Ensuring %s port-forwarding (Guest Port: %s, Host Port: %s, Rule Name: %s)", clonedShowResponse.Name, strconv.Itoa(config.PortForwardingRules[0].PortForwardingGuestPort), strconv.Itoa(config.PortForwardingRules[0].PortForwardingHostPort), config.PortForwardingRules[0].PortForwardingRuleName))
		mockui.Error(fmt.Sprintf("Found an existing host port rule (%s)! Skipping without setting...", strconv.Itoa(config.PortForwardingRules[0].PortForwardingHostPort)))

		state.Put("vm_name", config.VMName)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, mockui.SayMessages[1].Message, "Ensuring foo port-forwarding (Guest Port: 8080, Host Port: 80, Rule Name: rule1)")
		assert.Equal(t, mockui.ErrorCalled, true)
		assert.Equal(t, mockui.ErrorMessage, "Found an existing host port rule (80)! Skipping without setting...")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("[ARM] clone vm and create local tag if none exists", func(t *testing.T) {
		config := &Config{
			VMName:       "foo",
			SourceVMName: "source_foo",
			PackerConfig: common.PackerConfig{
				PackerBuilderType: "veertu-anka-vm-clone",
			},
			HostArch: "arm64",
		}

		registryParams := client.RegistryParams{
			HostArch: config.HostArch,
		}
		registryPushParams := client.RegistryPushParams{
			Tag:      "local-tag-123",
			RemoteVM: "",
			Local:    true,
			Force:    false,
			VMID:     config.SourceVMName,
		}

		step.vmName = config.VMName

		state.Put("vm_name", step.vmName)
		state.Put("config", config)

		gomock.InOrder(
			ankaClient.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			ankaClient.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			ankaUtil.EXPECT().RandSeq(10).Return("123").Times(1),
			ankaClient.EXPECT().RegistryPush(registryParams, registryPushParams).Return(nil).Times(1),
			ankaClient.EXPECT().Clone(client.CloneParams{VMName: step.vmName, SourceUUID: sourceShowResponse.UUID}).Return(nil).Times(1),
			ankaClient.EXPECT().Show(step.vmName).Return(clonedShowResponse, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say("Preparing source VM by creating a local tag (necessary in Anka 3 to optimize disk usage of clones)")
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", sourceShowResponse.Name, step.vmName))

		stepAction := step.Run(ctx, state)

		assert.Equal(t, mockui.SayMessages[0].Message, "Preparing source VM by creating a local tag (necessary in Anka 3 to optimize disk usage of clones)")
		assert.Equal(t, mockui.SayMessages[1].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})
}
