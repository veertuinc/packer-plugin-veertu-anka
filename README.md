# Packer Plugin for Anka

This is a [Packer](https://www.packer.io/) Plugin for building images that work with [Veertu's Anka macOS Virtualization tool](https://veertu.com/).

- Note that this plugin does not manage VM templates. Once it creates a Template, it is up to you to use it or delete it.

- For use with the post-processor, it's important to use `anka registry add` to [set your default registry on the machine building your templates/tags](https://docs.veertu.com/anka/intel/command-line-reference/#registry-add).
## v2.0.0 Breaking Changes

- Plugin will only work with Packer v1.7 or later.

- Plugin has been renamed from `packer-builder-veertu-anka` to `packer-plugin-veertu-anka`.

- Builder has been renamed from `veertu-anka` to `veertu-anka-vm-clone` and `veertu-anka-vm-create`.

- Pre-version-1.5 "legacy" Packer templates, which were exclusively JSON and follow a different format, are no longer compatible and must be updated to either HCL or the new JSON format: https://www.packer.io/docs/templates/hcl_templates/syntax-json

## Compatibility

Packer Version | Veertu Anka Plugin Version
--- | ---
1.7.0 and above | >= 2.0.0
below 1.7.0 | < 2.0.0

## Installing with `packer init`

1. Add a packer block to your .pkr.hcl like this:

    ```
    packer {
      required_plugins {
        veertu-anka = {
          version = ">= v2.1.0"
          source = "github.com/veertuinc/veertu-anka"
        }
      }
    }
    ```

2. Then run `packer init {HCL file name}`
3. Run your `packer build` command with your hcl template

## Installing from Binary

1. [Install Packer v1.7 or newer](https://www.packer.io/downloads)
2. [Install Veertu Anka v2.3.1 or newer](https://veertu.com/download-anka-build/)
3. Download the [latest release](https://github.com/veertuinc/packer-plugin-veertu-anka/releases) for your host environment
4. Unzip the plugin binaries to a location where Packer will detect them at run-time, such as any of the following:
    * The directory where the packer binary is.
    * The `~/.packer.d/plugins` directory.
    * The current working directory.
5. Rename the binary file to `packer-plugin-veertu-anka`
6. Run your `packer build` command with your hcl template

## Documentation

| Builders | Post Processors |
| --- | --- |
| [[ veertu-anka-vm-create ]](./docs/builders/vm-create.mdx) | [[ veertu-anka-registry-push ]](./docs/post-processors/anka-registry-push.mdx) |
| [[ veertu-anka-vm-clone ]](./docs/builders/vm-clone.mdx) | |

## Usage

> Currently file provisioners do not support ~ or \$HOME in the destination paths. Please use absolute or relative paths.

The most basic pkr.hcl file you can build from is:

```hcl
source "veertu-anka-vm-create" "anka-packer-base-macos" {
  installer = "/Applications/Install macOS Big Sur.app/"
  vm_name = "anka-packer-base-macos"
}

build {
  sources = [
    "source.veertu-anka-vm-create.anka-packer-base-macos"
  ]

  post-processor "veertu-anka-registry-push" {
    tag = "veertu-registry-push-test"
  }
}
```

This will create a "base" VM template using the `.app` you specified in `installer` with the name `anka-packer-base-macos`. Once the VM has been successfully created, it will push that VM to your default registry with the `veertu-registry-push-test` tag.

> If you didn't specify `vm_name`, we would automatically pull it from the installer app and create a name like `anka-packer-base-11.4-16.6.01`.

> When using `installer`, you can modify the base VM default resource values with `disk_size`, `ram_size`, and `vcpu_count`. Otherwise, the Anka CLI will determine what's best for you based on the host's hardware.

> **However, hw_uuid, port_forwarding_rules, and several other configuration settings are ignored for the created "base" vm.** We recommend using the `veertu-anka-vm-clone` builder to modify these values.

You can also skip the creation of the base VM template and use an existing VM template:

```hcl
source "veertu-anka-vm-clone" "anka-packer-from-source" { 
  vm_name = "anka-packer-from-source"
  source_vm_name = "anka-packer-base-macos"
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source",
  ]
}
```

This will check to see if the VM template/tag exists locally, and if not, pull it from the registry:

```bash
❯ PKR_VAR_source_vm_tag="v1" PACKER_LOG=1 packer build -var 'source_vm_name=anka-packer-base-macos' examples/clone-existing-with-port-forwarding-rules.pkr.hcl
. . .
2021/04/07 14:11:52 packer-plugin-veertu-anka plugin: 2021/04/07 14:11:52 Searching for anka-packer-base-macos locally...
2021/04/07 14:11:52 packer-plugin-veertu-anka plugin: 2021/04/07 14:11:52 Executing anka --machine-readable show anka-packer-base-macos
2021/04/07 14:11:53 packer-plugin-veertu-anka plugin: 2021/04/07 14:11:53 Could not find anka-packer-base-macos locally, looking in anka registry...
2021/04/07 14:11:53 packer-plugin-veertu-anka plugin: 2021/04/07 14:11:53 Executing anka --machine-readable registry pull --tag v1 anka-packer-base-macos
```

> Within your `.pkrvars.hcl` files, you can utilize `variable` blocks and then assign them values using the command line `packer build -var 'foo=bar'` or as environment variables `PKR_VAR_foo=bar` https://www.packer.io/docs/templates/hcl_templates/variables#assigning-values-to-build-variables

This will clone `anka-packer-base-macos` to a new VM and, if there are set configurations, make them.

> Check out the [examples directory](./examples) to see how port-forwarding and other options are used

### Build Variables

Packer allows for the exposure of build variables which connects information related to the artifact that was built. Those variables can then be accessed by `post-processors` and `provisioners`.

The variables we expose are:

* `VMName`: name of the artifact vm
* `OSVersion`: OS version from which the artifact was created 
  * eg. 10.15.7
* `DarwinVersion`: Darwin version that is compatible with the current OS version
  * eg. 19.6.0

```hcl
locals {
  source_vm_name = "anka-packer-base-11.2-16.4.06"
}

source "veertu-anka-vm-clone" "anka-macos-from-source" {
  "source_vm_name": "${local.source_vm_name}",
  "vm_name": "anka-macos-from-source"
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-macos-from-source"
  ]

  provisioner "shell" {
    inline = [
      "echo vm_name is ${build.VMName}",
      "echo os_version is ${build.OSVersion}",
      "echo darwin_version is ${build.DarwinVersion}"
    ]
  }
}
```

---

## Development

You will need a recent golang installed and setup. See `go.mod` for which version is expected.

We use [gomock](https://github.com/golang/mock) to quickly and reliably mock our interfaces for testing. This allows us to easily test when we expect logic to be called without having to rewrite golang standard library functions with custom mock logic. To generate one of these mocked interfaces, installed the mockgen binary by following the link provided and then run the `make go.test`.

- You must install `packer-sdc` to generate docs and HCL2spec.

### Building, Linting, and Testing

```bash
make all
```

<!-- We recommend using goreleaser to perform all of the building, linting, and testing:

```bash
PACKER_CI_PROJECT_API_VERSION=$(go run . describe 2>/dev/null | jq -r '.api_version') goreleaser build --single-target --snapshot --rm-dist
``` -->

When testing with an example HCL:

```bash
export PACKER_LOG=1; packer build examples/create-from-installer.pkr.hcl
```

To test the post processor you will need an active vpn connection that can reach an anka registry. You can setup an anka registry by either adding the registry locally with:

```bash
anka registry add <registry_name> <registry_url>
```

-or-

You can setup the `create-from-installer-with-post-processing.pkr.hcl` with the correct registry values and update the make target `anka.test` to use that .pkr.hcl file and run:

```bash
make create-test
```

[Packer Builder]: https://www.packer.io/docs/extending/custom-builders.html
[Veertu Anka]: https://veertu.com/
