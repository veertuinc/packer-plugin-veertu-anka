package anka

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	c "github.com/veertuinc/packer-builder-veertu-anka/client"
	mocks "github.com/veertuinc/packer-builder-veertu-anka/mocks"
	"gotest.tools/assert"
)

func TestStartVMRun(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	client := mocks.NewMockClient(mockCtrl)
	util := mocks.NewMockUtil(mockCtrl)

	step := StepStartVM{}
	ui := packer.TestUi(t)
	ctx := context.Background()
	state := new(multistep.BasicStateBag)

	state.Put("ui", ui)
	state.Put("util", util)
	state.Put("vm_name", "foo")

	t.Run("start vm", func(t *testing.T) {
		config := &Config{}

		state.Put("client", client)
		state.Put("config", config)

		client.EXPECT().Start(c.StartParams{VMName: "foo"}).Return(nil).Times(1)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("start vm with boot delay", func(t *testing.T) {
		config := &Config{
			BootDelay: "1s",
		}

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Waiting for %s for clone to boot", config.BootDelay))

		state.Put("client", client)
		state.Put("config", config)

		client.EXPECT().Start(c.StartParams{VMName: "foo"}).Return(nil).Times(1)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Waiting for 1s for clone to boot")
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})

	t.Run("start vm but fail to start", func(t *testing.T) {
		config := &Config{}

		state.Put("client", client)
		state.Put("config", config)

		gomock.InOrder(
			client.EXPECT().Start(c.StartParams{VMName: "foo"}).Return(fmt.Errorf("failed to start vm %s", "foo")).Times(1),
			util.EXPECT().StepError(ui, state, fmt.Errorf("failed to start vm %s", "foo")).Return(multistep.ActionHalt).Times(1),
		)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, multistep.ActionHalt, stepAction)
	})
}
