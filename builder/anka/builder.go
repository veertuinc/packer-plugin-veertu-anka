package anka

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/packerbuilderdata"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
	"github.com/veertuinc/packer-builder-veertu-anka/util"
)

// BuilderId is the unique ID for this builder.
const BuilderId = "packer.veertu-anka"

// Builder represents a Packer Builder.
type Builder struct {
	config *Config
	runner multistep.Runner
}

// Prepare processes the build configuration parameters.
func (b *Builder) Prepare(raws ...interface{}) ([]string, []string, error) {
	generatedData := []string{"VMName", "OSVersion", "DarwinVersion"}

	c, errs := NewConfig(raws...)
	if errs != nil {
		return nil, nil, errs
	}
	b.config = c

	return generatedData, nil, nil
}

// Run executes an Anka Packer build and returns a packer.Artifact
func (b *Builder) Run(ctx context.Context, ui packer.Ui, hook packer.Hook) (packer.Artifact, error) {
	ankaClient := &client.AnkaClient{}
	util := &util.AnkaUtil{}

	// Setup the state bag and initial state for the steps
	state := new(multistep.BasicStateBag)
	state.Put("config", b.config)
	state.Put("hook", hook)
	state.Put("ui", ui)
	state.Put("client", ankaClient)
	state.Put("util", util)

	generatedData := &packerbuilderdata.GeneratedData{State: state}

	steps := []multistep.Step{
		&StepTempDir{},
	}

	switch b.config.PackerConfig.PackerBuilderType {
	case "veertu-anka-vm-create":
		steps = append(steps, &StepCreateVM{})
	case "veertu-anka-vm-clone":
		steps = append(steps, &StepCloneVM{})
	default:
		return nil, errors.New("wrong type for builder. must be of type clone or create")
	}

	steps = append(steps,
		&StepStartVM{},
		&communicator.StepConnect{
			Config: &b.config.Comm,
			CustomConnect: map[string]multistep.Step{
				"anka": &StepConnectAnka{},
			},
			Host: func(state multistep.StateBag) (string, error) {
				return "", errors.New("No host implemented for anka builder (which is ok)")
			},
		},
		&StepSetGeneratedData{
			GeneratedData: generatedData,
		},
		&commonsteps.StepProvision{},
	)

	// Run!
	b.runner = commonsteps.NewRunner(steps, b.config.PackerConfig, ui)
	b.runner.Run(ctx, state)

	// If there was an error, return that
	rawErr, ok := state.GetOk("error")
	if ok {
		return nil, rawErr.(error)
	}

	// If it was cancelled, then just return
	_, ok = state.GetOk(multistep.StateCancelled)
	if ok {
		return nil, nil
	}

	// Check we can describe the VM
	descr, err := ankaClient.Describe(state.Get("vm_name").(string))
	if err != nil {
		return nil, err
	}

	license, err := ankaClient.License()
	if err != nil {
		return nil, err
	}

	if license.LicenseType == "com.veertu.anka.develop" {
		log.Printf("developer license present, can only stop vms: https://ankadocs.veertu.com/docs/licensing/#anka-license-feature-differences")
		b.config.StopVM = true
	}

	if b.config.StopVM {
		ui.Say(fmt.Sprintf("Stopping VM %s", descr.Name))

		err := ankaClient.Stop(client.StopParams{VMName: descr.Name})
		if err != nil {
			return nil, err
		}
	} else {
		ui.Say(fmt.Sprintf("Suspending VM %s", descr.Name))

		err := ankaClient.Suspend(client.SuspendParams{VMName: descr.Name})
		if err != nil {
			return nil, err
		}
	}

	// No errors, must've worked
	return &Artifact{
		vmId:      descr.UUID,
		vmName:    descr.Name,
		StateData: map[string]interface{}{"generated_data": generatedData.State.Get("generated_data")},
	}, nil
}

// ConfigSpec returns an HCL spec of the config
func (b *Builder) ConfigSpec() hcldec.ObjectSpec {
	return b.config.FlatMapstructure().HCL2Spec()
}
