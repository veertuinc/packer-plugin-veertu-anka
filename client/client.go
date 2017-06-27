package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type Client struct {
}

func (c *Client) Version() (string, error) {
	out, err := exec.Command("anka", "version").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

type SuspendParams struct {
	VMName string
}

func (c *Client) Suspend(params SuspendParams) error {
	_, err := runAnkaCommand("suspend", params.VMName)
	return err
}

type StartParams struct {
	VMName string
}

func (c *Client) Start(params StartParams) error {
	_, err := runAnkaCommand("start", params.VMName)
	return err
}

func (c *Client) Run(params RunParams) error {
	runner := NewRunner(params)

	err := runner.Start()
	if err != nil {
		return err
	}

	log.Printf("Waiting for command to run")
	return runner.Wait()
}

type CreateDiskParams struct {
	DiskSize     string
	InstallerApp string
}

func (c *Client) CreateDisk(params CreateDiskParams) (string, error) {
	var stderr bytes.Buffer
	var stdout bytes.Buffer

	cmd := exec.Command(
		"anka",
		// "--machine-readable", NB: isn't supported yet
		"create-disk",
		"--size",
		params.DiskSize,
		"--app",
		params.InstallerApp,
	)
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

	re := regexp.MustCompile(`disk (.+?) created successfully`)
	matches := re.FindStringSubmatch(strings.TrimSpace(stdout.String()))

	if len(matches) == 0 {
		return "", fmt.Errorf(
			"Unknown error creating disk\nStderr: %s\n Stdout: %s",
			stderr.String(), stdout.String())
	}

	return matches[1], nil
}

type CreateParams struct {
	ImageID  string
	Name     string
	RamSize  string
	CPUCount int
}

type CreateResponse struct {
	UUID string `json:"uuid"`
}

func (c *Client) Create(params CreateParams) (CreateResponse, error) {
	args := []string{
		"create",
		"--image-id", params.ImageID,
		"--ram-size", params.RamSize,
		"--cpu-count", strconv.Itoa(params.CPUCount),
		params.Name,
	}

	output, err := runAnkaCommand(args...)
	if err != nil {
		return CreateResponse{}, err
	}

	var response CreateResponse
	err = json.Unmarshal(output.Body, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}

type DescribeResponse struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
	UUID    string `json:"uuid"`
	CPU     struct {
		Cores int `json:"cores"`
	} `json:"cpu"`
	RAM string `json:"ram"`
	Usb struct {
		Tablet   int         `json:"tablet"`
		Kbd      int         `json:"kbd"`
		Host     interface{} `json:"host"`
		Location interface{} `json:"location"`
		PciSlot  int         `json:"pci_slot"`
		Mouse    int         `json:"mouse"`
	} `json:"usb"`
	OpticalDrives []interface{} `json:"optical_drives"`
	HardDrives    []struct {
		Controller string `json:"controller"`
		PciSlot    int    `json:"pci_slot"`
		File       string `json:"file"`
	} `json:"hard_drives"`
	NetworkCards []struct {
		Index               int           `json:"index"`
		Mode                string        `json:"mode"`
		MacAddress          string        `json:"mac_address"`
		PortForwardingRules []interface{} `json:"port_forwarding_rules"`
		PciSlot             int           `json:"pci_slot"`
		Type                string        `json:"type"`
	} `json:"network_cards"`
	Smbios struct {
		Type string `json:"type"`
	} `json:"smbios"`
	Smc struct {
		Type string `json:"type"`
	} `json:"smc"`
	Nvram    bool `json:"nvram"`
	Firmware struct {
		Type string `json:"type"`
	} `json:"firmware"`
	Display struct {
		Headless    int `json:"headless"`
		FrameBuffer struct {
			PciSlot  int    `json:"pci_slot"`
			VncPort  int    `json:"vnc_port"`
			Height   int    `json:"height"`
			Width    int    `json:"width"`
			VncIP    string `json:"vnc_ip"`
			Password string `json:"password"`
		} `json:"frame_buffer"`
	} `json:"display"`
}

func (c *Client) Describe(vmName string) (DescribeResponse, error) {
	output, err := runAnkaCommand("describe", vmName)
	if err != nil {
		return DescribeResponse{}, err
	}

	var response DescribeResponse
	err = json.Unmarshal(output.Body, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}

type ShowResponse struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	CPUCores int    `json:"cpu_cores"`
	RAM      string `json:"ram"`
	ImageID  string `json:"image_id"`
	Status   string `json:"status"`
}

func (c *Client) Show(vmName string) (ShowResponse, error) {
	output, err := runAnkaCommand("info", vmName)
	if err != nil {
		return ShowResponse{}, err
	}

	var response ShowResponse
	err = json.Unmarshal(output.Body, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}

type CloneParams struct {
	VMName     string
	SourceUUID string
}

func (c *Client) Clone(params CloneParams) error {
	_, err := runAnkaCommand("clone", params.SourceUUID, params.VMName)
	if err != nil {
		return err
	}

	return nil
}

type StopParams struct {
	VMName string
	Force  bool
}

func (c *Client) Stop(params StopParams) error {
	args := []string{
		"stop",
	}

	if params.Force {
		args = append(args, "--force")
	}

	args = append(args, params.VMName)
	_, err := runAnkaCommand(args...)
	return err
}

type DeleteParams struct {
	VMName string
	Force  bool
}

func (c *Client) Delete(params DeleteParams) error {
	args := []string{
		"delete",
	}

	if params.Force {
		args = append(args, "--force")
	}

	args = append(args, params.VMName)
	_, err := runAnkaCommand(args...)
	return err
}

func runAnkaCommand(args ...string) (machineReadableOutput, error) {
	log.Printf("Executing anka --machine-readable %s", strings.Join(args, " "))

	cmdArgs := append([]string{"--machine-readable"}, args...)
	cmd := exec.Command("anka", cmdArgs...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed with an error of %v", err)
	}

	parsed, err := parseOutput(output)
	if err != nil {
		return machineReadableOutput{}, err
	}

	if err = parsed.GetError(); err != nil {
		return machineReadableOutput{}, err
	}

	return parsed, nil
}

const (
	statusOK    = "OK"
	statusERROR = "ERROR"
)

type machineReadableOutput struct {
	Status        string `json:"status"`
	Body          json.RawMessage
	Message       string `json:"message"`
	Code          int    `json:"code"`
	ExceptionType string `json:"exception_type"`
}

func (parsed *machineReadableOutput) GetError() error {
	if parsed.Status != statusOK {
		return errors.New(parsed.Message)
	}
	return nil
}

func parseOutput(output []byte) (machineReadableOutput, error) {
	log.Printf("Response JSON: %s", output)

	var parsed machineReadableOutput
	if err := json.Unmarshal(output, &parsed); err != nil {
		return parsed, err
	}

	// log.Printf("Response %#v", parsed)
	return parsed, nil
}
