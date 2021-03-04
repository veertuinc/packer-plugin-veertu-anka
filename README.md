# Packer Plugin for Anka

This is a [Packer Builder] for building images that work with [Veertu Anka], a macOS virtualization tool.

Note that this builder does not manage images. Once it creates an image, it is up to you to use it or delete it.

### v2.0.0 Breaking Changes

* Plugin will only work with Packer v1.7 or later.
* Plugin has been renamed from `packer-builder-veertu-anka` to `packer-plugin-veertu-anka`.
* Builder has been renamed from `veertu-anka` to `veertu-anka-vm`.

### Compatibility

Packer Version | Veertu Anka Plugin Version
--- | ---
Up to 1.4.5 | 1.1.0
1.5.x and above | 1.2.0
1.7.x and above | 2.0.0

## Installing from Binary

1. [Install Packer v1.7 or newer](https://www.packer.io/downloads)
2. [Install Veertu Anka v2.3.1 or newer](https://veertu.com/download-anka-build/)
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
      "installer_app": "/Applications/Install macOS Big Sur.app",
      "type": "veertu-anka"
    }
  ],
  "post-processors": [
    {
      "type": "veertu-anka-registry-push",
      "tag": "latest"
    }
  ]
}
```

This will create a base VM template using the `.app` you specified in `installer_app` with a name like `anka-packer-base-{macOSVersion}`. Once the base VM template is created, it will create a clone from it (that shares the underlying layers from the base VM template, minimizing the amount of disk space used). Once the VM has been successfully created, it will push that VM to your default registry with the `latest` tag.

> When using `installer_app`, you can modify the base VM default resource values with `disk_size`, `ram_size`, and `cpu_count`. Otherwise, defaults will be used (see "Configuration" section).

You can also skip the creation of the base VM template and use an existing VM template (`10.15.6`):

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

Or, create a variable inside for the `source_vm_name` and then run: `packer build -var 'source_vm_name=10.15.6' examples/macos-catalina-existing.json`.

> The `installer_app` is ignored if you've specified `source_vm_name` and it does not exist already

This will clone `10.15.6` to a new VM and, if there are differences from the base VM, modify CPU, RAM, and DISK.

> Check out the [examples directory](./examples) to see how port-forwarding and other options are used

## Builders 

### veertu-anka-vm

#### Required Configuration

* `type` (String)

Must be `veertu-anka-vm`.

#### Optional Configuration

* `boot_delay` (String)

The time to wait before running packer provisioner commands, defaults to `10s`.

* `cpu_count` (String)

The number of CPU cores, defaults to `2`.

* `disk_size` (String)

The size in "[0-9]+G" format, defaults to `25G`.

> We will automatically resize the internal disk for you by executing: `diskutil apfs resizeContainer disk1 0`

* `hw_uuid` (String)

The Hardware UUID you wish to set (usually generated with `uuidgen`).

* `installer_app` (String)

The path to a macOS installer. This must be provided if `source_vm_name` isn't provided. This process takes about 20 minutes. The resulting VM template name will be `anka-packer-base-{macOSVersion}`.

* `port_forwarding_rules` (Struct)

> If port forwarding rules are already set and you want to not have them fail the packer build, use `packer build --force`

```json
  "builders": [{
    "type": "veertu-anka",
    "cpu_count": 8,
    "ram_size": "10G",
    "source_vm_name": "anka-packer-base-10.15.7",
    "port_forwarding_rules": [
      {
        "port_forwarding_guest_port": 80,
        "port_forwarding_host_port": 12345,
        "port_forwarding_rule_name": "website"
      },
      {
        "port_forwarding_guest_port": 8080
      }
    ]
  }]
```

* `ram_size` (String)

The size in "[0-9]+G" format, defaults to `2G`.

* `source_vm_name` (String)

The VM to clone for provisioning, either stopped or suspended.

* `vm_name` (String)

The name for the VM that is created. One is generated if not provided (`anka-packer-{10RandomCharacters}`).

## Post Processors

### veertu-anka-registry-push

#### Required Configuration

* `type` (String)

Must be `veertu-anka-registry-push`

#### Optional Configuration

* `cacert` (String)

Path to a CA Root certificate.

* `cert` (String)

Path to your node certificate (if certificate authority is enabled).

* `description` (String)

The description of the tag.

* `insecure` (Boolean)

Skip TLS verification.

* `key` (String)

Path to your node certificate key if the client/node certificate doesn't contain one.

* `local` (Boolean)

Assign a tag to your local template and avoid pushing to the Registry.

* `registry-path` (String)

The registry URL (will use your default configuration if not set).

* `remtoe` (String)

The registry name (will use your default configuration if not set).

* `remote-vm` (String)

The name of a registry template you want to push the local template onto.

* `tag` (String)

The name of the tag to push (will default as 'latest' if not set).

## Development

You will need a recent golang installed and setup. See `go.mod` for which version is expected.

```bash
make packer-test
```

-or-

```bash
make build-and-install && PACKER_LOG=1 packer build examples/create-from-installer.json
```

[Packer Builder]: https://www.packer.io/docs/extending/custom-builders.html
[Veertu Anka]: https://veertu.com/
