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
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/packer"
)

var ankaExecutable = "anka"

const ankaDebugFIFODrainTimeout = 1 * time.Second

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

// FormatShowResponse builds a Packer log line from parsed ShowResponse fields.
// Empty string fields and zero cpu_cores/hard_drive are omitted. Values are never truncated.
func FormatShowResponse(contextLabel string, show ShowResponse) string {
	parts := []string{fmt.Sprintf("anka show (%s):", contextLabel)}
	appendKV := func(key, value string) {
		if value == "" {
			return
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	appendKV("name", show.Name)
	appendKV("uuid", show.UUID)
	appendKV("status", show.Status)
	if show.VCPUCores != 0 {
		appendKV("cpu_cores", strconv.Itoa(show.VCPUCores))
	}
	appendKV("ram", show.RAM)
	if show.HardDrive != 0 {
		appendKV("hard_drive", strconv.FormatUint(show.HardDrive, 10))
	}
	appendKV("image_id", show.ImageID)
	appendKV("version", show.Version)
	return strings.Join(parts, " ")
}

// LogShowResponse writes FormatShowResponse to the Packer UI.
func LogShowResponse(ui packer.Ui, contextLabel string, show ShowResponse) {
	ui.Say(FormatShowResponse(contextLabel, show))
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
	time.Sleep(1 * time.Second)
	if err := process.Signal(syscall.SIGUSR2); err != nil {
		log.Printf("failed to send SIGUSR2 for registry progress: %v", err)
	}
}

func ankaProcessArgs(args ...string) []string {
	return append([]string{"--machine-readable"}, args...)
}

func liveAnkaDebugEnabled() bool {
	return strings.EqualFold(os.Getenv("ANKA_LOG_LEVEL"), "debug")
}

func ankaChildEnvironment(hostEnviron []string, logFIFOPath string) []string {
	var env []string
	for _, item := range hostEnviron {
		pair := strings.SplitN(item, "=", 2)
		key := pair[0]
		if key == "ANKA_LOG_FILE" || key == "ANKA_LOG_LEVEL" {
			continue
		}
		if strings.HasPrefix(key, "ANKA_") || strings.HasPrefix(key, "PATH") {
			value := ""
			if len(pair) > 1 {
				value = pair[1]
			}
			env = append(env, key+"="+value)
		}
	}
	return append(env, "ANKA_LOG_FILE="+logFIFOPath, "ANKA_LOG_LEVEL=debug")
}

type ankaDebugFIFO struct {
	path   string
	reader *os.File
	keeper *os.File
}

func createAnkaDebugFIFO() (*ankaDebugFIFO, error) {
	tempFile, err := os.CreateTemp(os.TempDir(), "packer-anka-debug-*.fifo")
	if err != nil {
		return nil, err
	}
	fifoPath := tempFile.Name()
	_ = tempFile.Close()
	if err := os.Remove(fifoPath); err != nil {
		return nil, err
	}
	if err := syscall.Mkfifo(fifoPath, 0600); err != nil {
		return nil, err
	}
	// readFD must stay non-blocking: os.File then uses the runtime poller, so
	// Close interrupts a Read that is waiting on a silent writer.
	readFD, err := syscall.Open(fifoPath, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		_ = os.Remove(fifoPath)
		return nil, err
	}
	writeFD, err := syscall.Open(fifoPath, syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		_ = syscall.Close(readFD)
		_ = os.Remove(fifoPath)
		return nil, err
	}
	return &ankaDebugFIFO{
		path:   fifoPath,
		reader: os.NewFile(uintptr(readFD), fifoPath),
		keeper: os.NewFile(uintptr(writeFD), fifoPath),
	}, nil
}

func readAnkaDebugFIFO(fifo *ankaDebugFIFO, live bool, outputStreamer chan string, debugLines *[]string, done chan struct{}) {
	defer close(done)
	if fifo == nil || fifo.reader == nil {
		return
	}
	scanner := bufio.NewScanner(fifo.reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		*debugLines = append(*debugLines, line)
		if !live {
			continue
		}
		if outputStreamer != nil {
			outputStreamer <- line
		} else {
			log.Printf("%s", line)
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, os.ErrClosed) {
		log.Printf("error reading anka debug fifo: %v", err)
	}
}

// finishAnkaDebugFIFO lets the reader drain lines Anka wrote before exit, then
// force-closes it: processes Anka leaves running (e.g. the VM hypervisor after
// `anka start`) keep the write end open, so EOF may never arrive.
func finishAnkaDebugFIFO(fifo *ankaDebugFIFO, readerDone <-chan struct{}) {
	if fifo != nil && fifo.keeper != nil {
		_ = fifo.keeper.Close()
	}
	if readerDone != nil {
		select {
		case <-readerDone:
		case <-time.After(ankaDebugFIFODrainTimeout):
		}
	}
	if fifo != nil && fifo.reader != nil {
		_ = fifo.reader.Close()
	}
	if readerDone != nil {
		<-readerDone
	}
}

func attachDebugLines(err error, debugLines []string, outputStreamer chan string, live bool) error {
	if err == nil || len(debugLines) == 0 {
		return err
	}
	if live {
		return err
	}
	if outputStreamer != nil {
		for _, line := range debugLines {
			outputStreamer <- line
		}
		return err
	}
	if machineReadableError, ok := err.(MachineReadableError); ok {
		machineReadableError.DebugLines = append([]string(nil), debugLines...)
		return machineReadableError
	}
	return fmt.Errorf("%w\n%s", err, strings.Join(debugLines, "\n"))
}

func runAnkaProcess(outputStreamer chan string, sendProgressSignal bool, args ...string) (MachineReadableOutput, error) {
	cmdArgs := ankaProcessArgs(args...)

	log.Printf("Executing anka %s", strings.Join(cmdArgs, " "))

	fifo, err := createAnkaDebugFIFO()
	if err != nil {
		return MachineReadableOutput{}, err
	}
	defer os.Remove(fifo.path)

	liveDebug := liveAnkaDebugEnabled()
	var debugLines []string
	readerDone := make(chan struct{})
	go readAnkaDebugFIFO(fifo, liveDebug, outputStreamer, &debugLines, readerDone)

	cmd := exec.Command(ankaExecutable, cmdArgs...)
	cmd.Env = ankaChildEnvironment(os.Environ(), fifo.path)

	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		finishAnkaDebugFIFO(fifo, readerDone)
		return MachineReadableOutput{}, err
	}

	if outputStreamer == nil {
		cmd.Stderr = io.Discard
	} else {
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			finishAnkaDebugFIFO(fifo, readerDone)
			return MachineReadableOutput{}, err
		}
		go streamLinesToChannel(stderrPipe, outputStreamer)
	}

	err = cmd.Start()
	if err != nil {
		finishAnkaDebugFIFO(fifo, readerDone)
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
			_ = cmd.Wait()
			finishAnkaDebugFIFO(fifo, readerDone)
			return MachineReadableOutput{}, attachDebugLines(errors.New("missing machine readable output"), debugLines, outputStreamer, liveDebug)
		}
		finalOutput = lastNonProgressLine
	} else {
		_, ok := scannerErr.(customErr)
		if !ok {
			_ = cmd.Wait()
			finishAnkaDebugFIFO(fifo, readerDone)
			return MachineReadableOutput{}, attachDebugLines(scannerErr, debugLines, outputStreamer, liveDebug)
		}
		finalOutput = scannerErr.Error()
	}

	parsed, err := parseOutput([]byte(finalOutput))
	waitErr := cmd.Wait()
	finishAnkaDebugFIFO(fifo, readerDone)
	if err != nil {
		return MachineReadableOutput{}, attachDebugLines(err, debugLines, outputStreamer, liveDebug)
	}

	err = parsed.GetError()
	if err != nil {
		return MachineReadableOutput{}, attachDebugLines(err, debugLines, outputStreamer, liveDebug)
	}
	if waitErr != nil {
		return MachineReadableOutput{}, attachDebugLines(waitErr, debugLines, outputStreamer, liveDebug)
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
