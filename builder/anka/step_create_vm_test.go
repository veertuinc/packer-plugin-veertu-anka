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
	c "github.com/veertuinc/packer-builder-veertu-anka/client"
	mocks "github.com/veertuinc/packer-builder-veertu-anka/mocks"
	"gotest.tools/assert"
)

func TestCreateVMRun(t *testing.T) {
	// gomock implementation for testing the client
	// used for tracking and asserting expectations
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish() // will run assertions at this point for our expectations
	client := mocks.NewMockClient(mockCtrl)

	step := StepCreateVM{}
	ui := packer.TestUi(t)
	ctx := context.Background()
	state := new(multistep.BasicStateBag)

	state.Put("ui", ui)

	t.Run("create vm", func(t *testing.T) {
		var sourceShowResponse c.ShowResponse
		err := json.Unmarshal(json.RawMessage(`{ "UUID": "123456" }`), &sourceShowResponse)
		if err != nil {
			t.Fail()
		}

		config := &Config{
			SourceVMName: "source_foo",
			VMName:       "foo",
		}

		state.Put("config", config)
		state.Put("client", client)

		cloneParams := c.CloneParams{
			VMName:     config.VMName,
			SourceUUID: sourceShowResponse.UUID,
		}

		gomock.InOrder(
			client.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			client.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			client.EXPECT().Exists(config.VMName).Return(false, nil).Times(1),
			client.EXPECT().Clone(cloneParams).Return(nil).Times(1),
			client.EXPECT().Show(config.VMName).Return(c.ShowResponse{}, nil).Times(1),
			client.EXPECT().Describe(config.VMName).Return(c.DescribeResponse{}, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", config.SourceVMName, config.VMName))

		state.Put("vm_name", config.VMName)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("create vm with modify vm resources", func(t *testing.T) {
		var sourceShowResponse c.ShowResponse
		err := json.Unmarshal(json.RawMessage(`{ "UUID": "123456" }`), &sourceShowResponse)
		if err != nil {
			t.Fail()
		}

		var clonedShowResponse c.ShowResponse
		err = json.Unmarshal(json.RawMessage(`{ "Name": "foo", "CPUCores": 8, "HardDrive": 40, "RAM": "8G", "UUID": "123456" }`), &clonedShowResponse)
		if err != nil {
			t.Fail()
		}

		config := &Config{
			CPUCount:     "4",
			DiskSize:     "120G",
			RAMSize:      "16G",
			SourceVMName: "source_foo",
			VMName:       "foo",
		}

		state.Put("config", config)
		state.Put("client", client)

		cloneParams := c.CloneParams{
			VMName:     config.VMName,
			SourceUUID: sourceShowResponse.UUID,
		}

		stopParams := c.StopParams{
			VMName: clonedShowResponse.Name,
			Force:  true,
		}

		runParams := c.RunParams{
			VMName:  clonedShowResponse.Name,
			Command: []string{"diskutil", "apfs", "resizeContainer", "disk1", "0"},
		}

		gomock.InOrder(
			client.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			client.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			client.EXPECT().Exists(config.VMName).Return(false, nil).Times(1),
			client.EXPECT().Clone(cloneParams).Return(nil).Times(1),
			client.EXPECT().Show(config.VMName).Return(clonedShowResponse, nil).Times(1),
			client.EXPECT().Stop(stopParams).Return(nil).Times(1),
			client.EXPECT().Modify(clonedShowResponse.Name, "set", "hard-drive", "-s", config.DiskSize).Return(nil).Times(1),
			client.EXPECT().Run(runParams).Return(nil, 0).Times(1),
			client.EXPECT().Stop(stopParams).Return(nil).Times(2),
			client.EXPECT().Modify(clonedShowResponse.Name, "set", "ram", config.RAMSize).Return(nil).Times(1),
			client.EXPECT().Stop(stopParams).Return(nil).Times(1),
			client.EXPECT().Modify(clonedShowResponse.Name, "set", "cpu", "-c", config.CPUCount).Return(nil).Times(1),
			client.EXPECT().Describe(config.VMName).Return(c.DescribeResponse{}, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", config.SourceVMName, config.VMName))
		mockui.Say(fmt.Sprintf("Modifying VM %s disk size to %s", clonedShowResponse.Name, config.DiskSize))
		mockui.Say(fmt.Sprintf("Modifying VM %s RAM to %s", clonedShowResponse.Name, config.RAMSize))
		mockui.Say(fmt.Sprintf("Modifying VM %s CPU core count to %v", clonedShowResponse.Name, config.CPUCount))

		state.Put("vm_name", config.VMName)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, mockui.SayMessages[1].Message, "Modifying VM foo disk size to 120G")
		assert.Equal(t, mockui.SayMessages[2].Message, "Modifying VM foo RAM to 16G")
		assert.Equal(t, mockui.SayMessages[3].Message, "Modifying VM foo CPU core count to 4")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("create vm with modify vm resources but no ram changes", func(t *testing.T) {
		var sourceShowResponse c.ShowResponse
		err := json.Unmarshal(json.RawMessage(`{ "UUID": "123456" }`), &sourceShowResponse)
		if err != nil {
			t.Fail()
		}

		var clonedShowResponse c.ShowResponse
		err = json.Unmarshal(json.RawMessage(`{ "Name": "foo", "CPUCores": 8, "HardDrive": 40, "RAM": "8G", "UUID": "123456" }`), &clonedShowResponse)
		if err != nil {
			t.Fail()
		}

		config := &Config{
			CPUCount:     "4",
			DiskSize:     "120G",
			SourceVMName: "source_foo",
			VMName:       "foo",
		}

		state.Put("config", config)
		state.Put("client", client)

		cloneParams := c.CloneParams{
			VMName:     config.VMName,
			SourceUUID: sourceShowResponse.UUID,
		}

		stopParams := c.StopParams{
			VMName: clonedShowResponse.Name,
			Force:  true,
		}

		runParams := c.RunParams{
			VMName:  clonedShowResponse.Name,
			Command: []string{"diskutil", "apfs", "resizeContainer", "disk1", "0"},
		}

		gomock.InOrder(
			client.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			client.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			client.EXPECT().Exists(config.VMName).Return(false, nil).Times(1),
			client.EXPECT().Clone(cloneParams).Return(nil).Times(1),
			client.EXPECT().Show(config.VMName).Return(clonedShowResponse, nil).Times(1),
			client.EXPECT().Stop(stopParams).Return(nil).Times(1),
			client.EXPECT().Modify(clonedShowResponse.Name, "set", "hard-drive", "-s", config.DiskSize).Return(nil).Times(1),
			client.EXPECT().Run(runParams).Return(nil, 0).Times(1),
			client.EXPECT().Stop(stopParams).Return(nil).Times(2),
			client.EXPECT().Modify(clonedShowResponse.Name, "set", "cpu", "-c", config.CPUCount).Return(nil).Times(1),
			client.EXPECT().Describe(config.VMName).Return(c.DescribeResponse{}, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", config.SourceVMName, config.VMName))
		mockui.Say(fmt.Sprintf("Modifying VM %s disk size to %s", clonedShowResponse.Name, config.DiskSize))
		mockui.Say(fmt.Sprintf("Modifying VM %s CPU core count to %v", clonedShowResponse.Name, config.CPUCount))

		state.Put("vm_name", config.VMName)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, mockui.SayMessages[1].Message, "Modifying VM foo disk size to 120G")
		assert.Equal(t, mockui.SayMessages[2].Message, "Modifying VM foo CPU core count to 4")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("create vm with modify vm resources but no cpu changes", func(t *testing.T) {
		var sourceShowResponse c.ShowResponse
		err := json.Unmarshal(json.RawMessage(`{ "UUID": "123456" }`), &sourceShowResponse)
		if err != nil {
			t.Fail()
		}

		var clonedShowResponse c.ShowResponse
		err = json.Unmarshal(json.RawMessage(`{ "Name": "foo", "CPUCores": 8, "HardDrive": 40, "RAM": "8G", "UUID": "123456" }`), &clonedShowResponse)
		if err != nil {
			t.Fail()
		}

		config := &Config{
			DiskSize:     "120G",
			RAMSize:      "16G",
			SourceVMName: "source_foo",
			VMName:       "foo",
		}

		state.Put("config", config)
		state.Put("client", client)

		cloneParams := c.CloneParams{
			VMName:     config.VMName,
			SourceUUID: sourceShowResponse.UUID,
		}

		stopParams := c.StopParams{
			VMName: clonedShowResponse.Name,
			Force:  true,
		}

		runParams := c.RunParams{
			VMName:  clonedShowResponse.Name,
			Command: []string{"diskutil", "apfs", "resizeContainer", "disk1", "0"},
		}

		gomock.InOrder(
			client.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			client.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			client.EXPECT().Exists(config.VMName).Return(false, nil).Times(1),
			client.EXPECT().Clone(cloneParams).Return(nil).Times(1),
			client.EXPECT().Show(config.VMName).Return(clonedShowResponse, nil).Times(1),
			client.EXPECT().Stop(stopParams).Return(nil).Times(1),
			client.EXPECT().Modify(clonedShowResponse.Name, "set", "hard-drive", "-s", config.DiskSize).Return(nil).Times(1),
			client.EXPECT().Run(runParams).Return(nil, 0).Times(1),
			client.EXPECT().Stop(stopParams).Return(nil).Times(2),
			client.EXPECT().Modify(clonedShowResponse.Name, "set", "ram", config.RAMSize).Return(nil).Times(1),
			client.EXPECT().Describe(config.VMName).Return(c.DescribeResponse{}, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", config.SourceVMName, config.VMName))
		mockui.Say(fmt.Sprintf("Modifying VM %s disk size to %s", clonedShowResponse.Name, config.DiskSize))
		mockui.Say(fmt.Sprintf("Modifying VM %s RAM to %s", clonedShowResponse.Name, config.RAMSize))

		state.Put("vm_name", config.VMName)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, mockui.SayMessages[1].Message, "Modifying VM foo disk size to 120G")
		assert.Equal(t, mockui.SayMessages[2].Message, "Modifying VM foo RAM to 16G")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("create vm with modify vm properties", func(t *testing.T) {
		var sourceShowResponse c.ShowResponse
		err := json.Unmarshal(json.RawMessage(`{ "UUID": "123456" }`), &sourceShowResponse)
		if err != nil {
			t.Fail()
		}

		var clonedShowResponse c.ShowResponse
		err = json.Unmarshal(json.RawMessage(`{ "Name": "foo", "UUID": "123456" }`), &clonedShowResponse)
		if err != nil {
			t.Fail()
		}

		var clonedDescribeResponse c.DescribeResponse
		err = json.Unmarshal(json.RawMessage(`{  }`), &clonedDescribeResponse)
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
				"SourceVMName": "source_foo",
				"VMName": "foo",
				"HWUUID": "abcdefgh"
			}
		`), &config)
		if err != nil {
			t.Fail()
		}

		state.Put("config", &config)
		state.Put("client", client)

		cloneParams := c.CloneParams{
			VMName:     config.VMName,
			SourceUUID: sourceShowResponse.UUID,
		}

		stopParams := c.StopParams{
			VMName: clonedShowResponse.Name,
			Force:  true,
		}

		gomock.InOrder(
			client.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			client.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			client.EXPECT().Exists(config.VMName).Return(false, nil).Times(1),
			client.EXPECT().Clone(cloneParams).Return(nil).Times(1),
			client.EXPECT().Show(config.VMName).Return(clonedShowResponse, nil).Times(1),
			client.EXPECT().Describe(config.VMName).Return(c.DescribeResponse{}, nil).Times(1),
			client.EXPECT().Stop(stopParams).Return(nil).Times(1),
			client.EXPECT().Modify(clonedShowResponse.Name, "add", "port-forwarding", "--host-port", strconv.Itoa(config.PortForwardingRules[0].PortForwardingHostPort), "--guest-port", strconv.Itoa(config.PortForwardingRules[0].PortForwardingGuestPort), "rule1").Return(nil).Times(1),
			client.EXPECT().Stop(stopParams).Return(nil).Times(1),
			client.EXPECT().Modify(clonedShowResponse.Name, "set", "custom-variable", "hw.UUID", config.HWUUID).Return(nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", config.SourceVMName, config.VMName))
		mockui.Say(fmt.Sprintf("Ensuring %s port-forwarding (Guest Port: %s, Host Port: %s, Rule Name: %s)", clonedShowResponse.Name, strconv.Itoa(config.PortForwardingRules[0].PortForwardingGuestPort), strconv.Itoa(config.PortForwardingRules[0].PortForwardingHostPort), config.PortForwardingRules[0].PortForwardingRuleName))
		mockui.Say(fmt.Sprintf("Modifying VM custom-variable hw.UUID to %s", config.HWUUID))

		state.Put("vm_name", config.VMName)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, mockui.SayMessages[1].Message, "Ensuring foo port-forwarding (Guest Port: 8080, Host Port: 80, Rule Name: rule1)")
		assert.Equal(t, mockui.SayMessages[2].Message, "Modifying VM custom-variable hw.UUID to abcdefgh")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("create vm with modify vm properties with already enabled port forwarding rules", func(t *testing.T) {
		var sourceShowResponse c.ShowResponse
		err := json.Unmarshal(json.RawMessage(`{ "UUID": "123456" }`), &sourceShowResponse)
		if err != nil {
			t.Fail()
		}

		var clonedShowResponse c.ShowResponse
		err = json.Unmarshal(json.RawMessage(`{ "Name": "foo", "UUID": "123456" }`), &clonedShowResponse)
		if err != nil {
			t.Fail()
		}

		var clonedDescribeResponse c.DescribeResponse
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

		state.Put("config", &config)
		state.Put("client", client)

		cloneParams := c.CloneParams{
			VMName:     config.VMName,
			SourceUUID: sourceShowResponse.UUID,
		}

		gomock.InOrder(
			client.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			client.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			client.EXPECT().Exists(config.VMName).Return(false, nil).Times(1),
			client.EXPECT().Clone(cloneParams).Return(nil).Times(1),
			client.EXPECT().Show(config.VMName).Return(clonedShowResponse, nil).Times(1),
			client.EXPECT().Describe(config.VMName).Return(clonedDescribeResponse, nil).Times(1),
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

	t.Run("create vm with both modify resources and properties", func(t *testing.T) {
		var sourceShowResponse c.ShowResponse
		err := json.Unmarshal(json.RawMessage(`{ "UUID": "123456" }`), &sourceShowResponse)
		if err != nil {
			t.Fail()
		}

		var clonedShowResponse c.ShowResponse
		err = json.Unmarshal(json.RawMessage(`{ "Name": "foo", "CPUCores": 8, "HardDrive": 40, "RAM": "8G", "UUID": "123456" }`), &clonedShowResponse)
		if err != nil {
			t.Fail()
		}

		var clonedDescribeResponse c.DescribeResponse
		err = json.Unmarshal(json.RawMessage(`{  }`), &clonedDescribeResponse)
		if err != nil {
			t.Fail()
		}

		var config Config
		err = json.Unmarshal(json.RawMessage(`
			{
				"CPUCount": "4",
				"DiskSize": "120G",
				"RAMSize": "16G",
				"PortForwardingRules": [
					{
						"PortForwardingGuestPort": 8080,
						"PortForwardingHostPort": 80,
						"PortForwardingRuleName": "rule1"
					}
				],
				"SourceVMName": "source_foo",
				"VMName": "foo",
				"HWUUID": "abcdefgh"
			}
		`), &config)
		if err != nil {
			t.Fail()
		}

		state.Put("config", &config)
		state.Put("client", client)

		cloneParams := c.CloneParams{
			VMName:     config.VMName,
			SourceUUID: sourceShowResponse.UUID,
		}

		stopParams := c.StopParams{
			VMName: clonedShowResponse.Name,
			Force:  true,
		}

		runParams := c.RunParams{
			VMName:  clonedShowResponse.Name,
			Command: []string{"diskutil", "apfs", "resizeContainer", "disk1", "0"},
		}

		gomock.InOrder(
			client.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			client.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			client.EXPECT().Exists(config.VMName).Return(false, nil).Times(1),
			client.EXPECT().Clone(cloneParams).Return(nil).Times(1),
			client.EXPECT().Show(config.VMName).Return(clonedShowResponse, nil).Times(1),

			// modify vm resources
			client.EXPECT().Stop(stopParams).Return(nil).Times(1),
			client.EXPECT().Modify(clonedShowResponse.Name, "set", "hard-drive", "-s", config.DiskSize).Return(nil).Times(1),
			client.EXPECT().Run(runParams).Return(nil, 0).Times(1),
			client.EXPECT().Stop(stopParams).Return(nil).Times(2),
			client.EXPECT().Modify(clonedShowResponse.Name, "set", "ram", config.RAMSize).Return(nil).Times(1),
			client.EXPECT().Stop(stopParams).Return(nil).Times(1),
			client.EXPECT().Modify(clonedShowResponse.Name, "set", "cpu", "-c", config.CPUCount).Return(nil).Times(1),

			client.EXPECT().Describe(config.VMName).Return(c.DescribeResponse{}, nil).Times(1),

			// modify vm properties
			client.EXPECT().Stop(stopParams).Return(nil).Times(1),
			client.EXPECT().Modify(clonedShowResponse.Name, "add", "port-forwarding", "--host-port", strconv.Itoa(config.PortForwardingRules[0].PortForwardingHostPort), "--guest-port", strconv.Itoa(config.PortForwardingRules[0].PortForwardingGuestPort), config.PortForwardingRules[0].PortForwardingRuleName).Return(nil).Times(1),
			client.EXPECT().Stop(stopParams).Return(nil).Times(1),
			client.EXPECT().Modify(clonedShowResponse.Name, "set", "custom-variable", "hw.UUID", config.HWUUID).Return(nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", config.SourceVMName, config.VMName))
		mockui.Say(fmt.Sprintf("Modifying VM %s disk size to %s", clonedShowResponse.Name, config.DiskSize))
		mockui.Say(fmt.Sprintf("Modifying VM %s RAM to %s", clonedShowResponse.Name, config.RAMSize))
		mockui.Say(fmt.Sprintf("Modifying VM %s CPU core count to %v", clonedShowResponse.Name, config.CPUCount))
		mockui.Say(fmt.Sprintf("Ensuring %s port-forwarding (Guest Port: %s, Host Port: %s, Rule Name: %s)", clonedShowResponse.Name, strconv.Itoa(config.PortForwardingRules[0].PortForwardingGuestPort), strconv.Itoa(config.PortForwardingRules[0].PortForwardingHostPort), config.PortForwardingRules[0].PortForwardingRuleName))
		mockui.Say(fmt.Sprintf("Modifying VM custom-variable hw.UUID to %s", config.HWUUID))

		state.Put("vm_name", config.VMName)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, mockui.SayMessages[1].Message, "Modifying VM foo disk size to 120G")
		assert.Equal(t, mockui.SayMessages[2].Message, "Modifying VM foo RAM to 16G")
		assert.Equal(t, mockui.SayMessages[3].Message, "Modifying VM foo CPU core count to 4")
		assert.Equal(t, mockui.SayMessages[4].Message, "Ensuring foo port-forwarding (Guest Port: 8080, Host Port: 80, Rule Name: rule1)")
		assert.Equal(t, mockui.SayMessages[5].Message, "Modifying VM custom-variable hw.UUID to abcdefgh")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("create vm when source exists and is running", func(t *testing.T) {
		var sourceShowResponse c.ShowResponse
		err := json.Unmarshal(json.RawMessage(`{ "Status": "running", "UUID": "123456" }`), &sourceShowResponse)
		if err != nil {
			t.Fail()
		}

		config := &Config{
			SourceVMName: "source_foo",
			VMName:       "foo",
		}

		state.Put("config", config)
		state.Put("client", client)

		cloneParams := c.CloneParams{
			VMName:     config.VMName,
			SourceUUID: sourceShowResponse.UUID,
		}

		suspendParams := c.SuspendParams{
			VMName: config.SourceVMName,
		}

		gomock.InOrder(
			client.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			client.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			client.EXPECT().Suspend(suspendParams).Return(nil).Times(1),
			client.EXPECT().Exists(config.VMName).Return(false, nil).Times(1),
			client.EXPECT().Clone(cloneParams).Return(nil).Times(1),
			client.EXPECT().Show(config.VMName).Return(c.ShowResponse{}, nil).Times(1),
			client.EXPECT().Describe(config.VMName).Return(c.DescribeResponse{}, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Suspending VM %s", config.SourceVMName))
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", config.SourceVMName, config.VMName))

		state.Put("vm_name", config.VMName)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Suspending VM source_foo")
		assert.Equal(t, mockui.SayMessages[1].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("create vm when vm already exists", func(t *testing.T) {
		var sourceShowResponse c.ShowResponse
		err := json.Unmarshal(json.RawMessage(`{ "UUID": "123456" }`), &sourceShowResponse)
		if err != nil {
			t.Fail()
		}

		packerConfig := common.PackerConfig{
			PackerForce: true,
		}

		config := &Config{
			PackerConfig: packerConfig,
			SourceVMName: "source_foo",
			VMName:       "foo",
		}

		state.Put("config", config)
		state.Put("client", client)

		cloneParams := c.CloneParams{
			VMName:     config.VMName,
			SourceUUID: sourceShowResponse.UUID,
		}

		deleteParams := c.DeleteParams{
			VMName: config.VMName,
		}

		gomock.InOrder(
			client.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			client.EXPECT().Show(config.SourceVMName).Return(sourceShowResponse, nil).Times(1),
			client.EXPECT().Exists(config.VMName).Return(true, nil).Times(1),
			client.EXPECT().Delete(deleteParams).Return(nil).Times(1),
			client.EXPECT().Clone(cloneParams).Return(nil).Times(1),
			client.EXPECT().Show(config.VMName).Return(c.ShowResponse{}, nil).Times(1),
			client.EXPECT().Describe(config.VMName).Return(c.DescribeResponse{}, nil).Times(1),
		)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Suspending VM %s", config.SourceVMName))
		mockui.Say(fmt.Sprintf("Deleting existing virtual machine %s", config.VMName))
		mockui.Say(fmt.Sprintf("Cloning source VM %s into a new virtual machine: %s", config.SourceVMName, config.VMName))

		state.Put("vm_name", config.VMName)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Suspending VM source_foo")
		assert.Equal(t, mockui.SayMessages[1].Message, "Deleting existing virtual machine foo")
		assert.Equal(t, mockui.SayMessages[2].Message, "Cloning source VM source_foo into a new virtual machine: foo")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})
}
