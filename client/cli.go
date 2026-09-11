package client

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/packer"
)

// StreamOutputToUI copies command progress lines to the Packer UI.
// Call finish after the command returns so the last lines print.
func StreamOutputToUI(ui packer.Ui) (outputStream chan string, finish func()) {
	outputStream = make(chan string)
	outputStreamDone := make(chan struct{})
	go func() {
		defer close(outputStreamDone)
		for msg := range outputStream {
			ui.Say(msg)
		}
	}()
	finish = func() {
		close(outputStream)
		<-outputStreamDone
	}
	return outputStream, finish
}

func runAnkaCommand(args ...string) (MachineReadableOutput, error) {
	return runCommandStreamer(nil, args...)
}

func runAnkaCommandWithProgress(outputStreamer chan string, args ...string) (MachineReadableOutput, error) {
	return runAnkaProcess(outputStreamer, true, args...)
}

func runCommandStreamer(outputStreamer chan string, args ...string) (MachineReadableOutput, error) {
	return runAnkaProcess(outputStreamer, false, args...)
}

// streamLinesToChannel reads lines and sends each to the channel.
// Progress JSON from SIGUSR2 ({"p":0.62}) is formatted as a percent.
func streamLinesToChannel(reader io.Reader, outputStreamer chan string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		emitStreamerLine(outputStreamer, strings.TrimSpace(scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		log.Printf("error reading anka command output: %v", err)
	}
}

func emitStreamerLine(outputStreamer chan string, line string) {
	if outputStreamer == nil || line == "" {
		return
	}
	if formattedProgress, isProgress := formatProgressJSONLine(line); isProgress {
		outputStreamer <- formattedProgress
		return
	}
	outputStreamer <- line
}

// formatProgressJSONLine converts Anka SIGUSR2 progress JSON ({"p":0.62}) to "62%".
// Lines that include a machine-readable "status" field are not progress.
func formatProgressJSONLine(line string) (string, bool) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return "", false
	}
	if _, hasStatus := raw["status"]; hasStatus {
		return "", false
	}
	progressRaw, hasProgress := raw["p"]
	if !hasProgress {
		return "", false
	}
	var progressFraction float64
	if err := json.Unmarshal(progressRaw, &progressFraction); err != nil {
		return "", false
	}
	percent := int(progressFraction*100 + 0.5)
	return fmt.Sprintf("%d%%", percent), true
}

// sendProgressSignalToProcess sends SIGUSR2 so Anka writes {"p":...} progress lines.
func sendProgressSignalToProcess(process *os.Process) {
	if process == nil {
		return
	}
	// Give Anka time to install its SIGUSR2 handler. The default action for
	// SIGUSR2 is terminate, so a signal that is too early can kill the push.
	time.Sleep(200 * time.Millisecond)
	if err := process.Signal(syscall.SIGUSR2); err != nil {
		log.Printf("failed to send SIGUSR2 for registry progress: %v", err)
	}
}

func runAnkaProcess(outputStreamer chan string, sendProgressSignal bool, args ...string) (MachineReadableOutput, error) {

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
		go streamLinesToChannel(stderrPipe, outputStreamer)
	}

	err = cmd.Start()
	if err != nil {
		return MachineReadableOutput{}, err
	}

	if sendProgressSignal {
		go sendProgressSignalToProcess(cmd.Process)
	}

	outScanner := bufio.NewScanner(outPipe)
	outScanner.Split(customSplit)

	var lastNonProgressLine string
	for outScanner.Scan() {
		line := strings.TrimSpace(outScanner.Text())
		if line == "" {
			continue
		}
		if formattedProgress, isProgress := formatProgressJSONLine(line); isProgress {
			if outputStreamer != nil {
				outputStreamer <- formattedProgress
			}
			continue
		}
		lastNonProgressLine = line
	}

	scannerErr := outScanner.Err()
	finalOutput := ""
	if scannerErr == nil {
		if lastNonProgressLine == "" {
			return MachineReadableOutput{}, errors.New("missing machine readable output")
		}
		finalOutput = lastNonProgressLine
	} else {
		_, ok := scannerErr.(customErr)
		if !ok {
			return MachineReadableOutput{}, err
		}
		finalOutput = scannerErr.Error()
	}

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

func registryCommandArgs(registryParams RegistryParams, args ...string) []string {
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

	return append(cmdArgs, args...)
}

func runRegistryCommand(registryParams RegistryParams, args ...string) (MachineReadableOutput, error) {
	return runAnkaCommand(registryCommandArgs(registryParams, args...)...)
}

func runRegistryCommandWithProgress(registryParams RegistryParams, outputStreamer chan string, args ...string) (MachineReadableOutput, error) {
	return runAnkaCommandWithProgress(outputStreamer, registryCommandArgs(registryParams, args...)...)
}
