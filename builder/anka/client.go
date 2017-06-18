package anka

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/hashicorp/packer/packer"
	"github.com/hashicorp/packer/template/interpolate"
)

type Client struct {
	Ui  packer.Ui
	Ctx *interpolate.Context
}

func (c *Client) CreateDisk(diskSize, installerApp string) error {
	var stderr bytes.Buffer
	var stdout bytes.Buffer

	cmd := exec.Command("anka", "create-disk", "--size", diskSize, "--app", installerApp)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	log.Printf("Creating a %s disk with %s", diskSize, installerApp)
	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		err = fmt.Errorf("Error creating disk: %s\nStderr: %s",
			err, stderr.String())
		return err
	}

	log.Println(strings.TrimSpace(stdout.String()))
	return errors.New("Not implemented")
}
