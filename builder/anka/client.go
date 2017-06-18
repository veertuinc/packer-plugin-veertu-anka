package anka

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/packer/packer"
	"github.com/hashicorp/packer/template/interpolate"
)

type Client struct {
	Ui  packer.Ui
	Ctx *interpolate.Context
}

type CreateDiskParams struct {
	DiskSize     string
	InstallerApp string
}

func (c *Client) Version() (string, error) {
	out, err := exec.Command("anka", "version").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Client) CreateDisk(params CreateDiskParams) (string, error) {
	var stderr bytes.Buffer
	var stdout bytes.Buffer

	cmd := exec.Command("anka", "create-disk", "--size", params.DiskSize, "--app", params.InstallerApp)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	log.Printf("Creating disk with %#v", params)
	if err := cmd.Start(); err != nil {
		return "", err
	}

	if err := cmd.Wait(); err != nil {
		err = fmt.Errorf("Error creating disk: %s\nStderr: %s",
			err, stderr.String())
		return "", err
	}

	log.Println(stderr.String(), stdout.String())

	re := regexp.MustCompile(`disk (.+?) created successfully`)
	matches := re.FindStringSubmatch(strings.TrimSpace(stdout.String()))

	log.Println(matches)

	if len(matches) == 0 {
		return "", fmt.Errorf(
			"Unknown error creating disk\nStderr: %s\n Stdout: %s",
			stderr.String(), stdout.String())
	}

	return matches[1], nil
}

type CreateVMParams struct {
	ImageID  string
	Name     string
	RamSize  string
	CPUCount int
}

func (c *Client) CreateVM(params CreateVMParams) (string, error) {
	var stderr bytes.Buffer
	var stdout bytes.Buffer

	cmd := exec.Command(
		"anka",
		"create",
		"--image-id", params.ImageID,
		"--ram-size", params.RamSize,
		"--cpu-count", strconv.Itoa(params.CPUCount),
		params.Name,
	)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	log.Printf("Creating a VM with %#v", params)
	if err := cmd.Start(); err != nil {
		return "", err
	}

	if err := cmd.Wait(); err != nil {
		err = fmt.Errorf("Error creating disk: %s\nStderr: %s",
			err, stderr.String())
		return "", err
	}

	// re := regexp.MustCompile(`disk (.+?) created successfully`)
	// matches := re.FindStringSubmatch(strings.TrimSpace(stdout.String()))
	// return matches[1]

	log.Println(strings.TrimSpace(stdout.String()))
	return "", errors.New("CreateVM Not implemented")
}
