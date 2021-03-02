package anka

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	c "github.com/veertuinc/packer-builder-veertu-anka/client"
	mocks "github.com/veertuinc/packer-builder-veertu-anka/mocks"
	"gotest.tools/assert"
)

func TestStartVMRun(t *testing.T) {
	// gomock implementation for testing the client
	// used for tracking and asserting expectations
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish() // will run assertions at this point for our expectations
	client := mocks.NewMockClient(mockCtrl)

	step := StepStartVM{}
	ui := packer.TestUi(t)
	ctx := context.Background()
	state := new(multistep.BasicStateBag)

	state.Put("ui", ui)
	state.Put("vm_name", "foo")

	t.Run("start vm", func(t *testing.T) {
		config := &Config{}

		state.Put("client", client)
		state.Put("config", config)

		client.EXPECT().Start(c.StartParams{VMName: "foo"}).Return(nil).Times(1)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, stepAction, multistep.ActionContinue)
	})

	t.Run("start vm with boot delay", func(t *testing.T) {
		config := &Config{
			BootDelay: "1s",
		}

		d, err := time.ParseDuration(config.BootDelay)
		if err != nil {
			t.Fail()
		}

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Waiting for %s for clone to boot", d))

		state.Put("client", client)
		state.Put("config", config)

		client.EXPECT().Start(c.StartParams{VMName: "foo"}).Return(nil).Times(1)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, mockui.SayMessages[0].Message, "Waiting for 1s for clone to boot")
		assert.Equal(t, stepAction, multistep.ActionContinue)
	})
}
