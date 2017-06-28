package anka

import (
	"errors"
)

// Artifact represents an Anka image as the result of a Packer build.
type Artifact struct {
	vmName string
	vmId   string
}

// BuilderId returns the builder Id.
func (*Artifact) BuilderId() string {
	return BuilderId
}

// Destroy destroys the image represented by the artifact.
func (self *Artifact) Destroy() error {
	// log.Printf("Destroying image: %s", self.String())
	// err := self.client.destroyImage(self.imageId)
	return errors.New("Destroy not implemented")
}

// Files returns the files represented by the artifact.
func (*Artifact) Files() []string {
	return nil
}

// Id returns the VM UUID.
func (self *Artifact) Id() string {
	return self.vmId
}

func (self *Artifact) State(name string) interface{} {
	return nil
}

// String returns the string representation of the artifact.
func (self *Artifact) String() string {
	return self.vmName
}
