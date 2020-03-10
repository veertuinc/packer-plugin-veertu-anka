package anka

import (
	"github.com/hashicorp/packer/packer"
	"github.com/hashicorp/packer/helper/multistep"
)



func stepError(ui packer.Ui, state multistep.StateBag, err error) multistep.StepAction {
	state.Put("error", err)
	ui.Error(err.Error())
	return multistep.ActionHalt
}