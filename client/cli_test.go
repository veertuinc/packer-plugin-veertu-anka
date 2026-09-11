package client

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

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
