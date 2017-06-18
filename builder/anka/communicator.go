package anka

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync"

	"github.com/hashicorp/packer/packer"
)

type Communicator struct {
	Config  *Config
	Client  *Client
	HostDir string
	VMDir   string
	VMName  string
	lock    sync.Mutex
}

func (c *Communicator) Start(remote *packer.RemoteCmd) error {
	return errors.New("Start not implemented")
}

func (c *Communicator) Upload(dst string, src io.Reader, fi *os.FileInfo) error {
	log.Printf("Upload %#v", c)

	// Create a temporary file to store the upload
	tempfile, err := ioutil.TempFile(c.HostDir, "upload")
	if err != nil {
		return err
	}
	defer os.Remove(tempfile.Name())

	// Copy the contents to the temporary file
	_, err = io.Copy(tempfile, src)
	if err != nil {
		return err
	}

	if fi != nil {
		tempfile.Chmod((*fi).Mode())
	}
	tempfile.Close()

	log.Printf("Created temp dir in %s", tempfile.Name())
	log.Printf("Copying from %s to %s", tempfile.Name(), dst)

	err = c.Client.Run(RunParams{
		VMName:      c.VMName,
		VolumesFrom: c.HostDir,
		Command:     []string{"cp", "-a", path.Base(tempfile.Name()), dst},
	})

	log.Printf("Run result: %#v", err)

	// // Copy the file into place by copying the temporary file we put
	// // into the shared folder into the proper location in the container
	// cmd := &packer.RemoteCmd{
	// 	Command: fmt.Sprintf("command cp %s/%s %s", c.ContainerDir,
	// 		filepath.Base(tempfile.Name()), dst),
	// }

	// if err := c.Start(cmd); err != nil {
	// 	return err
	// }

	// // Wait for the copy to complete
	// cmd.Wait()
	// if cmd.ExitStatus != 0 {
	// 	return fmt.Errorf("Upload failed with non-zero exit status: %d", cmd.ExitStatus)
	// }

	// anka run -v . llamas ls
	return errors.New("communicator.Upload isn't implemented")
}

func (c *Communicator) UploadDir(dst string, src string, exclude []string) error {
	return errors.New("communicator.UploadDir isn't implemented")
}

func (c *Communicator) Download(src string, dst io.Writer) error {
	return errors.New("communicator.UploadDir isn't implemented")
}

func (c *Communicator) DownloadDir(src string, dst string, exclude []string) error {
	return errors.New("communicator.DownloadDir isn't implemented")
}

// // Runs the given command and blocks until completion
// func (c *Communicator) run(cmd *exec.Cmd, remote *packer.RemoteCmd, stdin io.WriteCloser, stdout, stderr io.ReadCloser) {
// 	// For Docker, remote communication must be serialized since it
// 	// only supports single execution.
// 	c.lock.Lock()
// 	defer c.lock.Unlock()

// 	wg := sync.WaitGroup{}
// 	repeat := func(w io.Writer, r io.ReadCloser) {
// 		io.Copy(w, r)
// 		r.Close()
// 		wg.Done()
// 	}

// 	if remote.Stdout != nil {
// 		wg.Add(1)
// 		go repeat(remote.Stdout, stdout)
// 	}

// 	if remote.Stderr != nil {
// 		wg.Add(1)
// 		go repeat(remote.Stderr, stderr)
// 	}

// 	// Start the command
// 	log.Printf("Executing %s:", strings.Join(cmd.Args, " "))
// 	if err := cmd.Start(); err != nil {
// 		log.Printf("Error executing: %s", err)
// 		remote.SetExited(254)
// 		return
// 	}

// 	var exitStatus int

// 	if remote.Stdin != nil {
// 		go func() {
// 			io.Copy(stdin, remote.Stdin)
// 			// close stdin to support commands that wait for stdin to be closed before exiting.
// 			stdin.Close()
// 		}()
// 	}

// 	wg.Wait()
// 	err := cmd.Wait()

// 	if exitErr, ok := err.(*exec.ExitError); ok {
// 		exitStatus = 1

// 		// There is no process-independent way to get the REAL
// 		// exit status so we just try to go deeper.
// 		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
// 			exitStatus = status.ExitStatus()
// 		}
// 	}

// 	// Set the exit status which triggers waiters
// 	remote.SetExited(exitStatus)
// }
