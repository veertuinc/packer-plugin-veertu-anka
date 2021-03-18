package client

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"

	"github.com/veertuinc/packer-builder-veertu-anka/common"
)

const (
	AnkaNameAlreadyExistsErrorCode   = 18
	AnkaVMNotFoundExceptionErrorCode = 3
)

type CloneParams struct {
	VMName     string
	SourceUUID string
}

func (c *AnkaClient) Clone(params CloneParams) error {
	_, err := runCommand("clone", params.SourceUUID, params.VMName)
	if err != nil {
		merr, ok := err.(MachineReadableError)
		if ok {
			if merr.Code == AnkaNameAlreadyExistsErrorCode {
				return &common.VMAlreadyExistsError{}
			}
		}
		return err
	}

	return nil
}

type CopyParams struct {
	Src string
	Dst string
}

func (c *AnkaClient) Copy(params CopyParams) error {
	_, err := runCommand("cp", "-af", params.Src, params.Dst)
	return err
}

type CreateParams struct {
	Name         string
	InstallerApp string
	OpticalDrive string
	RAMSize      string
	DiskSize     string
	CPUCount     string
}

type CreateResponse struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	CPUCores int    `json:"cpu_cores"`
	RAM      string `json:"ram"`
	ImageID  string `json:"image_id"`
	Status   string `json:"status"`
}

func (c *AnkaClient) Create(params CreateParams, outputStreamer chan string) (CreateResponse, error) {
	var response CreateResponse

	args := []string{
		"create",
		"--app", params.InstallerApp,
		"--ram-size", params.RAMSize,
		"--cpu-count", params.CPUCount,
		"--disk-size", params.DiskSize,
		params.Name,
	}

	output, err := runCommandStreamer(outputStreamer, args...)
	if err != nil {
		return response, err
	}

	err = json.Unmarshal(output.Body, &response)
	if err != nil {
		return response, fmt.Errorf("Failed parsing output: %q (%v)", output.Body, err)
	}

	return response, nil
}

type DeleteParams struct {
	VMName string
}

func (c *AnkaClient) Delete(params DeleteParams) error {
	args := []string{
		"delete",
		"--yes",
	}

	args = append(args, params.VMName)

	_, err := runCommand(args...)
	return err
}

type DescribeResponse struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
	UUID    string `json:"uuid"`
	CPU     struct {
		Cores   int `json:"cores"`
		Threads int `json:"threads"`
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
		Index               int    `json:"index"`
		Mode                string `json:"mode"`
		MacAddress          string `json:"mac_address"`
		PortForwardingRules []struct {
			GuestPort int    `json:"guest_port"`
			RuleName  string `json:"rule_name"`
			Protocol  string `json:"protocol"`
			HostIP    string `json:"host_ip"`
			HostPort  int    `json:"host_port"`
		} `json:"port_forwarding_rules"`
		PciSlot int    `json:"pci_slot"`
		Type    string `json:"type"`
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

func (c *AnkaClient) Describe(vmName string) (DescribeResponse, error) {
	var response DescribeResponse

	output, err := runCommand("describe", vmName)
	if err != nil {
		return response, err
	}

	err = json.Unmarshal(output.Body, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}

func (c *AnkaClient) Exists(vmName string) (bool, error) {
	_, err := c.Show(vmName)
	if err == nil {
		return true, nil
	}

	switch err.(type) {
	// case *json.UnmarshalTypeError:
	case *common.VMNotFoundException:
		return false, nil
	}

	return false, err
}

func (c *AnkaClient) Modify(vmName string, command string, property string, flags ...string) error {
	ankaCommand := []string{"modify", vmName, command, property}
	ankaCommand = append(ankaCommand, flags...)

	output, err := runCommand(ankaCommand...)
	if err != nil {
		return err
	}
	if output.Status != "OK" {
		log.Print("Error executing modify command: ", output.ExceptionType, " ", output.Message)
		return fmt.Errorf(output.Message)
	}

	return nil
}

type RunParams struct {
	VMName         string
	Volume         string
	Command        []string
	Stdin          io.Reader
	Stdout, Stderr io.Writer
	Debug          bool
	User           string
}

func (c *AnkaClient) Run(params RunParams) (int, error) {
	runner := NewRunner(params)

	err := runner.Start()
	if err != nil {
		return 1, err
	}

	log.Printf("Waiting for command to run")
	return runner.Wait()
}

type ShowResponse struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	CPUCores  int    `json:"cpu_cores"`
	RAM       string `json:"ram"`
	ImageID   string `json:"image_id"`
	Status    string `json:"status"`
	HardDrive uint64 `json:"hard_drive"`
}

func (sr ShowResponse) IsRunning() bool {
	return sr.Status == "running"
}

func (sr ShowResponse) IsStopped() bool {
	return sr.Status == "stopped"
}

func (c *AnkaClient) Show(vmName string) (ShowResponse, error) {
	var response ShowResponse

	output, err := runCommand("show", vmName)
	if err != nil {
		merr, ok := err.(MachineReadableError)
		if ok {
			if merr.Code == AnkaVMNotFoundExceptionErrorCode {
				return response, &common.VMNotFoundException{}
			}
		}
		return response, err
	}

	err = json.Unmarshal(output.Body, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}

type StartParams struct {
	VMName string
}

func (c *AnkaClient) Start(params StartParams) error {
	_, err := runCommand("start", params.VMName)
	return err
}

type StopParams struct {
	VMName string
	Force  bool
}

func (c *AnkaClient) Stop(params StopParams) error {
	args := []string{
		"stop",
	}

	if params.Force {
		args = append(args, "--force")
	}

	args = append(args, params.VMName)

	_, err := runCommand(args...)
	return err
}

type SuspendParams struct {
	VMName string
}

func (c *AnkaClient) Suspend(params SuspendParams) error {
	_, err := runCommand("suspend", params.VMName)
	return err
}

func (c *AnkaClient) UpdateAddons(vmName string) error {
	args := []string{"start", "--update-addons", vmName}

	_, err := runCommand(args...)
	return err
}

type VersionResponse struct {
	Status string              `json:"status"`
	Body   VersionResponseBody `json:"body"`
}

type VersionResponseBody struct {
	Product string `json:"product"`
	Version string `json:"version"`
	Build   string `json:"build"`
}

func (c *AnkaClient) Version() (VersionResponse, error) {
	var response VersionResponse

	out, err := exec.Command("anka", "--machine-readable", "version").Output()
	if err != nil {
		return response, err
	}

	err = json.Unmarshal([]byte(out), &response)
	return response, err
}
