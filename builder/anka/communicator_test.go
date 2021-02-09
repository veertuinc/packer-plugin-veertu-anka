package anka

import (
	"crypto/sha256"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"context"

	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template"
	oldPacker "github.com/hashicorp/packer/packer"
	"github.com/hashicorp/packer/provisioner/file"
	"github.com/hashicorp/packer/provisioner/shell"
)

func TestCommunicator_impl(t *testing.T) {
	var _ packer.Communicator = new(Communicator)
}

// TestUploadDownload verifies that basic upload / download functionality works
func TestUploadDownload(t *testing.T) {
	ui := packer.TestUi(t)

	tpl, err := template.Parse(strings.NewReader(ankaBuilderConfig))
	if err != nil {
		t.Fatalf("Unable to parse config: %s", err)
	}

	if os.Getenv("PACKER_ANKA_DISK_BASE") == "" {
		t.Skip("This test is only run when PACKER_ANKA_DISK_BASE is set")
	}

	tpl.Builders["veertu-anka"].Config["source_vm_name"] = os.Getenv("PACKER_ANKA_DISK_BASE")
	log.Printf("%#v", tpl.Builders["veertu-anka"].Config)

	// Setup the builder
	builder := &Builder{}
	_, warnings, err := builder.Prepare(tpl.Builders["veertu-anka"].Config)
	if err != nil {
		t.Fatalf("Error preparing configuration %s", err)
	}
	if len(warnings) > 0 {
		t.Fatal("Encountered configuration warnings; aborting")
	}

	// Setup the provisioners
	upload := &file.Provisioner{}
	err = upload.Prepare(tpl.Provisioners[0].Config)
	if err != nil {
		t.Fatalf("Error preparing upload: %s", err)
	}
	download := &file.Provisioner{}
	err = download.Prepare(tpl.Provisioners[1].Config)
	if err != nil {
		t.Fatalf("Error preparing download: %s", err)
	}
	defer os.Remove("my-strawberry-cake")

	// Add hooks so the provisioners run during the build
	hooks := map[string][]packer.Hook{}
	hooks[packer.HookProvision] = []packer.Hook{
		&oldPacker.ProvisionHook{
			Provisioners: []*oldPacker.HookedProvisioner{
				{Provisioner: upload, Config: nil, TypeName: ""},
				{Provisioner: download, Config: nil, TypeName: ""},
			},
		},
	}
	hook := &packer.DispatchHook{Mapping: hooks}

	// Run things
	artifact, err := builder.Run(context.Background(), ui, hook)
	if err != nil {
		t.Fatalf("Error running build %s", err)
	}
	// Preemptive cleanup
	defer func() {
		_ = artifact.Destroy()
	}()

	// Verify that the thing we downloaded is the same thing we sent up.
	inputFile, err := ioutil.ReadFile("test-fixtures/onecakes/strawberry")
	if err != nil {
		t.Fatalf("Unable to read input file: %s", err)
	}
	outputFile, err := ioutil.ReadFile("my-strawberry-cake")
	if err != nil {
		t.Fatalf("Unable to read output file: %s", err)
	}
	if sha256.Sum256(inputFile) != sha256.Sum256(outputFile) {
		t.Fatalf("Input and output files do not match\n"+
			"Input:\n%s\nOutput:\n%s\n", inputFile, outputFile)
	}
}

// TestShellProvisioner verifies that shell provisioning works
func TestExecuteShellCommand(t *testing.T) {
	ui := packer.TestUi(t)

	tpl, err := template.Parse(strings.NewReader(ankaBuilderShellConfig))
	if err != nil {
		t.Fatalf("Unable to parse config: %s", err)
	}

	if os.Getenv("PACKER_ANKA_DISK_BASE") == "" {
		t.Skip("This test is only run when PACKER_ANKA_DISK_BASE is set")
	}

	tpl.Builders["veertu-anka"].Config["source_vm_name"] = os.Getenv("PACKER_ANKA_DISK_BASE")
	log.Printf("%#v", tpl.Builders["veertu-anka"].Config)

	// Setup the builder
	builder := &Builder{}
	_, warnings, err := builder.Prepare(tpl.Builders["veertu-anka"].Config)
	if err != nil {
		t.Fatalf("Error preparing configuration %s", err)
	}
	if len(warnings) > 0 {
		t.Fatal("Encountered configuration warnings; aborting")
	}

	// Setup the provisioners
	inline := &shell.Provisioner{}
	err = inline.Prepare(tpl.Provisioners[0].Config)
	if err != nil {
		t.Fatalf("Error preparing inline: %s", err)
	}

	scripts := &shell.Provisioner{}
	err = scripts.Prepare(tpl.Provisioners[1].Config)
	if err != nil {
		t.Fatalf("Error preparing scripts: %s", err)
	}

	download := &file.Provisioner{}
	err = download.Prepare(tpl.Provisioners[2].Config)
	if err != nil {
		t.Fatalf("Error preparing download: %s", err)
	}
	defer os.Remove("provisioner_log")

	// Add hooks so the provisioners run during the build
	hooks := map[string][]packer.Hook{}
	hooks[packer.HookProvision] = []packer.Hook{
		&oldPacker.ProvisionHook{
			Provisioners: []*oldPacker.HookedProvisioner{
				{Provisioner: inline, Config: nil, TypeName: ""},
				{Provisioner: scripts, Config: nil, TypeName: ""},
				{Provisioner: download, Config: nil, TypeName: ""},
			},
		},
	}
	hook := &packer.DispatchHook{Mapping: hooks}

	// Run things
	artifact, err := builder.Run(context.Background(), ui, hook)
	if err != nil {
		t.Fatalf("Error running build %s", err)
	}

	// Preemptive cleanup
	defer func() {
		_ = artifact.Destroy()
	}()

	outputFile, err := ioutil.ReadFile("provisioner_log")
	if err != nil {
		t.Fatalf("Unable to read output file: %s", err)
	}

	if string(outputFile) != "inline\nanka\nroot\nscript1\nscript2\n" {
		t.Fatalf("Didn't expect output of %q", outputFile)
	}
}

const ankaBuilderConfig = `
{
  "builders": [
    {
      "type": "veertu-anka",
      "disk_size": "40G",
      "source_vm_name": "blah"
    }
  ],
  "provisioners": [
    {
      "type": "file",
      "source": "test-fixtures/onecakes/strawberry",
      "destination": "/tmp/strawberry-cake"
    },
    {
      "type": "file",
      "source": "/tmp/strawberry-cake",
      "destination": "my-strawberry-cake",
      "direction": "download"
    }
  ]
}
`

const ankaBuilderShellConfig = `
{
  "builders": [
    {
      "type": "veertu-anka",
      "disk_size": "40G",
      "source_vm_name": "blah"
    }
  ],
  "provisioners": [
    {
      "type": "shell",
      "inline": [
        "echo inline >> /tmp/provisioner_log",
        "whoami >> /tmp/provisioner_log",
        "echo hello from shell!",
        "sudo whoami >> /tmp/provisioner_log"
        ]
    },
    {
      "type": "shell",
      "scripts": [
        "test-fixtures/scripts/script1.sh",
        "test-fixtures/scripts/script2.sh"
      ]
    },
    {
      "type": "file",
      "source": "/tmp/provisioner_log",
      "destination": "provisioner_log",
      "direction": "download"
    }
  ]
}
`
