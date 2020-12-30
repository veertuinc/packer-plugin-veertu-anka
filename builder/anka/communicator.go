package anka

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/hashicorp/packer/packer"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
)

type Communicator struct {
	Config  *Config
	Client  *client.Client
	HostDir string
	VMDir   string
	VMName  string
}

func (c *Communicator) Start(ctx context.Context, remote *packer.RemoteCmd) error {
	log.Printf("Communicator Start: %s", remote.Command)

	runner := client.NewRunner(client.RunParams{
		VMName:  c.VMName,
		Command: []string{remote.Command},
		Volume:  "",
		Stdout:  remote.Stdout,
		Stderr:  remote.Stderr,
		Stdin:   remote.Stdin,
	})

	if err := runner.Start(); err != nil {
		return err
	}

	go func() {
		err, exitCode := runner.Wait()
		if err != nil {
			log.Printf("Runner exited with error: %v", err)
		}
		remote.SetExited(exitCode)
	}()

	return nil

}

func (c *Communicator) Upload(dst string, src io.Reader, fi *os.FileInfo) error {
	log.Printf("Uploading file to VM: %s", dst)

	// Create a temporary file to store the upload
	tempfile, err := ioutil.TempFile(c.HostDir, "upload")
	if err != nil {
		return err
	}
	defer os.Remove(tempfile.Name())
	defer tempfile.Close()

	log.Printf("Copying from reader to %s", tempfile.Name())
	w, err := io.Copy(tempfile, src)
	if err != nil {
		return err
	}

	if fi != nil {
		tempfile.Chmod((*fi).Mode())
	}
	tempfile.Close()

	err = c.Client.Copy(client.CopyParams{
		Src: tempfile.Name(),
		Dst: c.VMName + ":" + dst,
	})

	log.Printf("Copied %d bytes from %s to %s", w, tempfile.Name(), dst)
	return err
}

func (c *Communicator) UploadDir(dst string, src string, exclude []string) error {
	return c.Client.Copy(client.CopyParams{
		Src: src,
		Dst: c.VMName + ":" + dst,
	})
}

func (c *Communicator) Download(src string, dst io.Writer) error {
	log.Printf("Downloading file from VM: %s", src)

	// Create a temporary file to store the download
	tempfile, err := ioutil.TempFile(c.HostDir, "download")
	if err != nil {
		return err
	}
	defer os.Remove(tempfile.Name())
	defer tempfile.Close()

	err = c.Client.Copy(client.CopyParams{
		Src: c.VMName + ":" + src,
		Dst: tempfile.Name(),
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
	return c.Client.Copy(client.CopyParams{
		Src: c.VMName + ":" + src,
		Dst: dst,
	})
}
