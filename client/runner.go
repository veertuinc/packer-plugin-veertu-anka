package client

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
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
	wg             sync.WaitGroup
	params         RunParams
	cmd            *exec.Cmd
	started        time.Time
	stdout, stderr io.ReadCloser
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

	r.stderr, err = r.cmd.StderrPipe()
	if err != nil {
		return err
	}

	r.stdout, err = r.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGCHLD,
	)
	go func() {
		<-sigc
		log.Printf("Got SIGCHLD from child")
	}()

	log.Printf("Starting command: %s", strings.Join(r.cmd.Args, " "))
	r.started = time.Now()
	if err := r.cmd.Start(); err != nil {
		return err
	}

	log.Printf("Spawned process pid %d", r.cmd.Process.Pid)
	return r.readStreams()
}

func (r *Runner) readStreams() error {
	if r.params.Stdout != nil {
		r.wg.Add(1)
		go func() {
			scanner := bufio.NewScanner(r.stdout)
			for scanner.Scan() {
				line := scanner.Text()
				log.Printf("[stdout] %s", line)
			}
			log.Printf("Finished reading stdout")
			r.wg.Done()
		}()
	}

	if r.params.Stderr != nil {
		r.wg.Add(1)
		go func() {
			scanner := bufio.NewScanner(r.stderr)
			for scanner.Scan() {
				line := scanner.Text()
				log.Printf("[stderr] %s", line)
			}
			log.Printf("Finished reading stderr")
			r.wg.Done()
		}()
	}

	return nil
}

func (r *Runner) Wait() error {
	r.wg.Wait()
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
