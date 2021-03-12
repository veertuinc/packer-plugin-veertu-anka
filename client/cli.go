package client

import (
	"bufio"
	"errors"
	"log"
	"os/exec"
	"strings"
)

func runCommand(args ...string) (MachineReadableOutput, error) {
	return runCommandStreamer(nil, args...)
}

func runCommandStreamer(outputStreamer chan string, args ...string) (MachineReadableOutput, error) {
	if outputStreamer != nil {
		args = append([]string{"--debug"}, args...)
	}

	cmdArgs := append([]string{"--machine-readable"}, args...)

	log.Printf("Executing anka %s", strings.Join(cmdArgs, " "))

	cmd := exec.Command("anka", cmdArgs...)

	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return MachineReadableOutput{}, err
	}

	if outputStreamer == nil {
		cmd.Stderr = cmd.Stdout
	}

	if err = cmd.Start(); err != nil {
		return MachineReadableOutput{}, err
	}

	outScanner := bufio.NewScanner(outPipe)
	outScanner.Split(customSplit)

	for outScanner.Scan() {
		out := outScanner.Text()

		if outputStreamer != nil {
			outputStreamer <- out
		}
	}

	scannerErr := outScanner.Err()
	if scannerErr == nil {
		return MachineReadableOutput{}, errors.New("missing machine readable output")
	}
	if _, ok := scannerErr.(customErr); !ok {
		return MachineReadableOutput{}, err
	}

	finalOutput := scannerErr.Error()

	parsed, err := parseOutput([]byte(finalOutput))
	if err != nil {
		return MachineReadableOutput{}, err
	}

	cmd.Wait()

	if err = parsed.GetError(); err != nil {
		return MachineReadableOutput{}, err
	}

	return parsed, nil
}

func runRegistryCommand(registryParams RegistryParams, args ...string) (MachineReadableOutput, error) {
	cmdArgs := []string{"registry"}

	if registryParams.RegistryName != "" {
		cmdArgs = append(cmdArgs, "--remote", registryParams.RegistryName)
	}

	if registryParams.RegistryURL != "" {
		cmdArgs = append(cmdArgs, "--registry-path", registryParams.RegistryURL)
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

	return runCommand(cmdArgs...)
}
