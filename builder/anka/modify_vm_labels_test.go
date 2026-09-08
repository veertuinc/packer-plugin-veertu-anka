package anka

import (
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-plugin-veertu-anka/client"
	"github.com/veertuinc/packer-plugin-veertu-anka/mocks"
	"gotest.tools/v3/assert"
)

func TestValidateVMLabels(t *testing.T) {
	t.Run("accepts set delete rename and delete_all", func(t *testing.T) {
		err := validateVMLabels([]VMLabel{
			{Name: "env", Value: "ci"},
			{Name: "legacy", SetName: "team"},
			{Name: "old", Delete: true},
			{DeleteAll: true},
		})
		assert.NilError(t, err)
	})

	t.Run("rejects delete_all with other fields", func(t *testing.T) {
		err := validateVMLabels([]VMLabel{{DeleteAll: true, Name: "x"}})
		assert.Assert(t, err != nil)
		assert.Assert(t, strings.Contains(err.Error(), "delete_all"))
	})

	t.Run("rejects delete without name", func(t *testing.T) {
		err := validateVMLabels([]VMLabel{{Delete: true}})
		assert.Assert(t, err != nil)
	})

	t.Run("rejects rename with value", func(t *testing.T) {
		err := validateVMLabels([]VMLabel{{Name: "a", SetName: "b", Value: "c"}})
		assert.Assert(t, err != nil)
	})

	t.Run("rejects empty value on set", func(t *testing.T) {
		err := validateVMLabels([]VMLabel{{Name: "env", Value: ""}})
		assert.Assert(t, err != nil)
	})

	t.Run("rejects name only without value", func(t *testing.T) {
		err := validateVMLabels([]VMLabel{{Name: "only-name"}})
		assert.Assert(t, err != nil)
	})
}

func TestApplyVMLabels(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ankaClient := mocks.NewMockClient(ctrl)
	ui := &packer.MockUi{}
	stopParams := client.StopParams{VMName: "foo"}
	labels := []VMLabel{
		{Name: "env", Value: "ci"},
		{Name: "legacy", SetName: "team"},
		{Name: "old", Delete: true},
		{DeleteAll: true},
	}

	gomock.InOrder(
		ankaClient.EXPECT().Stop(stopParams).Return(nil).Times(1),
		ankaClient.EXPECT().Modify("foo", "label", "env", "ci").Return(nil).Times(1),
		ankaClient.EXPECT().Modify("foo", "label", "--set-name", "legacy", "team").Return(nil).Times(1),
		ankaClient.EXPECT().Modify("foo", "label", "-d", "old").Return(nil).Times(1),
		ankaClient.EXPECT().Modify("foo", "delete", "label", "-a").Return(nil).Times(1),
	)

	err := applyVMLabels(ankaClient, stopParams, "foo", labels, ui)
	assert.NilError(t, err)
	assert.Equal(t, 4, len(ui.SayMessages))
}

func TestApplyVMLabelsEmpty(t *testing.T) {
	ui := &packer.MockUi{}
	err := applyVMLabels(nil, client.StopParams{}, "foo", nil, ui)
	assert.NilError(t, err)
}
