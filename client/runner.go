package client

import (
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

type RunParams struct {
	VMName         string
	VolumesFrom    string
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
		args = append(args, "--debug")
	}

	if params.Stdout == nil {
		params.Stdout = os.Stdout
	}

	if params.Stderr == nil {
		params.Stderr = os.Stderr
	}

	args = append(args, "run")

	if params.User != "" {
		args = append(args, "--user", params.User)
	}

	if params.VolumesFrom != "" {
		args = append(args, "--volumes-from", params.VolumesFrom)
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
	switch cause := errors.Cause(err).(type) {
	case *exec.ExitError:
		// The program has exited with an exit code != 0
		// There is no platform independent way to retrieve
		// the exit code, but the following will work on Unix/macOS
		if status, ok := cause.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	return 1
}
