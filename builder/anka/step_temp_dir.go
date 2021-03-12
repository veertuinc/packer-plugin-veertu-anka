package anka

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-builder-veertu-anka/util"
)

var (
	err     error
	tempdir string
)

// StepTempDir creates a temporary directory that we use in order to
// share data with the anka vm over the communicator.
type StepTempDir struct {
	tempDir string
}

// Run will create the temporary directory used by the anka vm
func (s *StepTempDir) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	util := state.Get("util").(util.Util)
	onError := func(err error) multistep.StepAction {
		return util.StepError(ui, state, err)
	}

	ui.Say("Creating a temporary directory for sharing data...")

	configTmpDir, err := util.ConfigTmpDir()
	if err != nil {
		err := fmt.Errorf("Error making temp dir: %s", err)
		return onError(err)
	}

	tempdir, err = ioutil.TempDir(configTmpDir, "packer-anka")

	s.tempDir = tempdir
	state.Put("temp_dir", s.tempDir)

	return multistep.ActionContinue
}

// Cleanup is run when errors occur
// Will cleanup the temporary directory if something fails
func (s *StepTempDir) Cleanup(state multistep.StateBag) {
	os.RemoveAll(s.tempDir)
}
