package anka

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/veertuinc/packer-plugin-veertu-anka/client"
)

// ThinOCIPull is one thin OCI tag that the clone builder pulls onto source_vm_name before it clones.
// Every entry is pulled with --thin: a full pull after thin tags would remove those thin tags.
type ThinOCIPull struct {
	URL       string   `mapstructure:"url"`
	Transform []string `mapstructure:"transform"`
}

func validateThinOCIPulls(thinOCIPulls []ThinOCIPull) error {
	for index, thinOCIPull := range thinOCIPulls {
		if err := validateThinOCIPull(thinOCIPull); err != nil {
			return fmt.Errorf("thin_oci_pulls[%d]: %w", index, err)
		}
	}
	return nil
}

func validateThinOCIPull(thinOCIPull ThinOCIPull) error {
	if thinOCIPull.URL == "" {
		return errors.New("url is required")
	}
	if !client.IsOCIRemote(thinOCIPull.URL) {
		return fmt.Errorf("url %q must be an oci:// URL or a URL with /v2 in its path", thinOCIPull.URL)
	}
	for _, transform := range thinOCIPull.Transform {
		if strings.TrimSpace(transform) == "" {
			return errors.New("transform entries must not be empty")
		}
	}
	return nil
}

func applyThinOCIPulls(ankaClient client.Client, vmName string, thinOCIPulls []ThinOCIPull, ui packer.Ui) error {
	for _, thinOCIPull := range thinOCIPulls {
		ui.Say(fmt.Sprintf("Pulling thin OCI tag %s onto source VM %s", thinOCIPull.URL, vmName))
		pullParams := client.OCIPullParams{
			VMID:       vmName,
			Remote:     thinOCIPull.URL,
			Thin:       true,
			Transforms: thinOCIPull.Transform,
		}
		err := client.RunWithOutputStreamToUI(ui, func(outputStream chan string) error {
			return ankaClient.OCIPull(pullParams, outputStream)
		})
		if err != nil {
			return fmt.Errorf("failed to pull thin OCI tag %s onto source VM %s: %w", thinOCIPull.URL, vmName, err)
		}
	}
	return nil
}
