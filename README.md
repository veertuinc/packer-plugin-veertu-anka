# Packer Builder for Anka

This is a [Packer Builder] for building images that work with [Veertu Anka], a
macOS virtualization tool.

Note that this builder does not manage images. Once it creates an image, it is up
to you to use it or delete it.

### Compatibility

Packer Version | Builder for Anka Version
--- | ---
Up to 1.4.5 | 1.1.0
1.5.x and above | 1.2.0

## Installing from Binary

1. Install Packer
2. Install Veertu Anka
3. Download the [latest release](https://github.com/veertuinc/packer-builder-veertu-anka/releases) for your host environment
4. Unzip the plugin binaries to a location where Packer will detect them at run-time, such as any of the following:
    * The directory where the packer binary is.
    * The `~/.packer.d/plugins` directory.
    * The current working directory.
5. Change to a directory where you have packer templates, and run as usual.

## Usage

The most basic json file you can build from is:

```json
{
  "builders": [
    {
      "installer_app": "/Applications/Install macOS Catalina.app",
      "type": "veertu-anka"
    }
  ]
}
```

This will create a base VM template using the `.app` you specified in `installer_app` (using the version values from the installer's Info.plist file). Once the base VM template is created, it will create a clone from it (that shares the underlying layers from the base VM template, minimizing the amount of disk space used).

You can also skip the creation of the base VM template and use an existing:

```json
{
  "provisioners": [
    {
      "type": "shell",
      "inline": [
        "sleep 5",
        "echo hello world",
        "echo llamas rock"
      ]
    }
  ],
  "builders": [{
    "type": "veertu-anka",
    "cpu_count": 8,
    "ram_size": "10G",
    "disk_size": "150G",
    "source_vm_name": "10.15.6"
  }]
}
```

> `installer_app` is ignored if you've specified `source_vm_name` and it exists already

This will clone `10.15.6` to a new VM and modify CPU, RAM, and DISK.

## Configuration

* `type` (required)

Must be `veertu-anka`

* `installer_app` (optional)

The path to a macOS installer. This must be provided if `source_vm_name` isn't
provided. This process takes about 20 minutes

* `disk_size` (optional)

The size in "[0-9]+G" format, defaults to `80G`

> We will automatically resize the internal disk for you by executing: `diskutil apfs resizeContainer disk1 0`

* `ram_size` (optional)

The size in "[0-9]+G" format, defaults to `8G`

* `cpu_count` (optional)

The number of CPU cores, defaults to `4`

* `source_vm_name` (optional)

The VM to clone for provisioning, either stopped or suspended.

* `vm_name` (optional)

The name for the VM that is created. One is generated if not provided.

* `boot_delay` (optional)

The time to wait before running packer provisioner commands, defaults to `10s`.

## Development

You will need a recent golang installed and setup. See `go.mod` for which version is expected.

```bash
make packer-test
```

If you've already built a base macOS VM, you can use:

```bash
make packer-test SOURCE_VM_NAME=macos-10.12.3-base
```

```bash
make build-and-install && PACKER_LOG=1 packer build examples/macos-catalina-existing.json
```

## Release

We use [goreleaser](https://goreleaser.com).

To locally try out the release workflow (build, package, but don't sign or publish):

```bash
make release-dry-run
```

[Packer Builder]: https://www.packer.io/docs/extending/custom-builders.html
[Veertu Anka]: https://veertu.com/
