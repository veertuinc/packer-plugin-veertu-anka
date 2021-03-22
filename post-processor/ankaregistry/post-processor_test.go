package ankaregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/alecthomas/assert"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-builder-veertu-anka/builder/anka"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
	mocks "github.com/veertuinc/packer-builder-veertu-anka/mocks"
)

var templateList []client.RegistryListResponse

func TestAnkaRegistryPostProcessor(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ankaClient := mocks.NewMockClient(mockCtrl)

	ui := packer.TestUi(t)

	artifact := &anka.Artifact{}

	t.Run("should push to registry", func(t *testing.T) {
		config := Config{
			RegistryName: "go-mock-anka-registry",
			RegistryURL:  "mockurl:mockport",
			RemoteVM:     "foo",
			Tag:          "registry-push",
			Description:  "mock for testing anka registry push",
		}

		pp := PostProcessor{
			config: config,
			client: ankaClient,
		}

		registryParams := client.RegistryParams{
			RegistryName: config.RegistryName,
			RegistryURL:  config.RegistryURL,
		}

		pushParams := client.RegistryPushParams{
			Tag:         config.Tag,
			Description: config.Description,
			RemoteVM:    config.RemoteVM,
			Local:       false,
		}

		ankaClient.EXPECT().RegistryPush(registryParams, pushParams).Return(nil).Times(1)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Pushing template to Anka Registry as %s with tag %s", config.RemoteVM, config.Tag))

		assert.Equal(t, mockui.SayMessages[0].Message, "Pushing template to Anka Registry as foo with tag registry-push")

		pp.PostProcess(context.Background(), ui, artifact)
	})

	t.Run("with force push to registry but no existing templates", func(t *testing.T) {
		packerConfig := common.PackerConfig{
			PackerForce: true,
		}

		config := Config{
			PackerConfig: packerConfig,
			RegistryName: "go-mock-anka-registry",
			RegistryURL:  "mockurl:mockport",
			RemoteVM:     "foo",
			Tag:          "registry-push",
			Description:  "mock for testing anka registry push",
		}

		pp := PostProcessor{
			config: config,
			client: ankaClient,
		}

		registryParams := client.RegistryParams{
			RegistryName: config.RegistryName,
			RegistryURL:  config.RegistryURL,
		}

		pushParams := client.RegistryPushParams{
			Tag:         config.Tag,
			Description: config.Description,
			RemoteVM:    config.RemoteVM,
			Local:       false,
		}

		ankaClient.EXPECT().RegistryList(registryParams).Return([]client.RegistryListResponse{}, nil).Times(1)
		ankaClient.EXPECT().RegistryPush(registryParams, pushParams).Return(nil).Times(1)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Pushing template to Anka Registry as %s with tag %s", config.RemoteVM, config.Tag))

		assert.Equal(t, mockui.SayMessages[0].Message, "Pushing template to Anka Registry as foo with tag registry-push")

		pp.PostProcess(context.Background(), ui, artifact)
	})

	t.Run("with force push to registry and existing templates", func(t *testing.T) {
		err := json.Unmarshal(json.RawMessage(`[{ "id": "foo_id", "name": "foo" }]`), &templateList)
		if err != nil {
			t.Fail()
		}

		packerConfig := common.PackerConfig{
			PackerForce: true,
		}

		config := Config{
			PackerConfig: packerConfig,
			RegistryName: "go-mock-anka-registry",
			RegistryURL:  "mockurl:mockport",
			RemoteVM:     "foo",
			Tag:          "registry-push",
			Description:  "mock for testing anka registry push",
		}

		pp := PostProcessor{
			config: config,
			client: ankaClient,
		}

		registryParams := client.RegistryParams{
			RegistryName: config.RegistryName,
			RegistryURL:  config.RegistryURL,
		}

		pushParams := client.RegistryPushParams{
			Tag:         config.Tag,
			Description: config.Description,
			RemoteVM:    config.RemoteVM,
			Local:       false,
		}

		ankaClient.EXPECT().RegistryList(registryParams).Return(templateList, nil).Times(1)
		ankaClient.EXPECT().RegistryRevert(registryParams.RegistryURL, templateList[0].ID).Return(nil).Times(1)
		ankaClient.EXPECT().RegistryPush(registryParams, pushParams).Return(nil).Times(1)

		mockui := packer.MockUi{}
		mockui.Say(fmt.Sprintf("Pushing template to Anka Registry as %s with tag %s", config.RemoteVM, config.Tag))
		mockui.Say(fmt.Sprintf("Found existing template %s on registry that matches name '%s'", templateList[0].ID, config.RemoteVM))
		mockui.Say(fmt.Sprintf("Reverted latest tag for template '%s' on registry", templateList[0].ID))

		assert.Equal(t, mockui.SayMessages[0].Message, "Pushing template to Anka Registry as foo with tag registry-push")
		assert.Equal(t, mockui.SayMessages[1].Message, "Found existing template foo_id on registry that matches name 'foo'")
		assert.Equal(t, mockui.SayMessages[2].Message, "Reverted latest tag for template 'foo_id' on registry")

		pp.PostProcess(context.Background(), ui, artifact)
	})
}
