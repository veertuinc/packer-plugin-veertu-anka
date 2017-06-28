package client

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-cmd/cmd"
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
	params     RunParams
	cmd        *cmd.Cmd
	started    time.Time
	statusChan <-chan cmd.Status
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

	log.Printf("%#v", args)

	return &Runner{
		params: params,
		cmd:    cmd.NewCmd("anka", args...),
	}
}

func (r *Runner) Start() {
	log.Printf("Starting command: %s", strings.Join(r.cmd.Args, " "))
	r.started = time.Now()
	r.statusChan = r.cmd.Start()

	ticker := time.NewTicker(time.Millisecond * 100)

	go func() {
		var stdoutN, stderrN int
		for range ticker.C {
			status := r.cmd.Status()

			// relay stdout to provided writer
			for _, line := range status.Stdout[stdoutN:] {
				fmt.Fprintf(r.params.Stdout, "%s\n", line)
			}
			stdoutN = len(status.Stdout)

			// relay stderr to provided writer
			for _, line := range status.Stderr[stderrN:] {
				fmt.Fprintf(r.params.Stderr, "%s\n", line)
			}
			stderrN = len(status.Stderr)

			// stop when done
			if status.Complete {
				ticker.Stop()
			}
		}
	}()
}

func (r *Runner) Wait() (error, int) {
	status := <-r.statusChan

	log.Printf("Command finished in %s with %d", time.Now().Sub(r.started), status.Exit)
	return status.Error, status.Exit
}
