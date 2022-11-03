# Packer Plugin for Anka

This is a [Packer](https://www.packer.io/) Plugin for building images that work with [Veertu's Anka macOS Virtualization tool](https://veertu.com/).

- For use with the post-processor, it's important to use `anka registry add` to [set your default registry on the machine building your templates/tags](https://docs.veertu.com/anka/apple/command-line-reference/#registry-add).

## v3.0.0 Breaking Changes

- In order to minimize code complexity, Anka 2.x returns json which is not supported by Packer Plugin for Anka 3.x. However, Packer Plugin for Anka 2.x will continue to function with Anka 2.x.

## Compatibility

Packer Version | Veertu Anka Plugin Version
| --- | --- |
| 1.7.0 and above | >= 2.0.0 |
| below 1.7.0 | < 2.0.0 |
| 2.x | < 3.1.0 |
| 3.x | >= 3.1.0 |

## Installing with `packer init`

1. Add a packer block to your .pkr.hcl like this:

    ```
    packer {
      required_plugins {
        veertu-anka = {
          version = ">= v3.0.0"
          source = "github.com/veertuinc/veertu-anka"
        }
      }
    }
    ```

2. Then run `packer init {HCL file name}`
3. Run your `packer build` command with your hcl template

## Installing from Binary

1. [Install Packer v1.8 or newer](https://www.packer.io/downloads).
2. [Install Veertu Anka](https://veertu.com/download-anka-build/).
3. Download the [latest release](https://github.com/veertuinc/packer-plugin-veertu-anka/releases) for your host environment.
4. Unzip the plugin binaries to a location where Packer will detect them at run-time, such as any of the following:
    * The directory where the packer binary is.
    * The `~/.packer.d/plugins` directory.
    * The current working directory.
5. Rename the binary file to `packer-plugin-veertu-anka`.
6. Run your `packer build` command with your hcl template.

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

This will create a "base" VM template using the `.app` or `.ipsw` you specified in `installer = "/Applications/Install macOS Big Sur.app/"` with the name `anka-packer-base-macos`. Once the VM has been successfully created, it will push that VM to your default registry with the tag `veertu-registry-push-test`.

> If you don't specify `vm_name`, we will obtain it from the installer and create a name like `anka-packer-base-12.6-21G115`.

> **However, hw_uuid, port_forwarding_rules, and several other configuration settings are ignored for the created "base" vm.** We recommend using the `veertu-anka-vm-clone` builder to modify these values.

You can also skip the creation of the base VM template and use an existing VM template:

```hcl
source "veertu-anka-vm-clone" "anka-packer-from-source" { 
  vm_name = "anka-packer-from-source"
  source_vm_name = "anka-packer-base-macos"
  always_fetch = true
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source",
  ]
}
```

This will check to see if the VM template/tag exists locally, and if not, pull it from the registry:

```bash
❯ PKR_VAR_source_vm_tag="v1" export PACKER_LOG=1; packer build -var 'source_vm_name=anka-packer-base-macos' examples/clone-existing-with-port-forwarding-rules.pkr.hcl
. . .
2021/04/07 14:11:52 packer-plugin-veertu-anka plugin: 2021/04/07 14:11:52 Searching for anka-packer-base-macos locally...
2021/04/07 14:11:52 packer-plugin-veertu-anka plugin: 2021/04/07 14:11:52 Executing anka --machine-readable show anka-packer-base-macos
2021/04/07 14:11:53 packer-plugin-veertu-anka plugin: 2021/04/07 14:11:53 Could not find anka-packer-base-macos locally, looking in anka registry...
2021/04/07 14:11:53 packer-plugin-veertu-anka plugin: 2021/04/07 14:11:53 Executing anka --machine-readable registry pull --tag v1 anka-packer-base-macos
```

> Within your `.pkrvars.hcl` files, you can utilize `variable` blocks and then assign them values using the command line `packer build -var 'foo=bar'` or as environment variables `PKR_VAR_foo=bar` https://www.packer.io/docs/templates/hcl_templates/variables#assigning-values-to-build-variables

This will clone `anka-packer-base-macos` to a new VM and, if there are set configurations, make them.

> Check out the [examples directory](./examples) to see how port-forwarding and other options are used.

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
make all && make install
```

<!-- We recommend using goreleaser to perform all of the building, linting, and testing:

```bash
PACKER_CI_PROJECT_API_VERSION=$(go run . describe 2>/dev/null | jq -r '.api_version') goreleaser build --single-target --snapshot --rm-dist
``` -->

When testing with an example HCL:

```bash
export ANKA_LOG_LEVEL=debug; export ANKA_DELETE_LOGS=0; export PACKER_LOG=1; packer build examples/create-from-installer.pkr.hcl
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
