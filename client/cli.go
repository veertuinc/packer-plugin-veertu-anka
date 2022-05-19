package client

import (
	"bufio"
	"errors"
	"log"
	"os"
	"os/exec"
	"strings"
)

func runAnkaCommand(args ...string) (MachineReadableOutput, error) {
	return runCommandStreamer(nil, args...)
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
	}

	err = cmd.Start()
	if err != nil {
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

	if registryParams.HostArch == "arm64" {
		// Anka 3 only has --remote and accepts the URL and also the repo name
		if registryParams.RegistryName != "" {
			cmdArgs = append(cmdArgs, "--remote", registryParams.RegistryName)
		} else if registryParams.RegistryURL != "" {
			cmdArgs = append(cmdArgs, "--remote", registryParams.RegistryURL)
		}
	} else { // Anka 2 / Intel
		if registryParams.RegistryName != "" {
			cmdArgs = append(cmdArgs, "--remote", registryParams.RegistryName)
		}
		if registryParams.RegistryURL != "" {
			cmdArgs = append(cmdArgs, "--registry-path", registryParams.RegistryURL)
		}
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
