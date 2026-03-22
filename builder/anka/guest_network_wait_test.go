package anka

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestGuestNetworkReadinessShCommand(t *testing.T) {
	cmd := guestNetworkReadinessShCommand()
	assert.Equal(t, 1, len(cmd))
	assert.Assert(t, strings.Contains(cmd[0], guestNetworkProbeTarget))
	assert.Assert(t, strings.Contains(cmd[0], "ping -c 1"))
}
