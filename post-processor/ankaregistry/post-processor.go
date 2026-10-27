//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package ankaregistry

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"runtime"

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

	Remote       string `mapstructure:"remote"`
	NodeCertPath string `mapstructure:"cert"`
	NodeKeyPath  string `mapstructure:"key"`
	CaRootPath   string `mapstructure:"cacert"`
	IsInsecure   bool   `mapstructure:"insecure"`

	Tag         string `mapstructure:"tag"`
	Description string `mapstructure:"description"`
	RemoteVM    string `mapstructure:"remote_vm"`
	Local       bool   `mapstructure:"local"`
	Force       bool   `mapstructure:"force"`
	// Thin pushes only the build's own layer as a thin OCI tag (`anka push --thin`). It requires an OCI registry URL in remote.
	Thin bool `mapstructure:"thin"`

	// DeleteTemplatePostPush runs `anka delete --yes` on the builder's local VM only after a successful remote registry push (not when local = true). See issue #142.
	DeleteTemplatePostPush bool `mapstructure:"delete_template_post_push"`

	HostArch string `mapstructure:"host_arch,omitempty"`

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

	var errs *packer.MultiError

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
		errs = packer.MultiErrorAppend(errs, errors.New("you must specify a valid tag for your Veertu Anka VM (e.g. 'latest')"))
	}

	if p.config.Local && p.config.RemoteVM != "" {
		errs = packer.MultiErrorAppend(errs, errors.New("the 'local' and 'remote_vm' settings are mutually exclusive"))
	}

	if p.config.Thin && p.config.Local {
		errs = packer.MultiErrorAppend(errs, errors.New("the 'local' and 'thin' settings are mutually exclusive"))
	}

	if p.config.Thin && !client.IsOCIRemote(p.config.Remote) {
		errs = packer.MultiErrorAppend(errs, errors.New("'thin' requires 'remote' to be an OCI registry URL (oci://... or a URL with /v2 in its path); the Anka Registry stores only full tags"))
	}

	p.config.HostArch = runtime.GOARCH

	if err := client.ValidateOCIRemoteTLSOptions(p.registryParams()); err != nil {
		errs = packer.MultiErrorAppend(errs, err)
	}

	p.client = &client.AnkaClient{}

	if errs != nil && len(errs.Errors) > 0 {
		return errs
	}

	return nil
}

// PostProcess runs the post processor logic which uploads the artifact to an anka registry
func (p *PostProcessor) PostProcess(ctx context.Context, ui packer.Ui, artifact packer.Artifact) (packer.Artifact, bool, bool, error) {
	var reposList []client.RegistryRemote
	var err error
	if artifact.BuilderId() != anka.BuilderId {
		err := fmt.Errorf(
			"unknown artifact type: %s\ncan only import from anka artifacts",
			artifact.BuilderId())
		return nil, false, false, err
	}

	if !p.config.Local && client.IsOCIRemote(p.config.Remote) {
		return p.pushToOCIRegistry(ui, artifact)
	}

	reposList, err = p.client.RegistryListRepos()
	if err != nil {
		return nil, false, false, err
	}

	if p.config.Remote == "" { // no remote set by user? use the default on the host
		for _, remote := range reposList {
			if remote.Default {
				p.config.Remote = remote.Name
			}
		}
	} else {
		_, err := url.ParseRequestURI(p.config.Remote)
		if err != nil { // not a url, so we should treat it as a string/name for the local registry
			foundRemoteName := false
			for _, repoRemote := range reposList {
				if repoRemote.Name == p.config.Remote {
					foundRemoteName = true
				}
			}
			if !foundRemoteName {
				return nil, false, false, fmt.Errorf("could not find configuration for registry remote name '%s'", p.config.Remote)
			}
		}
	}

	registryParams := p.registryParams()

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
		Force:       p.config.Force,
		VMID:        artifact.String(),
	}

	if showErr := p.logPrePushShow(ui, artifact); showErr != nil {
		return artifact, true, false, showErr
	}

	var id string
	var latestTag string
	var found bool
	var foundMessage string

	if p.config.Local {
		ui.Say(fmt.Sprintf("Tagging local template %s with tag %s", remoteVMName, remoteTag))
	} else {
		ui.Say(fmt.Sprintf("Pushing template to Anka Registry as %s with tag %s", remoteVMName, remoteTag))

		// Check if it already exists first
		templates, err := p.client.RegistryList(registryParams)
		if err != nil {
			return nil, false, false, err
		}

		for i := 0; i < len(templates); i++ {
			if templates[i].Name == remoteVMName {
				id = templates[i].ID
				latestTag = templates[i].Latest
				if !pushParams.Force { // avoid revert and error if we're forcing the push with the CLI
					found = true
				}
				foundMessage = fmt.Sprintf("Found existing template %s on registry that matches name '%s'", id, remoteVMName)
				ui.Say(foundMessage)
				break
			}
		}

		if p.config.PackerForce { // differs from processor's force: true
			if id != "" && latestTag == remoteTag {
				err = p.client.RegistryRevert(registryParams.Remote, id)
				if err != nil {
					return nil, false, false, err
				}
				ui.Say(fmt.Sprintf("Reverted latest tag for template '%s' on registry", id))
			}
			found = false
		}

	}

	if found {
		return artifact, true, false, errors.New(foundMessage)
	}

	pushErr := client.RunWithOutputStreamToUI(ui, func(outputStream chan string) error {
		return p.client.RegistryPush(registryParams, pushParams, outputStream)
	})
	if pushErr != nil {
		return artifact, true, false, pushErr
	}

	return p.finishSuccessfulPush(ui, artifact)
}

