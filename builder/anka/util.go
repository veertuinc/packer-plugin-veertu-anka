package anka

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

func stepError(ui packer.Ui, state multistep.StateBag, err error) multistep.StepAction {
	state.Put("error", err)
	ui.Error(err.Error())
	return multistep.ActionHalt
}

func convertDiskSizeToBytes(diskSize string) (error, uint64) {
	match, err := regexp.MatchString("^[0-9]+[g|G|m|M]$", diskSize)
	if err != nil {
		return err, uint64(0)
	}
	if !match {
		return fmt.Errorf("Input %s is not a valid disk size input", diskSize), 0
	}

	numericValue, err := strconv.Atoi(diskSize[:len(diskSize)-1])
	if err != nil {
		return err, uint64(0)
	}
	suffix := diskSize[len(diskSize)-1:]

	switch strings.ToUpper(suffix) {
	case "G":
		return nil, uint64(numericValue * 1024 * 1024 * 1024)
	case "M":
		return nil, uint64(numericValue * 1024 * 1024)
	default:
		return fmt.Errorf("Invalid disk size suffix: %s", suffix), uint64(0)
	}
}
