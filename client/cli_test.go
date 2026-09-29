package client

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestMain(m *testing.M) {
	helper := os.Getenv("ANKA_PACKER_ANKA_HELPER")
	if helper == "" {
		for _, arg := range os.Args[1:] {
			if arg == "--machine-readable" {
				os.Exit(4)
			}
		}
	}
	switch helper {
	case "success-with-debug":
		writeHelperFIFO("debug-from-fifo")
		fmt.Print(`{"status":"OK","body":"ok-body"}`)
		os.Exit(0)
	case "fail-with-debug":
		writeHelperFIFO("boom-debug")
		fmt.Print(`{"status":"FAIL","message":"nope","code":1}`)
		os.Exit(0)
	case "progress-stderr":
		fmt.Fprintln(os.Stderr, `{"p":0.62}`)
		fmt.Print(`{"status":"OK","body":"ok"}`)
		os.Exit(0)
	case "success-with-lingering-fifo-writer":
		startLingeringFIFOWriter()
		fmt.Print(`{"status":"OK","body":"ok"}`)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func writeHelperFIFO(line string) {
	logPath := os.Getenv("ANKA_LOG_FILE")
	fifo, err := os.OpenFile(logPath, os.O_WRONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "helper fifo open: %v\n", err)
		os.Exit(3)
	}
	if _, err := fmt.Fprintln(fifo, line); err != nil {
		fmt.Fprintf(os.Stderr, "helper fifo write: %v\n", err)
		os.Exit(3)
	}
	_ = fifo.Close()
}

// startLingeringFIFOWriter mimics a VM hypervisor that outlives `anka start`
// and keeps ANKA_LOG_FILE open without writing to it.
func startLingeringFIFOWriter() {
	fifo, err := os.OpenFile(os.Getenv("ANKA_LOG_FILE"), os.O_WRONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "helper fifo open: %v\n", err)
		os.Exit(3)
	}
	fmt.Fprintln(fifo, "debug-before-lingering")
	lingeringWriter := exec.Command("/bin/sleep", "30")
	lingeringWriter.ExtraFiles = []*os.File{fifo}
	if err := lingeringWriter.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "helper lingering writer start: %v\n", err)
		os.Exit(3)
	}
	lingeringWriterPID := strconv.Itoa(lingeringWriter.Process.Pid)
	if err := os.WriteFile(os.Getenv("ANKA_PACKER_LINGERING_WRITER_PID_FILE"), []byte(lingeringWriterPID), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "helper lingering writer pid: %v\n", err)
		os.Exit(3)
	}
}

func killLingeringFIFOWriter(lingeringWriterPIDFile string) {
	lingeringWriterPIDBytes, err := os.ReadFile(lingeringWriterPIDFile)
	if err != nil {
		return
	}
	lingeringWriterPID, err := strconv.Atoi(strings.TrimSpace(string(lingeringWriterPIDBytes)))
	if err != nil {
		return
	}
	_ = syscall.Kill(lingeringWriterPID, syscall.SIGKILL)
}

func TestAnkaProcessArgsNeverIncludeDebug(t *testing.T) {
	args := ankaProcessArgs("show", "vm")
	assert.Equal(t, "--machine-readable", args[0])
	assert.DeepEqual(t, []string{"--machine-readable", "show", "vm"}, args)
	for _, arg := range args {
		assert.Assert(t, arg != "--debug")
	}
}

func TestAnkaChildEnvironmentOverridesLogFileAndLevel(t *testing.T) {
	env := ankaChildEnvironment([]string{
		"ANKA_LOG_FILE=/old/anka.log",
		"ANKA_LOG_LEVEL=info",
		"ANKA_DEFAULT_USER=anka",
		"PATH=/bin",
		"HOME=/tmp",
	}, "/tmp/anka-debug.fifo")

	assert.Assert(t, hasEnv(env, "ANKA_LOG_FILE", "/tmp/anka-debug.fifo"))
	assert.Assert(t, hasEnv(env, "ANKA_LOG_LEVEL", "debug"))
	assert.Assert(t, hasEnv(env, "ANKA_DEFAULT_USER", "anka"))
	assert.Assert(t, hasEnv(env, "PATH", "/bin"))
	assert.Assert(t, !hasEnvKey(env, "HOME"))
	assert.Assert(t, !hasEnv(env, "ANKA_LOG_FILE", "/old/anka.log"))
}

