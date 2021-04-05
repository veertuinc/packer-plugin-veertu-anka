package anka

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-builder-veertu-anka/client"
)

// Communicator initializes what is shared between anka and packer
type Communicator struct {
	Config  *Config
	Client  client.Client
	HostDir string
	VMDir   string
	VMName  string
}

// Start runs the actual anka commands
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
		exitCode, err := runner.Wait()
		if err != nil {
			log.Printf("Runner exited with error: %v", err)
		}

		remote.SetExited(exitCode)
	}()

	return nil
}

// Upload uploads the source file to the destination
func (c *Communicator) Upload(dst string, src io.Reader, fi *os.FileInfo) error {
	log.Printf("Uploading file to VM: %s", dst)

	c.configureAnkaCP()

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
		_ = tempfile.Chmod((*fi).Mode())
	}

	if c.Config.UseAnkaCP {
		err = c.Client.Copy(client.CopyParams{
			Src: tempfile.Name(),
			Dst: c.VMName + ":" + dst,
		})
	} else {
		_, err = c.Client.Run(client.RunParams{
			VMName:  c.VMName,
			Command: []string{"cp", path.Base(tempfile.Name()), dst},
			Volume:  c.HostDir,
		})
	}

	log.Printf("Copied %d bytes from %s to %s", w, tempfile.Name(), dst)

	return err
}

// UploadDir uploads the source directory to the destination
func (c *Communicator) UploadDir(dst string, src string, exclude []string) error {
	c.configureAnkaCP()

	if !c.Config.UseAnkaCP {
		td, err := ioutil.TempDir(c.HostDir, "dirupload")
		if err != nil {
			return err
		}

		defer os.RemoveAll(td)

		walkFn := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relpath, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}

			hostpath := filepath.Join(td, relpath)

			if info.IsDir() {
				return os.MkdirAll(hostpath, info.Mode())
			}

			if info.Mode()&os.ModeSymlink == os.ModeSymlink {
				dest, err := os.Readlink(path)
				if err != nil {
					return err
				}

				return os.Symlink(dest, hostpath)
			}

			src, err := os.Open(path)
			if err != nil {
				return err
			}

			defer src.Close()

			dst, err := os.Create(hostpath)
			if err != nil {
				return err
			}

			defer dst.Close()

			log.Printf("Copying %s to %s", src.Name(), dst.Name())
			if _, err := io.Copy(dst, src); err != nil {
				return err
			}

			si, err := src.Stat()
			if err != nil {
				return err
			}

			return dst.Chmod(si.Mode())
		}

		err = filepath.Walk(src, walkFn)
		if err != nil {
			return err
		}

		containerDst := dst
		if src[len(src)-1] != '/' {
			containerDst = filepath.Join(dst, filepath.Base(src))
		}

		log.Printf("from %#v to %#v", td, containerDst)

		command := fmt.Sprintf("set -e; mkdir -p %s; command cp -R %s/* %s",
			containerDst, filepath.Base(td), containerDst,
		)

		_, err = c.Client.Run(client.RunParams{
			VMName:  c.VMName,
			Command: []string{"bash", "-c", command},
			Volume:  c.HostDir,
		})

		return err
	}

	return c.Client.Copy(client.CopyParams{
		Src: src,
		Dst: c.VMName + ":" + dst,
	})
}

// Download copies the file from the source to the destination
func (c *Communicator) Download(src string, dst io.Writer) error {
	log.Printf("Downloading file from VM: %s", src)

	c.configureAnkaCP()

	tempfile, err := ioutil.TempFile(c.HostDir, "download")
	if err != nil {
		return err
	}

	defer os.Remove(tempfile.Name())
	defer tempfile.Close()

	if c.Config.UseAnkaCP {
		err := c.Client.Copy(client.CopyParams{
			Src: c.VMName + ":" + src,
			Dst: tempfile.Name(),
		})
		if err != nil {
			return err
		}
	}

	log.Printf("Copying from %s to writer", tempfile.Name())

	w, err := io.Copy(dst, tempfile)
	if err != nil {
		return err
	}

	if !c.Config.UseAnkaCP {
		_, err := c.Client.Run(client.RunParams{
			VMName:  c.VMName,
			Command: []string{"cp", src, "./" + path.Base(tempfile.Name())},
			Volume:  c.HostDir,
		})
		if err != nil {
			return err
		}
	}

	log.Printf("Copied %d bytes", w)

	return nil
}

// DownloadDir copies the source directory to the destination
func (c *Communicator) DownloadDir(src string, dst string, exclude []string) error {
	c.configureAnkaCP()

	if !c.Config.UseAnkaCP {
		return errors.New("communicator.DownloadDir isn't implemented")
	}

	return c.Client.Copy(client.CopyParams{
		Src: c.VMName + ":" + src,
		Dst: dst,
	})
}

func (c *Communicator) configureAnkaCP() {
	if !c.Config.UseAnkaCP {
		errFindingFUSE := c.findFUSE()
		if errFindingFUSE != nil {
			c.Config.UseAnkaCP = true
		}
	}
}

func (c *Communicator) findFUSE() error {
	_, notFound := c.Client.Run(client.RunParams{
		VMName:  c.VMName,
		Command: []string{"kextstat | grep \"com.veertu.filesystems.vtufs\" &>/dev/null"},
	})

	return notFound
}
