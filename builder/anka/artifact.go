package anka

import (
	"errors"
)

// Artifact represents an Anka image as the result of a Packer build.
type Artifact struct {
	vmName    string
	vmId      string
	StateData map[string]interface{}
}

// NewVMTemplateArtifact returns an artifact for the named VM template (used by tests and post-processors that need a concrete builder artifact).
func NewVMTemplateArtifact(vmName, vmID string) *Artifact {
	return &Artifact{
		vmName:    vmName,
		vmId:      vmID,
		StateData: map[string]any{},
	}
}

// BuilderId returns the unique builder id.
func (*Artifact) BuilderId() string {
	return BuilderId
}

// Destroy destroys the image represented by the artifact.
func (a *Artifact) Destroy() error {
	return errors.New("Destroy not implemented")
}

// Files returns the files represented by the artifact.
func (a *Artifact) Files() []string {
	return nil
}

// Id returns the VM UUID.
func (a *Artifact) Id() string {
	return a.vmId
}

// State allows the caller to ask for builder specific state information
// relating to the artifact instance.
func (a *Artifact) State(name string) interface{} {
	return a.StateData[name]
}

// String returns the string representation of the artifact.
func (a *Artifact) String() string {
	return a.vmName
}
