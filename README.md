# Veertu Anka Builder (for packer.io)

The builder does not manage images. Once it creates an image, it is up to you to use it or delete it.

## Running

```bash
mkdir -p ~/.packer.d/plugins
go build -o ~/.packer.d/plugins/packer-builder-veertu-anka
packer build examples/macos-sierra.json
```

## Configuration

```
  "builders": [{
    "type": "veertu-anka",
    "installer_app": "/Applications/Install macOS Sierra.app/",
    "disk_size": "25G",
    "source_vm_name": "{{user `source_vm_name`}}"
  }]
```

* `type` (required)

Must be `veertu-anka`

* `installer_app` (optional)

The path to a macOS installer. This must be provided if `source_vm_name` isn't provided`. This process takes about 20 minutes

* `disk_size` (optional)

The size in "[0-9]+G" format, defaults to `25G`

* `source_vm_name` (optional)

The VM to clone for provisioning, either stopped or suspended.

## Development

```bash
make packer-test
```

If you've already built a base macOS VM, you can use:

```bash
make packer-test SOURCE_VM_NAME=macos-10.12.3-base
```