// pushToOCIRegistry uses `anka push`, which handles OCI tags. The Anka Registry list and revert API does not exist on an OCI registry,
// so an existing tag is replaced with --force (set by 'force' or `packer build -force`).
func (p *PostProcessor) pushToOCIRegistry(ui packer.Ui, artifact packer.Artifact) (packer.Artifact, bool, bool, error) {
	if showErr := p.logPrePushShow(ui, artifact); showErr != nil {
		return artifact, true, false, showErr
	}

	ociPushParams := client.OCIPushParams{
		VMID:        artifact.String(),
		Remote:      p.config.Remote,
		Tag:         p.config.Tag,
		Description: p.config.Description,
		RemoteVM:    p.config.RemoteVM,
		Thin:        p.config.Thin,
		Force:       p.config.Force || p.config.PackerForce,
	}

	ociTagKind := "full"
	if p.config.Thin {
		ociTagKind = "thin"
	}
	ui.Say(fmt.Sprintf("Pushing template %s to OCI registry %s as %s OCI tag %s", artifact.String(), p.config.Remote, ociTagKind, p.config.Tag))

	pushErr := client.RunWithOutputStreamToUI(ui, func(outputStream chan string) error {
		return p.client.OCIPush(ociPushParams, outputStream)
	})
	if pushErr != nil {
		return artifact, true, false, pushErr
	}

	return p.finishSuccessfulPush(ui, artifact)
}

func (p *PostProcessor) registryParams() client.RegistryParams {
	return client.RegistryParams{
		Remote:       p.config.Remote,
		NodeCertPath: p.config.NodeCertPath,
		NodeKeyPath:  p.config.NodeKeyPath,
		CaRootPath:   p.config.CaRootPath,
		IsInsecure:   p.config.IsInsecure,
		HostArch:     p.config.HostArch,
	}
}

func (p *PostProcessor) logPrePushShow(ui packer.Ui, artifact packer.Artifact) error {
	showResponse, err := p.client.Show(artifact.String())
	if err != nil {
		return err
	}
	client.LogShowResponse(ui, "pre-push", showResponse)
	return nil
}

func (p *PostProcessor) finishSuccessfulPush(ui packer.Ui, artifact packer.Artifact) (packer.Artifact, bool, bool, error) {
	ui.Say("Registry push successful")

	// delete_template_post_push only after a successful remote registry push (not local-only tagging).
	if p.config.DeleteTemplatePostPush && !p.config.Local {
		deleteErr := p.deleteTemplatePostPushAfterSuccessfulRemotePush(ui, artifact)
		if deleteErr != nil {
			return artifact, true, false, deleteErr
		}
	}

	return artifact, true, false, nil
}

func (p *PostProcessor) deleteTemplatePostPushAfterSuccessfulRemotePush(ui packer.Ui, artifact packer.Artifact) error {
	localTemplateVMName := artifact.String()
	if localTemplateVMName == "" {
		return errors.New("delete_template_post_push is true but the Anka builder artifact has no VM name; refusing to run anka delete")
	}
	ui.Say(fmt.Sprintf("Deleting local VM template %q after successful remote registry push (delete_template_post_push)", localTemplateVMName))
	return p.client.Delete(client.DeleteParams{VMName: localTemplateVMName})
}
