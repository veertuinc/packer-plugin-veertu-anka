# Packer Builder for Anka

This is a [Packer Builder] for building images that work with [Veertu Anka], a
macOS virtualization tool.

Note that this builder does not manage images. Once it creates an image, it is up
to you to use it or delete it.

## Installing from Binary

1. Install Packer
2. Install Veertu Anka
3. Download the [latest release](https://github.com/buildkite/packer-builder-veertu-anka/releases) for your host environment
4. Unzip the plugin binaries to a location where Packer will detect them at run-time, such as any of the following:
    * The directory where the packer binary is.
    * The `~/.packer.d/plugins` directory.
    * The current working directory.
5. Change to a directory where you have packer templates, and run as usual.

## Configuration

```json
{
  "builders": [{
    "type": "veertu-anka",
    "installer_app": "/Applications/Install macOS Sierra.app/",
    "disk_size": "25G",
    "source_vm_name": "{{user `source_vm_name`}}"
  }]
}
```

* `type` (required)

Must be `veertu-anka`

* `installer_app` (optional)

The path to a macOS installer. This must be provided if `source_vm_name` isn't
provided. This process takes about 20 minutes

* `disk_size` (optional)

The size in "[0-9]+G" format, defaults to `25G`

* `ram_size` (optional)

The size in "[0-9]+G" format, defaults to `2G`

* `cpu_count` (optional)

The number of CPU cores, defaults to `2`

* `source_vm_name` (optional)

The VM to clone for provisioning, either stopped or suspended.

If you specify both `source_vm_name` and `installer_app`, and a VM image with `source_vm_name`
does not exist locally, a VM image with that name is created for you using the `installer_app`.
This process takes about 20 minutes.

* `vm_name` (optional)

The name for the VM that is created. One is generated if not provided.

* `boot_delay` (optional)

The time to wait before running packer provisioner commands, defaults to `2s`.

## Development

You will need a recent golang installed and setup.

```bash
make packer-test
```

If you've already built a base macOS VM, you can use:

```bash
make packer-test SOURCE_VM_NAME=macos-10.12.3-base
```

[Packer Builder]: https://www.packer.io/docs/extending/custom-builders.html
[Veertu Anka]: https://veertu.com/
