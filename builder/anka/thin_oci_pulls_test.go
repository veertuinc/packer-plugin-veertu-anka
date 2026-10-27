package anka

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-plugin-veertu-anka/client"
	mocks "github.com/veertuinc/packer-plugin-veertu-anka/mocks"
	"gotest.tools/v3/assert"
)

const (
	thinOCIPullXcodeURL = "oci://registry.example.com/v2/veertu/tools:xcode-26.1"
	thinOCIPullCmakeURL = "oci://ghcr.io/homebrew/core/cmake:4.3.4"
)

func cloneConfigWithThinOCIPulls(thinOCIPulls ...map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"source_vm_name": "macos-base",
		"thin_oci_pulls": thinOCIPulls,
	}
}

func TestNewConfigThinOCIPulls(t *testing.T) {
	t.Run("keeps thin_oci_pulls in order with their transforms", func(t *testing.T) {
		config, err := NewConfig(cloneConfigWithThinOCIPulls(
			map[string]interface{}{"url": thinOCIPullXcodeURL},
			map[string]interface{}{"url": thinOCIPullCmakeURL, "transform": []string{"s|cmake/4.3.4|usr/local"}},
		))
		assert.NilError(t, err)
		assert.DeepEqual(t, []ThinOCIPull{
			{URL: thinOCIPullXcodeURL},
			{URL: thinOCIPullCmakeURL, Transform: []string{"s|cmake/4.3.4|usr/local"}},
		}, config.ThinOCIPulls)
	})

	t.Run("requires url", func(t *testing.T) {
		_, err := NewConfig(cloneConfigWithThinOCIPulls(map[string]interface{}{"transform": []string{"s|a|b"}}))
		assert.ErrorContains(t, err, "thin_oci_pulls[0]: url is required")
	})

	t.Run("requires an OCI url", func(t *testing.T) {
		_, err := NewConfig(cloneConfigWithThinOCIPulls(
			map[string]interface{}{"url": thinOCIPullXcodeURL},
			map[string]interface{}{"url": "http://anka.registry:8089"},
		))
		assert.ErrorContains(t, err, "thin_oci_pulls[1]: url \"http://anka.registry:8089\" must be an oci:// URL")
	})

	t.Run("rejects an empty transform", func(t *testing.T) {
		_, err := NewConfig(cloneConfigWithThinOCIPulls(map[string]interface{}{"url": thinOCIPullXcodeURL, "transform": []string{" "}}))
		assert.ErrorContains(t, err, "thin_oci_pulls[0]: transform entries must not be empty")
	})

	t.Run("requires source_vm_name", func(t *testing.T) {
		_, err := NewConfig(map[string]interface{}{
			"installer":      "/Applications/Install macOS.app",
			"thin_oci_pulls": []map[string]interface{}{{"url": thinOCIPullXcodeURL}},
		})
		assert.ErrorContains(t, err, "thin_oci_pulls requires source_vm_name")
	})

	t.Run("rejects TLS options with an OCI remote", func(t *testing.T) {
		_, err := NewConfig(map[string]interface{}{
			"source_vm_name": "macos-base",
			"remote":         "oci://registry.example.com/v2/veertu",
			"insecure":       true,
		})
		assert.ErrorContains(t, err, "only for Anka Registry remotes")
	})
}

