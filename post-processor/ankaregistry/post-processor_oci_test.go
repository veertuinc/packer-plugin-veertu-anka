package ankaregistry

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-plugin-veertu-anka/builder/anka"
	"github.com/veertuinc/packer-plugin-veertu-anka/client"
	mocks "github.com/veertuinc/packer-plugin-veertu-anka/mocks"
	"gotest.tools/v3/assert"
)

const ociRegistryRemote = "oci://registry.example.com/v2/veertu"

func TestConfigureOCIRegistryPush(t *testing.T) {
	configureWith := func(rawConfig map[string]interface{}) error {
		postProcessor := PostProcessor{}
		return postProcessor.Configure(rawConfig)
	}

	t.Run("thin push to an OCI remote is valid", func(t *testing.T) {
		assert.NilError(t, configureWith(map[string]interface{}{"tag": "xcode-26.1", "remote": ociRegistryRemote, "thin": true}))
	})

	t.Run("thin requires an OCI remote", func(t *testing.T) {
		for _, remote := range []string{"", "my-anka-registry", "http://anka.registry:8089"} {
			err := configureWith(map[string]interface{}{"tag": "xcode-26.1", "remote": remote, "thin": true})
			assert.ErrorContains(t, err, "'thin' requires 'remote' to be an OCI registry URL")
		}
	})

	t.Run("thin and local are mutually exclusive", func(t *testing.T) {
		err := configureWith(map[string]interface{}{"tag": "xcode-26.1", "remote": ociRegistryRemote, "thin": true, "local": true})
		assert.ErrorContains(t, err, "the 'local' and 'thin' settings are mutually exclusive")
	})

	t.Run("TLS options are rejected for an OCI remote", func(t *testing.T) {
		err := configureWith(map[string]interface{}{"tag": "xcode-26.1", "remote": ociRegistryRemote, "cacert": "/ca.pem"})
		assert.ErrorContains(t, err, "only for Anka Registry remotes")
	})
}

func TestOCIRegistryPostProcessor(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ankaClient := mocks.NewMockClient(mockCtrl)

	localArtifact := anka.NewVMTemplateArtifact("built-vm", "uuid-built")
	showResponse := client.ShowResponse{Name: "built-vm", UUID: "uuid-built"}

	postProcessorWith := func(config Config) PostProcessor {
		return PostProcessor{config: config, client: ankaClient}
	}

	// The mock fails on any unexpected call, so these tests also prove that the OCI path never calls
	// RegistryListRepos, RegistryList, RegistryRevert, or RegistryPush.
	t.Run("thin push uses anka push --thin", func(t *testing.T) {
		postProcessor := postProcessorWith(Config{
			Remote:      ociRegistryRemote,
			Tag:         "xcode-26.1",
			Description: "Xcode 26.1",
			RemoteVM:    "tools",
			Thin:        true,
		})
		gomock.InOrder(
			ankaClient.EXPECT().Show("built-vm").Return(showResponse, nil).Times(1),
			ankaClient.EXPECT().OCIPush(client.OCIPushParams{
				VMID:        "built-vm",
				Remote:      ociRegistryRemote,
				Tag:         "xcode-26.1",
				Description: "Xcode 26.1",
				RemoteVM:    "tools",
				Thin:        true,
			}, gomock.Any()).Return(nil).Times(1),
		)

		mockui := &packer.MockUi{}
		_, keep, forceOverride, err := postProcessor.PostProcess(context.Background(), mockui, localArtifact)

		assert.NilError(t, err)
		assert.Equal(t, true, keep)
		assert.Equal(t, false, forceOverride)
		assert.Assert(t, mockUiSaid(mockui, client.FormatShowResponse("pre-push", showResponse)))
		assert.Assert(t, mockUiSaid(mockui, "Pushing template built-vm to OCI registry "+ociRegistryRemote+" as thin OCI tag xcode-26.1"))
		assert.Assert(t, mockUiSaid(mockui, "Registry push successful"))
	})

	t.Run("full push with packer -force passes --force", func(t *testing.T) {
		postProcessor := postProcessorWith(Config{
			PackerConfig: common.PackerConfig{PackerForce: true},
			Remote:       ociRegistryRemote,
			Tag:          "vanilla",
		})
		gomock.InOrder(
			ankaClient.EXPECT().Show("built-vm").Return(showResponse, nil).Times(1),
			ankaClient.EXPECT().OCIPush(client.OCIPushParams{
				VMID:   "built-vm",
				Remote: ociRegistryRemote,
				Tag:    "vanilla",
				Force:  true,
			}, gomock.Any()).Return(nil).Times(1),
		)

		mockui := &packer.MockUi{}
		_, _, _, err := postProcessor.PostProcess(context.Background(), mockui, localArtifact)

		assert.NilError(t, err)
		assert.Assert(t, mockUiSaid(mockui, "Pushing template built-vm to OCI registry "+ociRegistryRemote+" as full OCI tag vanilla"))
	})

	t.Run("delete_template_post_push deletes the VM after an OCI push", func(t *testing.T) {
		postProcessor := postProcessorWith(Config{
			Remote:                 ociRegistryRemote,
			Tag:                    "xcode-26.1",
			Thin:                   true,
			DeleteTemplatePostPush: true,
		})
		gomock.InOrder(
			ankaClient.EXPECT().Show("built-vm").Return(showResponse, nil).Times(1),
			ankaClient.EXPECT().OCIPush(gomock.Any(), gomock.Any()).Return(nil).Times(1),
			ankaClient.EXPECT().Delete(client.DeleteParams{VMName: "built-vm"}).Return(nil).Times(1),
		)

		_, _, _, err := postProcessor.PostProcess(context.Background(), &packer.MockUi{}, localArtifact)

		assert.NilError(t, err)
	})

	t.Run("a failed OCI push returns the error and keeps the VM", func(t *testing.T) {
		postProcessor := postProcessorWith(Config{
			Remote:                 ociRegistryRemote,
			Tag:                    "xcode-26.1",
			Thin:                   true,
			DeleteTemplatePostPush: true,
		})
		gomock.InOrder(
			ankaClient.EXPECT().Show("built-vm").Return(showResponse, nil).Times(1),
			ankaClient.EXPECT().OCIPush(gomock.Any(), gomock.Any()).Return(client.MachineReadableError{
				MachineReadableOutput: &client.MachineReadableOutput{Status: "ERROR", Message: "upload failed"},
			}).Times(1),
		)

		_, _, _, err := postProcessor.PostProcess(context.Background(), &packer.MockUi{}, localArtifact)

		assert.ErrorContains(t, err, "upload failed")
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
