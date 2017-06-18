package anka

// // Artifact represents an Anka image as the result of a Packer build.
// type Artifact struct {
// 	imageName string
// 	imageId   string
// }

// // BuilderId returns the builder Id.
// func (*Artifact) BuilderId() string {
// 	return BuilderId
// }

// // Destroy destroys the Softlayer image represented by the artifact.
// func (self *Artifact) Destroy() error {
// 	log.Printf("Destroying image: %s", self.String())
// 	err := self.client.destroyImage(self.imageId)
// 	return err
// }

// // Files returns the files represented by the artifact.
// func (*Artifact) Files() []string {
// 	return nil
// }

// // Id returns the Softlayer image ID.
// func (self *Artifact) Id() string {
// 	return self.imageId
// }

// func (self *Artifact) State(name string) interface{} {
// 	return nil
// }

// // String returns the string representation of the artifact.
// func (self *Artifact) String() string {
// 	return fmt.Sprintf("%s::%s (%s)", self.datacenterName, self.imageId, self.imageName)
// }
