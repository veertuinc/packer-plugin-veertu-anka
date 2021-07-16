//go:generate packer-sdc mapstructure-to-hcl2 -type Config,PortForwardingRule

package anka

import (
	"errors"
	"os"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/mitchellh/mapstructure"
	"github.com/veertuinc/packer-plugin-veertu-anka/util"
)

const defaultBootDelay = "7s"

// PortForwardingRule defines the requirements for port forwarding
type PortForwardingRule struct {
	PortForwardingGuestPort int    `mapstructure:"port_forwarding_guest_port"`
	PortForwardingHostPort  int    `mapstructure:"port_forwarding_host_port"`
	PortForwardingRuleName  string `mapstructure:"port_forwarding_rule_name"`
}

// Config initializes the builders using mapstructure which decodes
// generic map values from either the json or hcl2 config files provided
type Config struct {
	common.PackerConfig `mapstructure:",squash"`
	Comm                communicator.Config `mapstructure:",squash"`

	AnkaUser     string `mapstructure:"anka_user"`
	AnkaPassword string `mapstructure:"anka_password"`

	InstallerApp string `mapstructure:"installer_app"`
	SourceVMName string `mapstructure:"source_vm_name"`
	SourceVMTag  string `mapstructure:"source_vm_tag"`

	VMName    string `mapstructure:"vm_name"`
	DiskSize  string `mapstructure:"disk_size"`
	RAMSize   string `mapstructure:"ram_size"`
	VCPUCount string `mapstructure:"vcpu_count"`

	AlwaysFetch bool `mapstructure:"always_fetch"`

	UpdateAddons bool `mapstructure:"update_addons"`

	RegistryName string `mapstructure:"registry_name"`
	RegistryURL  string `mapstructure:"registry_path"`
	NodeCertPath string `mapstructure:"cert"`
	NodeKeyPath  string `mapstructure:"key"`
	CaRootPath   string `mapstructure:"cacert"`
	IsInsecure   bool   `mapstructure:"insecure"`

	PortForwardingRules []PortForwardingRule `mapstructure:"port_forwarding_rules"`

	HWUUID    string `mapstructure:"hw_uuid,omitempty"`
	BootDelay string `mapstructure:"boot_delay"`
	UseAnkaCP bool   `mapstructure:"use_anka_cp"`

	StopVM bool `mapstructure:"stop_vm"`

	ctx interpolate.Context //nolint:structcheck
}

// NewConfig generates a machine readable config from the generic map values above
// and provides an inital setup values and scrubs the data for any mistakes
func NewConfig(raws ...interface{}) (*Config, error) {
	var c Config
	util := util.AnkaUtil{}

	var md mapstructure.Metadata
	err := config.Decode(&c, &config.DecodeOpts{
		Metadata:    &md,
		Interpolate: true,
	}, raws...)
	if err != nil {
		return nil, err
	}

	if c.BootDelay == "" {
		c.BootDelay = defaultBootDelay
	}

	if c.AnkaPassword != "" {
		os.Setenv("ANKA_DEFAULT_PASSWD", c.AnkaPassword)
	}

	if c.AnkaUser != "" {
		os.Setenv("ANKA_DEFAULT_USER", c.AnkaUser)
	}

	var errs *packer.MultiError

	if c.Comm.Type == "" {
		c.Comm.Type = "anka"
	}

	if c.InstallerApp == "" && c.SourceVMName == "" {
		errs = packer.MultiErrorAppend(errs, errors.New("installer_app or source_vm_name must be specified"))
	}

	if c.InstallerApp != "" && c.SourceVMName != "" {
		errs = packer.MultiErrorAppend(errs, errors.New("cannot specify both an installer_app and source_vm_name"))
	}

	if c.SourceVMName != "" && strings.ContainsAny(c.SourceVMName, " \n") {
		errs = packer.MultiErrorAppend(errs, errors.New("source_vm_name name contains spaces"))
	}

	if len(c.PortForwardingRules) > 0 {
		for index, rule := range c.PortForwardingRules {
			if rule.PortForwardingGuestPort == 0 {
				errs = packer.MultiErrorAppend(errs, errors.New("guest port is required"))
			}
			if rule.PortForwardingRuleName == "" {
				c.PortForwardingRules[index].PortForwardingRuleName = util.RandSeq(10)
			}
		}
	}

	if errs != nil && len(errs.Errors) > 0 {
		return nil, errs
	}

	return &c, nil
}