func TestCloneVMRunWithOCI(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ankaClient := mocks.NewMockClient(mockCtrl)
	ankaUtil := mocks.NewMockUtil(mockCtrl)

	ctx := context.Background()
	sourceShow := client.ShowResponse{Name: "macos-base", UUID: "source-uuid", Version: "vanilla"}
	clonedShow := client.ShowResponse{Name: "foo", UUID: "cloned-uuid"}

	newCloneState := func(config *Config, ui packer.Ui) *multistep.BasicStateBag {
		state := new(multistep.BasicStateBag)
		state.Put("ui", ui)
		state.Put("client", ankaClient)
		state.Put("util", ankaUtil)
		state.Put("config", config)
		return state
	}

	newCloneConfig := func() *Config {
		return &Config{
			VMName:       "foo",
			SourceVMName: "macos-base",
			PackerConfig: common.PackerConfig{PackerBuilderType: "veertu-anka-vm-clone"},
			ThinOCIPulls: []ThinOCIPull{
				{URL: thinOCIPullXcodeURL},
				{URL: thinOCIPullCmakeURL, Transform: []string{"s|cmake/4.3.4|usr/local", "s|x|y"}},
			},
		}
	}

	t.Run("applies thin_oci_pulls in order onto the source VM before clone", func(t *testing.T) {
		config := newCloneConfig()
		mockui := &packer.MockUi{}
		state := newCloneState(config, mockui)

		gomock.InOrder(
			ankaClient.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			ankaClient.EXPECT().Show(config.SourceVMName).Return(sourceShow, nil).Times(1),
			ankaClient.EXPECT().OCIPull(client.OCIPullParams{
				VMID: config.SourceVMName, Remote: thinOCIPullXcodeURL, Thin: true,
			}, gomock.Any()).Return(nil).Times(1),
			ankaClient.EXPECT().OCIPull(client.OCIPullParams{
				VMID: config.SourceVMName, Remote: thinOCIPullCmakeURL, Thin: true,
				Transforms: []string{"s|cmake/4.3.4|usr/local", "s|x|y"},
			}, gomock.Any()).Return(nil).Times(1),
			ankaClient.EXPECT().Clone(client.CloneParams{VMName: "foo", SourceUUID: sourceShow.UUID}).Return(nil).Times(1),
			ankaClient.EXPECT().Show("foo").Return(clonedShow, nil).Times(1),
		)

		stepAction := (&StepCloneVM{}).Run(ctx, state)

		assert.Equal(t, multistep.ActionContinue, stepAction)
		assert.Assert(t, mockUiSaid(mockui, "Pulling thin OCI tag "+thinOCIPullXcodeURL+" onto source VM macos-base"))
		assert.Assert(t, mockUiSaid(mockui, "Pulling thin OCI tag "+thinOCIPullCmakeURL+" onto source VM macos-base"))
	})

	t.Run("pulls a full OCI base, then thin_oci_pulls", func(t *testing.T) {
		config := newCloneConfig()
		config.ThinOCIPulls = config.ThinOCIPulls[:1]
		config.Remote = "oci://registry.example.com/v2/veertu/"
		config.SourceVMTag = "vanilla"
		config.AlwaysFetch = true
		mockui := &packer.MockUi{}
		state := newCloneState(config, mockui)

		gomock.InOrder(
			ankaClient.EXPECT().OCIPull(client.OCIPullParams{
				VMID: config.SourceVMName, Remote: config.Remote, Tag: "vanilla",
			}, gomock.Any()).Return(nil).Times(1),
			ankaClient.EXPECT().Show(config.SourceVMName).Return(sourceShow, nil).Times(1),
			ankaClient.EXPECT().OCIPull(client.OCIPullParams{
				VMID: config.SourceVMName, Remote: thinOCIPullXcodeURL, Thin: true,
			}, gomock.Any()).Return(nil).Times(1),
			ankaClient.EXPECT().Clone(client.CloneParams{VMName: "foo", SourceUUID: sourceShow.UUID}).Return(nil).Times(1),
			ankaClient.EXPECT().Show("foo").Return(clonedShow, nil).Times(1),
		)

		stepAction := (&StepCloneVM{}).Run(ctx, state)

		assert.Equal(t, multistep.ActionContinue, stepAction)
		assert.Assert(t, mockUiSaid(mockui, "Pulling source VM macos-base with vanilla tag from OCI registry oci://registry.example.com/v2/veertu/"))
	})

	t.Run("a failed thin pull stops the build before clone", func(t *testing.T) {
		config := newCloneConfig()
		mockui := &packer.MockUi{}
		state := newCloneState(config, mockui)
		thinPullErr := errors.New("layer download failed")

		gomock.InOrder(
			ankaClient.EXPECT().Exists(config.SourceVMName).Return(true, nil).Times(1),
			ankaClient.EXPECT().Show(config.SourceVMName).Return(sourceShow, nil).Times(1),
			ankaClient.EXPECT().OCIPull(gomock.Any(), gomock.Any()).Return(thinPullErr).Times(1),
			ankaUtil.EXPECT().StepError(mockui, state, gomock.Any()).
				DoAndReturn(func(_ packer.Ui, _ multistep.StateBag, err error) multistep.StepAction {
					assert.ErrorContains(t, err, "failed to pull thin OCI tag "+thinOCIPullXcodeURL+" onto source VM macos-base: layer download failed")
					return multistep.ActionHalt
				}).Times(1),
		)

		stepAction := (&StepCloneVM{}).Run(ctx, state)

		assert.Equal(t, multistep.ActionHalt, stepAction)
	})
}

func mockUiSaid(mockui *packer.MockUi, message string) bool {
	for _, sayMessage := range mockui.SayMessages {
		if sayMessage.Message == message {
			return true
		}
	}
	return false
}
