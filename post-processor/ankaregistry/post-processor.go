//go:generate mapstructure-to-hcl2 -type Config

package ankaregistry

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/veertuinc/packer-builder-veertu-anka/builder/anka"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
)

const BuilderIdRegistry = "packer.post-processor.veertu-anka-registry"

type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	RegistryName string `mapstructure:"remote"`
	RegistryURL  string `mapstructure:"registry-path"`
	NodeCertPath string `mapstructure:"cert"`
	NodeKeyPath  string `mapstructure:"key"`
	CaRootPath   string `mapstructure:"cacert"`
	IsInsecure   bool   `mapstructure:"insecure"`

	Tag         string `mapstructure:"tag"`
	Description string `mapstructure:"description"`
	RemoteVM    string `mapstructure:"remote-vm"`
	Local       bool   `mapstructure:"local"`

	ctx interpolate.Context
}

type PostProcessor struct {
	config Config
	client client.Client
}

func (p *PostProcessor) ConfigSpec() hcldec.ObjectSpec { return p.config.FlatMapstructure().HCL2Spec() }

func (p *PostProcessor) Configure(raws ...interface{}) error {
	err := config.Decode(&p.config, &config.DecodeOpts{
		PluginType:         BuilderIdRegistry,
		Interpolate:        true,
		InterpolateContext: &p.config.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{},
		},
	}, raws...)
	if err != nil {
		return err
	}

	if p.config.Tag == "" {
		return fmt.Errorf("You must specify a valid tag for your Veertu Anka VM (e.g. 'latest')")
	}

	p.client = &client.AnkaClient{}

	return nil
}

func (p *PostProcessor) PostProcess(ctx context.Context, ui packer.Ui, artifact packer.Artifact) (packer.Artifact, bool, bool, error) {
	if artifact.BuilderId() != anka.BuilderId {
		err := fmt.Errorf(
			"unknown artifact type: %s\ncan only import from anka artifacts",
			artifact.BuilderId())
		return nil, false, false, err
	}

	registryParams := client.RegistryParams{
		RegistryName: p.config.RegistryName,
		RegistryURL:  p.config.RegistryURL,
		NodeCertPath: p.config.NodeCertPath,
		NodeKeyPath:  p.config.NodeKeyPath,
		CaRootPath:   p.config.CaRootPath,
		IsInsecure:   p.config.IsInsecure,
	}

	remoteVMName := artifact.String()
	if p.config.RemoteVM != "" {
		remoteVMName = p.config.RemoteVM
	}

	remoteTag := "latest"
	if p.config.Tag != "" {
		remoteTag = p.config.Tag
	}

	pushParams := client.RegistryPushParams{
		Tag:         remoteTag,
		Description: p.config.Description,
		RemoteVM:    remoteVMName,
		Local:       p.config.Local,
		VMID:        artifact.Id(),
	}

	// If force is true, revert the template tag (if one exists) on the registry so we can push the VM without issue
	if p.config.PackerForce {
		var id string

		templates, err := p.client.RegistryList(registryParams)
		if err != nil {
			return nil, false, false, err
		}

		for i := 0; i < len(templates); i++ {
			if templates[i].Name == remoteVMName {
				id = templates[i].ID
				ui.Say(fmt.Sprintf("Found existing template %s on registry that matches name '%s'", id, remoteVMName))
				break
			}
		}

		if id != "" {
			if err := p.client.RegistryRevert(registryParams.RegistryURL, id); err != nil {
				return nil, false, false, err
			}
			ui.Say(fmt.Sprintf("Reverted latest tag for template '%s' on registry", id))
		}
	}

	ui.Say(fmt.Sprintf("Pushing template to Anka Registry as %s with tag %s", remoteVMName, remoteTag))
	pushErr := p.client.RegistryPush(registryParams, pushParams)

	return artifact, true, false, pushErr
}
