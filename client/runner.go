package client

import (
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type RunParams struct {
	VMName         string
	VolumesFrom    string
	Command        []string
	Stdin          io.Reader
	Stdout, Stderr io.Writer
	Debug          bool
}

type Runner struct {
	params         RunParams
	cmd            *exec.Cmd
	started        time.Time
	stdin          io.WriteCloser
	stdout, stderr io.ReadCloser
}

func NewRunner(params RunParams) *Runner {
	args := []string{}

	if params.Debug {
		args = append(args, "--debug")
	}

	if params.Stdin == nil {
		params.Stdin = os.Stdin
	}

	if params.Stdout == nil {
		params.Stdout = os.Stdout
	}

	if params.Stderr == nil {
		params.Stderr = os.Stderr
	}

	args = append(args, "run")

	if params.VolumesFrom != "" {
		args = append(args, "--volumes-from", params.VolumesFrom)
	}

	args = append(args, params.VMName)
	args = append(args, params.Command...)

	return &Runner{
		params: params,
		cmd:    exec.Command("anka", args...),
	}
}

func (r *Runner) Start() error {
	var err error

	r.stdin, err = r.cmd.StdinPipe()
	if err != nil {
		return err
	}

	r.stderr, err = r.cmd.StderrPipe()
	if err != nil {
		return err
	}

	r.stdout, err = r.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	log.Printf("Starting command: %s", strings.Join(r.cmd.Args, " "))
	r.started = time.Now()
	if err := r.cmd.Start(); err != nil {
		return err
	}

	return r.readStreams()
}

func (r *Runner) readStreams() error {
	repeat := func(w io.Writer, rd io.ReadCloser, note string) {
		log.Printf("Copying %s from %v to %v", note, rd, w)
		n, _ := io.Copy(w, rd)
		log.Printf("Copied %d bytes from %s", n, note)
		log.Printf("Closing %s", note)
		rd.Close()
	}

	// for now just close stdin
	if r.stdin != nil {
		r.stdin.Close()
	}

	if r.stdout != nil {
		go repeat(r.params.Stdout, r.stdout, "stdout")
	}

	if r.stderr != nil {
		go repeat(r.params.Stderr, r.stderr, "stderr")
	}

	return nil
}

func (r *Runner) Wait() error {

	log.Printf("Waiting for command to finish")
	err := r.cmd.Wait()

	log.Printf("Command finished in %s", time.Now().Sub(r.started))
	if err != nil {
		log.Printf("Command failed: %v", err)
	}
	return err
}

func (r *Runner) ExitStatus() int {
	err := r.Wait()
	if err == nil {
		return 0
	}

	exitStatus := 1
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitStatus = 1

		// There is no process-independent way to get the REAL
		// exit status so we just try to go deeper.
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			exitStatus = status.ExitStatus()
		}
	}

	log.Printf("Command exited with %d", exitStatus)
	return exitStatus
}
