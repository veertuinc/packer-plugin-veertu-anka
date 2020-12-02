package anka

import (
	"testing"
)

func testConfig() map[string]interface{} {
	return map[string]interface{}{
		"type":          "veertu-anka",
		"installer_app": "/Applications/Install macOS Catalina.app",
		"disk_size":     80,
	}
}

func TestPrepare(t *testing.T) {
	var b Builder

	c := testConfig()

	if _, _, err := b.Prepare(c); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}
