package anka

import (
	"errors"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-plugin-veertu-anka/client"
)

// VMLabel is one label operation applied with anka modify.
type VMLabel struct {
	Name      string `mapstructure:"name,omitempty"`
	Value     string `mapstructure:"value,omitempty"`
	Delete    bool   `mapstructure:"delete,omitempty"`
	SetName   string `mapstructure:"set_name,omitempty"`
	DeleteAll bool   `mapstructure:"delete_all,omitempty"`
}

func validateVMLabels(labels []VMLabel) error {
	for index, label := range labels {
		if err := validateVMLabel(label); err != nil {
			return fmt.Errorf("vm_label[%d]: %w", index, err)
		}
	}
	return nil
}

func validateVMLabel(label VMLabel) error {
	if label.DeleteAll {
		if label.Name != "" || label.Value != "" || label.Delete || label.SetName != "" {
			return errors.New("delete_all cannot be combined with other fields")
		}
		return nil
	}
	if label.Delete {
		if label.Name == "" {
			return errors.New("name is required when delete is true")
		}
		if label.Value != "" || label.SetName != "" {
			return errors.New("delete cannot be combined with value or set_name")
		}
		return nil
	}
	if label.SetName != "" {
		if label.Name == "" {
			return errors.New("name is required when set_name is set")
		}
		if label.Value != "" {
			return errors.New("set_name cannot be combined with value")
		}
		return nil
	}
	if label.Name == "" || label.Value == "" {
		return errors.New("entry must be name+value, delete, set_name, or delete_all")
	}
	return nil
}

func applyVMLabels(
	ankaClient client.Client,
	stopParams client.StopParams,
	vmName string,
	labels []VMLabel,
	ui packer.Ui,
) error {
	if len(labels) == 0 {
		return nil
	}

	if err := ankaClient.Stop(stopParams); err != nil {
		return err
	}

	for _, label := range labels {
		switch {
		case label.DeleteAll:
			ui.Say(fmt.Sprintf("Deleting all labels on VM %s", vmName))
			if err := ankaClient.Modify(vmName, "delete", "label", "-a"); err != nil {
				return err
			}
		case label.Delete:
			ui.Say(fmt.Sprintf("Deleting label %s on VM %s", label.Name, vmName))
			if err := ankaClient.Modify(vmName, "label", "-d", label.Name); err != nil {
				return err
			}
		case label.SetName != "":
			ui.Say(fmt.Sprintf("Renaming label %s to %s on VM %s", label.Name, label.SetName, vmName))
			if err := ankaClient.Modify(vmName, "label", "--set-name", label.Name, label.SetName); err != nil {
				return err
			}
		default:
			ui.Say(fmt.Sprintf("Setting label %s=%s on VM %s", label.Name, label.Value, vmName))
			if err := ankaClient.Modify(vmName, "label", label.Name, label.Value); err != nil {
				return err
			}
		}
	}

	return nil
}
