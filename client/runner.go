package client

import (
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hashicorp/packer/packer"
)

type RunParams struct {
	VMName         string
	Volume    string
	Command        []string
	Stdin          io.Reader
	Stdout, Stderr io.Writer
	Debug          bool
	User           string
}

type Runner struct {
	params  RunParams
	cmd     *exec.Cmd
	started time.Time
}

func NewRunner(params RunParams) *Runner {
	args := []string{}

	if params.Debug {
		args = append(args, "--log-level", "debug")
	}

	if params.Stdout == nil {
		params.Stdout = os.Stdout
	}

	if params.Stderr == nil {
		params.Stderr = os.Stderr
	}

	args = append(args, "run")

	if params.Volume != "" {
		args = append(args, "-v", params.Volume)
	} else {
		args = append(args, "-n")
	}

	args = append(args, params.VMName)
	args = append(args, params.Command...)

	cmd := exec.Command("anka", args...)
	cmd.Stdin = params.Stdin
	cmd.Stdout = params.Stdout
	cmd.Stderr = params.Stderr

	return &Runner{
		params: params,
		cmd:    cmd,
	}
}

func (r *Runner) Start() error {
	log.Printf("Starting command: %s", strings.Join(r.cmd.Args, " "))
	r.started = time.Now()
	return r.cmd.Start()
}

func (r *Runner) Wait() (error, int) {
	err := r.cmd.Wait()
	log.Printf("Command finished in %s with %v", time.Now().Sub(r.started), err)
	return err, getExitCode(err)
}

// GetExitCode extracts an exit code from an error where the platform supports it,
// otherwise returns 0 for no error and 1 for an error
func getExitCode(err error) int {
	if err == nil {
		return 0
	}
 	if eerr, ok := err.(*exec.ExitError); ok {
		code := eerr.ExitCode()
		if code == 125 {
			code = packer.CmdDisconnect
		}
		return code
	}

	return 1
}
