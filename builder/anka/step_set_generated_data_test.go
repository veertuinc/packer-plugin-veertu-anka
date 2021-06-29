package anka

import (
	"bytes"
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
	mocks "github.com/veertuinc/packer-builder-veertu-anka/mocks"
	"gotest.tools/assert"
)

func TestSetGeneratedDataRun(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ankaClient := mocks.NewMockClient(mockCtrl)
	ankaUtil := mocks.NewMockUtil(mockCtrl)

	state := new(multistep.BasicStateBag)
	step := StepSetGeneratedData{
		vmName:        "foo-11.2-16.4.06",
		GeneratedData: &packerbuilderdata.GeneratedData{State: state},
	}
	ui := packer.TestUi(t)
	ctx := context.Background()
	darwinVersion := client.RunParams{
		Command: []string{"/usr/bin/uname", "-r"},
		VMName:  step.vmName,
		Stdout:  &bytes.Buffer{},
	}
	osv := client.RunParams{
		Command: []string{"/usr/bin/sw_vers", "-productVersion"},
		VMName:  step.vmName,
		Stdout:  &bytes.Buffer{},
	}

	state.Put("ui", ui)
	state.Put("client", ankaClient)
	state.Put("util", ankaUtil)

	t.Run("expose variables", func(t *testing.T) {
		state.Put("vm_name", step.vmName)

		gomock.InOrder(
			ankaClient.EXPECT().Run(darwinVersion).Times(1),
			ankaClient.EXPECT().Run(osv).Times(1),
		)

		stepAction := step.Run(ctx, state)
		assert.Equal(t, multistep.ActionContinue, stepAction)
	})
}