func TestLiveAnkaDebugEnabled(t *testing.T) {
	t.Setenv("ANKA_LOG_LEVEL", "debug")
	assert.Equal(t, true, liveAnkaDebugEnabled())

	t.Setenv("ANKA_LOG_LEVEL", "DEBUG")
	assert.Equal(t, true, liveAnkaDebugEnabled())

	t.Setenv("ANKA_LOG_LEVEL", "info")
	assert.Equal(t, false, liveAnkaDebugEnabled())

	t.Setenv("ANKA_LOG_LEVEL", "")
	assert.Equal(t, false, liveAnkaDebugEnabled())
}

func TestRunAnkaProcessSuccessDropsFIFODebug(t *testing.T) {
	restore := useTestAnkaHelper(t, "success-with-debug")
	defer restore()
	t.Setenv("ANKA_LOG_LEVEL", "")

	streamer := make(chan string, 8)
	output, err := runAnkaProcess(streamer, false, "show", "vm")
	assert.NilError(t, err)
	assert.Equal(t, "OK", output.Status)
	assert.Equal(t, `"ok-body"`, string(output.Body))
	assert.Equal(t, 0, len(collectStreamer(streamer)))
}

func TestRunAnkaProcessFailureIncludesFIFODebug(t *testing.T) {
	restore := useTestAnkaHelper(t, "fail-with-debug")
	defer restore()
	t.Setenv("ANKA_LOG_LEVEL", "")

	_, err := runAnkaProcess(nil, false, "show", "vm")
	assert.Assert(t, err != nil)
	merr, ok := err.(MachineReadableError)
	assert.Assert(t, ok)
	assert.Equal(t, "nope", merr.Message)
	assert.Assert(t, strings.Contains(err.Error(), "boom-debug"))
}

func TestRunAnkaProcessFailureStreamsFIFOWhenLiveOff(t *testing.T) {
	restore := useTestAnkaHelper(t, "fail-with-debug")
	defer restore()
	t.Setenv("ANKA_LOG_LEVEL", "")

	streamer := make(chan string, 8)
	_, err := runAnkaProcess(streamer, false, "show", "vm")
	assert.Assert(t, err != nil)
	merr, ok := err.(MachineReadableError)
	assert.Assert(t, ok)
	assert.Equal(t, "nope", merr.Message)
	assert.Equal(t, 0, len(merr.DebugLines))
	lines := collectStreamer(streamer)
	assert.Assert(t, hasLine(lines, "boom-debug"))
}

func TestRunAnkaProcessLiveDebugStreamsFIFO(t *testing.T) {
	restore := useTestAnkaHelper(t, "success-with-debug")
	defer restore()
	t.Setenv("ANKA_LOG_LEVEL", "debug")

	streamer := make(chan string, 8)
	output, err := runAnkaProcess(streamer, false, "show", "vm")
	assert.NilError(t, err)
	assert.Equal(t, "OK", output.Status)
	lines := collectStreamer(streamer)
	assert.Assert(t, hasLine(lines, "debug-from-fifo"))
}

func TestRunAnkaProcessProgressStaysOnStderr(t *testing.T) {
	restore := useTestAnkaHelper(t, "progress-stderr")
	defer restore()
	t.Setenv("ANKA_LOG_LEVEL", "")

	streamer := make(chan string, 8)
	output, err := runAnkaProcess(streamer, false, "registry", "pull", "vm")
	assert.NilError(t, err)
	assert.Equal(t, "OK", output.Status)
	lines := collectStreamer(streamer)
	assert.DeepEqual(t, []string{"62%"}, lines)
}

