package anka

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/pathing"
)

// StepTempDir creates a temporary directory that we use in order to
// share data with the anka vm over the communicator.
type StepTempDir struct {
	tempDir string
}

func (s *StepTempDir) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)

	ui.Say("Creating a temporary directory for sharing data...")

	var err error
	var tempdir string

	configTmpDir, err := ConfigTmpDir()
	if err == nil {
		tempdir, err = ioutil.TempDir(configTmpDir, "packer-anka")
	}
	if err != nil {
		err := fmt.Errorf("Error making temp dir: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	s.tempDir = tempdir
	state.Put("temp_dir", s.tempDir)
	return multistep.ActionContinue
}

func (s *StepTempDir) Cleanup(state multistep.StateBag) {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

func ConfigTmpDir() (string, error) {
	configdir, err := pathing.ConfigDir()
	if err != nil {
		return "", err
	}
	if tmpdir := os.Getenv("PACKER_TMP_DIR"); tmpdir != "" {
		// override the config dir with tmp dir. Still stat it and mkdirall if
		// necessary.
		fp, err := filepath.Abs(tmpdir)
		log.Printf("found PACKER_TMP_DIR env variable; setting tmpdir to %s", fp)
		if err != nil {
			return "", err
		}
		configdir = fp
	}

	_, err = os.Stat(configdir)
	if os.IsNotExist(err) {
		log.Printf("Config dir %s does not exist; creating...", configdir)
		if err = os.MkdirAll(configdir, 0755); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	td, err := ioutil.TempDir(configdir, "tmp")
	if err != nil {
		return "", fmt.Errorf("Error creating temp dir: %s", err)

	}
	log.Printf("Set Packer temp dir to %s", td)
	return td, nil
}
