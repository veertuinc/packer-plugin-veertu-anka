//go:generate mapstructure-to-hcl2 -type Config
package anka

import (
	"errors"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/mitchellh/mapstructure"
)

const DEFAULT_BOOT_DELAY = "10s"

type Config struct {
	common.PackerConfig `mapstructure:",squash"`
	Comm                communicator.Config `mapstructure:",squash"`

	InstallerApp string `mapstructure:"installer_app"`
	SourceVMName string `mapstructure:"source_vm_name"`

	VMName   string `mapstructure:"vm_name"`
	DiskSize string `mapstructure:"disk_size"`
	RAMSize  string `mapstructure:"ram_size"`
	CPUCount string `mapstructure:"cpu_count"`

	PortForwardingRules []struct {
		PortForwardingGuestPort int    `mapstructure:"port_forwarding_guest_port"`
		PortForwardingHostPort  int    `mapstructure:"port_forwarding_host_port"`
		PortForwardingRuleName  string `mapstructure:"port_forwarding_rule_name"`
	} `mapstructure:"port_forwarding_rules,omitempty"`

	HWUUID     string `mapstructure:"hw_uuid,omitempty"`
	BootDelay  string `mapstructure:"boot_delay"`
	EnableHtt  bool   `mapstructure:"enable_htt"`
	DisableHtt bool   `mapstructure:"disable_htt"`
	UseAnkaCP  bool   `mapstructure:"use_anka_cp"`

	ctx interpolate.Context //nolint:structcheck
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

	// Handle Port Forwarding Rules
	if len(c.PortForwardingRules) > 0 {
		for index, rule := range c.PortForwardingRules {
			if rule.PortForwardingGuestPort == 0 {
				errs = packer.MultiErrorAppend(errs, errors.New("guest port is required"))
			}
			if rule.PortForwardingRuleName == "" {
				c.PortForwardingRules[index].PortForwardingRuleName = randSeq(10)
			}
		}
	}

	if strings.ContainsAny(c.SourceVMName, " \n") {
		errs = packer.MultiErrorAppend(errs, errors.New("source_vm_name name contains spaces"))
	}

	if c.BootDelay == "" {
		c.BootDelay = DEFAULT_BOOT_DELAY
	}

	if errs != nil && len(errs.Errors) > 0 {
		return nil, errs
	}

	return &c, nil
}
