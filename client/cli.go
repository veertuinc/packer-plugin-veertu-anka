package client

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

func runAnkaCommand(args ...string) (MachineReadableOutput, error) {
	return runCommandStreamer(nil, args...)
}

// streamStderrToChannel reads stderr line-by-line and sends each line to the channel.
// Used to stream anka create progress (e.g. "Installing macOS...") to the Packer UI.
func streamStderrToChannel(stderr io.Reader, outputStreamer chan string) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			outputStreamer <- line
		}
	}
}


func runCommandStreamer(outputStreamer chan string, args ...string) (MachineReadableOutput, error) {

	cmdArgs := append([]string{"--machine-readable"}, args...)

	log.Printf("Executing anka %s", strings.Join(cmdArgs, " "))

	cmd := exec.Command("anka", cmdArgs...)

	for _, e := range os.Environ() { // Ensure that ANKA_ environment variables from the host are available when executing anka commands
		pair := strings.SplitN(e, "=", 2)
		key := pair[0]
		val := pair[1]
		if strings.HasPrefix(key, "ANKA_") || strings.HasPrefix(key, "PATH") {
			cmd.Env = append([]string{key + "=" + val}, cmd.Env...)
		}
	}

	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return MachineReadableOutput{}, err
	}

	if outputStreamer == nil {
		cmd.Stderr = cmd.Stdout
	} else {
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return MachineReadableOutput{}, err
		}
		go streamStderrToChannel(stderrPipe, outputStreamer)
	}

	err = cmd.Start()
	if err != nil {
		return MachineReadableOutput{}, err
	}

	outScanner := bufio.NewScanner(outPipe)
	outScanner.Split(customSplit)

	for outScanner.Scan() {
		// Stdout is JSON (machine-readable); only stderr has human-readable progress
		// Don't send stdout to outputStreamer
	}

	scannerErr := outScanner.Err()
	if scannerErr == nil {
		return MachineReadableOutput{}, errors.New("missing machine readable output")
	}

	_, ok := scannerErr.(customErr)
	if !ok {
		return MachineReadableOutput{}, err
	}

	finalOutput := scannerErr.Error()

	parsed, err := parseOutput([]byte(finalOutput))
	if err != nil {
		return MachineReadableOutput{}, err
	}

	cmd.Wait()

	err = parsed.GetError()
	if err != nil {
		return MachineReadableOutput{}, err
	}

	return parsed, nil
}

func runRegistryCommand(registryParams RegistryParams, args ...string) (MachineReadableOutput, error) {
	cmdArgs := []string{"registry"}

	if registryParams.Remote != "" {
		cmdArgs = append(cmdArgs, "--remote", registryParams.Remote)
	}

	if registryParams.NodeCertPath != "" {
		cmdArgs = append(cmdArgs, "--cert", registryParams.NodeCertPath)
	}

	if registryParams.NodeKeyPath != "" {
		cmdArgs = append(cmdArgs, "--key", registryParams.NodeKeyPath)
	}

	if registryParams.CaRootPath != "" {
		cmdArgs = append(cmdArgs, "--cacert", registryParams.CaRootPath)
	}

	if registryParams.IsInsecure {
		cmdArgs = append(cmdArgs, "--insecure")
	}

	cmdArgs = append(cmdArgs, args...)

	return runAnkaCommand(cmdArgs...)
}
