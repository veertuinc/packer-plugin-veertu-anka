package client

import (
	"bytes"
	"encoding/json"
)

const (
	statusOK    = "OK"
	statusERROR = "ERROR"
)

type Client interface {
	Create(params CreateParams, outputStreamer chan string) (CreateResponse, error)
	Clone(params CloneParams) error
	Copy(params CopyParams) error
	Delete(params DeleteParams) error
	Describe(vmName string) (DescribeResponse, error)
	Exists(vmName string) (bool, error)
	Modify(vmName string, command string, property string, flags ...string) error
	RegistryList(registryParams RegistryParams) ([]RegistryListResponse, error)
	RegistryPull(registryParams RegistryParams, pullParams RegistryPullParams) error
	RegistryPush(registryParams RegistryParams, pushParams RegistryPushParams) error
	RegistryRevert(url string, id string) error
	Run(params RunParams) (error, int)
	Show(vmName string) (ShowResponse, error)
	Start(params StartParams) error
	Stop(params StopParams) error
	Suspend(params SuspendParams) error
	UpdateAddons(vmName string) error
	Version() (VersionResponse, error)
}

type AnkaClient struct {
}

type MachineReadableError struct {
	*MachineReadableOutput
}

func (ae MachineReadableError) Error() string {
	return ae.Message
}

type MachineReadableOutput struct {
	Status        string `json:"status"`
	Body          json.RawMessage
	Message       string `json:"message"`
	Code          int    `json:"code"`
	ExceptionType string `json:"exception_type"`
}

func (parsed *MachineReadableOutput) GetError() error {
	if parsed.Status != statusOK {
		return MachineReadableError{parsed}
	}
	return nil
}

func parseOutput(output []byte) (MachineReadableOutput, error) {
	var parsed MachineReadableOutput
	if err := json.Unmarshal(output, &parsed); err != nil {
		return parsed, err
	}

	return parsed, nil
}

func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func customSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// A tiny spin off on ScanLines

	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, dropCR(data[0:i]), nil
	}
	if atEOF { // Machine readable data is parsed here
		out := dropCR(data)
		return len(data), out, customErr{data: out}
	}
	return 0, nil, nil
}

type customErr struct {
	data []byte
}

func (e customErr) Error() string {
	return string(e.data)
}
