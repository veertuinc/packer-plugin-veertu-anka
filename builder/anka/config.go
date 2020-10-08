//go:generate mapstructure-to-hcl2 -type Config
package anka

import (
	"errors"

	"github.com/hashicorp/packer/common"
	"github.com/hashicorp/packer/helper/communicator"
	"github.com/hashicorp/packer/helper/config"
	"github.com/hashicorp/packer/packer"
	"github.com/hashicorp/packer/template/interpolate"
	"github.com/mitchellh/mapstructure"
)

type Config struct {
	common.PackerConfig `mapstructure:",squash"`
	Comm                communicator.Config `mapstructure:",squash"`

	InstallerApp string `mapstructure:"installer_app"`
	SourceVMName string `mapstructure:"source_vm_name"`

	VMName   string `mapstructure:"vm_name"`
	DiskSize string `mapstructure:"disk_size"`
	RAMSize  string `mapstructure:"ram_size"`
	CPUCount string `mapstructure:"cpu_count"`

	BootDelay  string `mapstructure:"boot_delay"`
	EnableHtt  bool   `mapstructure:"enable_htt"`
	DisableHtt bool   `mapstructure:"disable_htt"`

	ctx interpolate.Context
}

func NewConfig(raws ...interface{}) (*Config, error) {
	var c Config

	var md mapstructure.Metadata
	err := config.Decode(&c, &config.DecodeOpts{
		Metadata:    &md,
		Interpolate: true,
	}, raws...)
	if err != nil {
		return nil, err
	}

	// Accumulate any errors
	var errs *packer.MultiError

	// Default to the normal anku communicator type
	if c.Comm.Type == "" {
		c.Comm.Type = "anka"
	}

	if c.InstallerApp == "" && c.SourceVMName == "" {
		errs = packer.MultiErrorAppend(errs, errors.New("installer_app or source_vm_name must be specified"))
	}

	if c.DiskSize == "" {
		c.DiskSize = "25G"
	}

	if c.CPUCount == "" {
		c.CPUCount = "2"
	}

	if c.RAMSize == "" {
		c.RAMSize = "2G"
	}

	if c.BootDelay == "" {
		c.BootDelay = "2s"
	}

	if errs != nil && len(errs.Errors) > 0 {
		return nil, errs
	}

	return &c, nil
}