func TestRunAnkaProcessReturnsWhenChildKeepsFIFOOpen(t *testing.T) {
	restore := useTestAnkaHelper(t, "success-with-lingering-fifo-writer")
	defer restore()
	t.Setenv("ANKA_LOG_LEVEL", "")
	lingeringWriterPIDFile := filepath.Join(t.TempDir(), "lingering-writer.pid")
	t.Setenv("ANKA_PACKER_LINGERING_WRITER_PID_FILE", lingeringWriterPIDFile)
	defer killLingeringFIFOWriter(lingeringWriterPIDFile)

	runAnkaProcessErr := make(chan error, 1)
	go func() {
		_, err := runAnkaProcess(nil, false, "start", "vm")
		runAnkaProcessErr <- err
	}()

	select {
	case err := <-runAnkaProcessErr:
		assert.NilError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("runAnkaProcess did not return while a child process kept the debug FIFO open")
	}
}

func useTestAnkaHelper(t *testing.T, helper string) func() {
	t.Helper()
	t.Setenv("ANKA_PACKER_ANKA_HELPER", helper)
	previous := ankaExecutable
	ankaExecutable = os.Args[0]
	return func() {
		ankaExecutable = previous
	}
}

func collectStreamer(streamer chan string) []string {
	close(streamer)
	var lines []string
	for line := range streamer {
		lines = append(lines, line)
	}
	return lines
}

func hasLine(lines []string, want string) bool {
	for _, line := range lines {
		if line == want {
			return true
		}
	}
	return false
}

func hasEnv(env []string, key, value string) bool {
	want := key + "=" + value
	for _, item := range env {
		if item == want {
			return true
		}
	}
	return false
}

func hasEnvKey(env []string, key string) bool {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return true
		}
	}
	return false
}

func TestFormatProgressJSONLine(t *testing.T) {
	t.Run("converts fraction to percent", func(t *testing.T) {
		formatted, isProgress := formatProgressJSONLine(`{"p":0.62}`)
		assert.Equal(t, true, isProgress)
		assert.Equal(t, "62%", formatted)
	})

	t.Run("rounds half up", func(t *testing.T) {
		formatted, isProgress := formatProgressJSONLine(`{"p":0.666}`)
		assert.Equal(t, true, isProgress)
		assert.Equal(t, "67%", formatted)
	})

	t.Run("formats zero and complete", func(t *testing.T) {
		formatted, isProgress := formatProgressJSONLine(`{"p":0}`)
		assert.Equal(t, true, isProgress)
		assert.Equal(t, "0%", formatted)

		formatted, isProgress = formatProgressJSONLine(`{"p":1}`)
		assert.Equal(t, true, isProgress)
		assert.Equal(t, "100%", formatted)
	})

	t.Run("ignores machine readable status json", func(t *testing.T) {
		_, isProgress := formatProgressJSONLine(`{"status":"OK","body":null}`)
		assert.Equal(t, false, isProgress)
	})

	t.Run("ignores non json and lines without p", func(t *testing.T) {
		_, isProgress := formatProgressJSONLine(`Installing macOS...`)
		assert.Equal(t, false, isProgress)

		_, isProgress = formatProgressJSONLine(`{"message":"hello"}`)
		assert.Equal(t, false, isProgress)
	})
}

func TestSendProgressSignalToProcess(t *testing.T) {
	if os.Getenv("ANKA_PACKER_PROGRESS_HELPER") == "1" {
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, syscall.SIGUSR2)
		select {
		case <-signalCh:
			os.Exit(0)
		case <-time.After(5 * time.Second):
			os.Exit(2)
		}
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestSendProgressSignalToProcess$")
	cmd.Env = append(os.Environ(), "ANKA_PACKER_PROGRESS_HELPER=1")
	err := cmd.Start()
	assert.NilError(t, err)

	sendProgressSignalToProcess(cmd.Process)
	err = cmd.Wait()
	assert.NilError(t, err)
}
