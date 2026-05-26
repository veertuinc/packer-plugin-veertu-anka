package anka

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestGuestAPFSResizeContainerShellCommand(t *testing.T) {
	assert.Assert(t, strings.Contains(guestAPFSResizeContainerShellCommand, "disk0s2"))
	assert.Assert(t, strings.Contains(guestAPFSResizeContainerShellCommand, "/APFS Container:/"))
	assert.Assert(t, strings.Contains(guestAPFSResizeContainerShellCommand, "/APFS Physical Store:/"))
	assert.Assert(t, strings.Contains(guestAPFSResizeContainerShellCommand, "diskutil apfs resizeContainer"))
}
