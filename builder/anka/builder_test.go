package anka

import (
	"testing"
)

func testConfig() map[string]interface{} {
	return map[string]interface{}{
		"type":          "anka",
		"installer_app": "/Applications/Install macOS Sierra.app/",
		"disk_size":     25,
	}
}

func TestPrepare(t *testing.T) {
	var b Builder

	c := testConfig()

	if _, _, err := b.Prepare(c); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}
