package anka

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/hashicorp/packer/packer"
)

type Communicator struct {
	Config  *Config
	Client  *Client
	HostDir string
	VMDir   string
	VMName  string
}

func (c *Communicator) Start(remote *packer.RemoteCmd) error {
	log.Printf("Communicator Start: %s", remote.Command)

	params := RunParams{
		VMName:      c.VMName,
		Command:     []string{"sh", "-c", remote.Command},
		VolumesFrom: c.HostDir,
		Stdout:      remote.Stdout,
		Stderr:      remote.Stderr,
		Stdin:       remote.Stdin,
	}

	resp, err := c.Client.RunAsync(params)
	if err != nil {
		return err
	}

	go remote.SetExited(resp.ExitStatus())
	return nil

}

func (c *Communicator) Upload(dst string, src io.Reader, fi *os.FileInfo) error {
	log.Printf("Communicator Upload")

	// Create a temporary file to store the upload
	tempfile, err := ioutil.TempFile(c.HostDir, "upload")
	if err != nil {
		return err
	}
	defer os.Remove(tempfile.Name())

	// Copy the contents to the temporary file
	w, err := io.Copy(tempfile, src)
	if err != nil {
		return err
	}

	if fi != nil {
		tempfile.Chmod((*fi).Mode())
	}
	tempfile.Close()

	log.Printf("Created temp dir in %s", tempfile.Name())
	log.Printf("Copying %d bytes from %s to %s", w, tempfile.Name(), dst)

	return c.Client.Run(RunParams{
		VMName:      c.VMName,
		Command:     []string{"cp", path.Base(tempfile.Name()), dst},
		VolumesFrom: c.HostDir,
	})
}

func (c *Communicator) UploadDir(dst string, src string, exclude []string) error {
	return errors.New("communicator.UploadDir isn't implemented")
}

func (c *Communicator) Download(src string, dst io.Writer) error {
	log.Printf("Downloading file from VM: %s", src)

	// Create a temporary file to store the download
	tempfile, err := ioutil.TempFile(c.HostDir, "download")
	if err != nil {
		return err
	}
	defer os.Remove(tempfile.Name())

	// Copy it to a local file mounted on shared fs
	err = c.Client.Run(RunParams{
		VMName:      c.VMName,
		Command:     []string{"cp", src, "./" + path.Base(tempfile.Name())},
		VolumesFrom: c.HostDir,
	})

	log.Printf("Copying from %s to writer", tempfile.Name())
	w, err := io.Copy(dst, tempfile)
	if err != nil {
		return err
	}

	log.Printf("Copied %d bytes", w)
	return nil
}

func (c *Communicator) DownloadDir(src string, dst string, exclude []string) error {
	return errors.New("communicator.DownloadDir isn't implemented")
}
