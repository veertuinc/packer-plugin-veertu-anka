package client

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"

	"github.com/veertuinc/packer-plugin-veertu-anka/common"
)

const (
	AnkaNameAlreadyExistsErrorCode   = 18
	AnkaVMNotFoundExceptionErrorCode = 3
)

// https://docs.veertu.com/anka/intel/command-line-reference/#clone
type CloneParams struct {
	VMName     string
	SourceUUID string
}

func (c *AnkaClient) Clone(params CloneParams) error {
	_, err := runAnkaCommand("clone", params.SourceUUID, params.VMName)
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

// https://docs.veertu.com/anka/intel/command-line-reference/#cp
type CopyParams struct {
	Src string
	Dst string
}

func (c *AnkaClient) Copy(params CopyParams) error {
	_, err := runAnkaCommand("cp", "-pRLf", params.Src, params.Dst)
	return err
}

type CreateParams struct {
	Name         string
	InstallerApp string
	OpticalDrive string
	RAMSize      string
	DiskSize     string
	VCPUCount    string
}

// https://docs.veertu.com/anka/intel/command-line-reference/#create
type CreateResponse struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	VCPUCores int    `json:"cpu_cores"`
	RAM       string `json:"ram"`
	ImageID   string `json:"image_id"`
	Status    string `json:"status"`
}

func (c *AnkaClient) Create(params CreateParams, outputStreamer chan string) (CreateResponse, error) {
	var response CreateResponse

	args := []string{
		"create",
		"--app", params.InstallerApp,
		"--ram-size", params.RAMSize,
		"--cpu-count", params.VCPUCount,
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

// https://docs.veertu.com/anka/intel/command-line-reference/#delete
type DeleteParams struct {
	VMName string
}

func (c *AnkaClient) Delete(params DeleteParams) error {
	args := []string{
		"delete",
		"--yes",
	}

	args = append(args, params.VMName)

	_, err := runAnkaCommand(args...)
	return err
}

// https://docs.veertu.com/anka/intel/command-line-reference/#describe
type DescribeResponse struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
	UUID    string `json:"uuid"`
	VCPU    struct {
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

	output, err := runAnkaCommand("describe", vmName)
	if err != nil {
		return response, err
	}

	err = json.Unmarshal(output.Body, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}

// https://docs.veertu.com/anka/intel/command-line-reference/#show
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

// https://docs.veertu.com/anka/intel/command-line-reference/#license
type LicenseResponse struct {
	LicenseType string `json:"license_type"`
	Status      string `json:"status"`
}

func (c *AnkaClient) License() (LicenseResponse, error) {
	var response LicenseResponse

	output, err := runAnkaCommand("license", "show")
	if err != nil {
		return response, err
	}

	err = json.Unmarshal(output.Body, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}

// https://docs.veertu.com/anka/intel/command-line-reference/#modify
func (c *AnkaClient) Modify(vmName string, command string, property string, flags ...string) error {
	ankaCommand := []string{"modify", vmName, command, property}
	ankaCommand = append(ankaCommand, flags...)

	output, err := runAnkaCommand(ankaCommand...)
	if err != nil {
		return err
	}
	if output.Status != "OK" {
		log.Print("Error executing modify command: ", output.ExceptionType, " ", output.Message)
		return fmt.Errorf(output.Message)
	}

	return nil
}

// https://docs.veertu.com/anka/intel/command-line-reference/#run
type RunParams struct {
	VMName            string
	Volume            string
	WaitForNetworking bool
	WaitForTimeSync   bool
	Command           []string
	Stdin             io.Reader
	Stdout, Stderr    io.Writer
	Debug             bool
	User              string
	FuseAvailable     bool
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

// https://docs.veertu.com/anka/intel/command-line-reference/#show
type ShowResponse struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	VCPUCores int    `json:"cpu_cores"`
	RAM       string `json:"ram"`
	ImageID   string `json:"image_id"`
	Status    string `json:"status"`
	HardDrive uint64 `json:"hard_drive"`
	Version   string `json:"version"`
}

func (sr ShowResponse) IsRunning() bool {
	return sr.Status == "running"
}

func (sr ShowResponse) IsStopped() bool {
	return sr.Status == "stopped"
}

func (sr ShowResponse) IsSuspended() bool {
	return sr.Status == "suspended"
}

func (c *AnkaClient) Show(vmName string) (ShowResponse, error) {
	var response ShowResponse

	output, err := runAnkaCommand("show", vmName)
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

// https://docs.veertu.com/anka/intel/command-line-reference/#start
type StartParams struct {
	VMName string
}

func (c *AnkaClient) Start(params StartParams) error {
	_, err := runAnkaCommand("start", params.VMName)
	return err
}

// https://docs.veertu.com/anka/intel/command-line-reference/#stop
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
	// Check if it's suspended, and do a run to start, then graceful stop
	showResponse, err := c.Show(params.VMName)
	if err != nil {
		return err
	}
	if showResponse.IsSuspended() {
		_, err = c.Run(RunParams{
			VMName:            params.VMName,
			WaitForNetworking: true,
			WaitForTimeSync:   true,
			Command:           []string{"true"},
		})
	}
	if err != nil {
		return err
	}

	_, err = runAnkaCommand(args...)
	return err
}

// https://docs.veertu.com/anka/intel/command-line-reference/#suspend
type SuspendParams struct {
	VMName string
}

func (c *AnkaClient) Suspend(params SuspendParams) error {
	_, err := runAnkaCommand("suspend", params.VMName)
	return err
}

// https://docs.veertu.com/anka/intel/command-line-reference/#start
func (c *AnkaClient) UpdateAddons(vmName string) error {
	args := []string{"start", "--update-addons", vmName}

	_, err := runAnkaCommand(args...)
	return err
}

// https://docs.veertu.com/anka/intel/command-line-reference/#version
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

func (c *AnkaClient) FuseAvailable(vmName string) bool {
	exitCode, _ := c.Run(RunParams{
		VMName:  vmName,
		Command: []string{"kextstat | grep \"com.veertu.filesystems.vtufs\" &>/dev/null"},
	})
	return exitCode == 0
}
