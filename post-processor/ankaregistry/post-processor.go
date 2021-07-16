//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package ankaregistry

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/veertuinc/packer-plugin-veertu-anka/builder/anka"
	"github.com/veertuinc/packer-plugin-veertu-anka/client"
)

// BuilderIdRegistry unique id for this post processor
const BuilderIdRegistry = "packer.post-processor.veertu-anka-registry"

// Config initializes the post processor using mapstructure which decodes
// generic map values from either the json or hcl2 config files provided
type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	RegistryName string `mapstructure:"registry_name"`
	RegistryURL  string `mapstructure:"registry_path"`
	NodeCertPath string `mapstructure:"cert"`
	NodeKeyPath  string `mapstructure:"key"`
	CaRootPath   string `mapstructure:"cacert"`
	IsInsecure   bool   `mapstructure:"insecure"`

	Tag         string `mapstructure:"tag"`
	Description string `mapstructure:"description"`
	RemoteVM    string `mapstructure:"remote_vm"`
	Local       bool   `mapstructure:"local"`

	ctx interpolate.Context
}

// PostProcessor is used to run the post processor
type PostProcessor struct {
	config Config
	client client.Client
}

// ConfigSpec returns the decoded mapstructure config values into their respective format
func (p *PostProcessor) ConfigSpec() hcldec.ObjectSpec { return p.config.FlatMapstructure().HCL2Spec() }

// Configure sets up the post processor with the decoded config values
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

// PostProcess runs the post processor logic which uploads the artifact to an anka registry
func (p *PostProcessor) PostProcess(ctx context.Context, ui packer.Ui, artifact packer.Artifact) (packer.Artifact, bool, bool, error) {
	if artifact.BuilderId() != anka.BuilderId {
		err := fmt.Errorf(
			"unknown artifact type: %s\ncan only import from anka artifacts",
			artifact.BuilderId())
		return nil, false, false, err
	}

	if p.config.RegistryURL == "" {
		reposList, err := p.client.RegistryListRepos()
		if err != nil {
			return nil, false, false, err
		}

		if p.config.RegistryName == "" {
			p.config.RegistryName = reposList.Default
		}

		remote, ok := reposList.Remotes[p.config.RegistryName]
		if !ok {
			return nil, false, false, fmt.Errorf("Could not find configuration for registry '%s'", p.config.RegistryName)
		}

		p.config.RegistryURL = fmt.Sprintf("%s://%s:%s", remote.Scheme, remote.Host, remote.Port)
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
		RemoteVM:    p.config.RemoteVM,
		Local:       p.config.Local,
		VMID:        artifact.String(),
	}

	if p.config.PackerForce {
		var id string
		var latestTag string

		templates, err := p.client.RegistryList(registryParams)
		if err != nil {
			return nil, false, false, err
		}

		for i := 0; i < len(templates); i++ {
			if templates[i].Name == remoteVMName {
				id = templates[i].ID
				latestTag = templates[i].Latest
				ui.Say(fmt.Sprintf("Found existing template %s on registry that matches name '%s'", id, remoteVMName))
				break
			}
		}

		if id != "" && latestTag == remoteTag {
			err = p.client.RegistryRevert(registryParams.RegistryURL, id)
			if err != nil {
				return nil, false, false, err
			}

			ui.Say(fmt.Sprintf("Reverted latest tag for template '%s' on registry", id))
		}
	}

	ui.Say(fmt.Sprintf("Pushing template to Anka Registry as %s with tag %s", remoteVMName, remoteTag))
	pushErr := p.client.RegistryPush(registryParams, pushParams)

	return artifact, true, false, pushErr
}
