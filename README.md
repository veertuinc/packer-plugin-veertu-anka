# Packer Plugin for Anka

This is a [Packer](https://www.packer.io/) Plugin for building images that work with [Veertu's Anka macOS Virtualization tool](https://veertu.com/).

- For use with the post-processor, it's important to use `anka registry add` to [set your default registry on the machine building your templates/tags](https://docs.veertu.com/anka/apple/command-line-reference/#registry-add).


## Compatibility

| Packer Version | Anka Packer Plugin Version |
| --- | --- |
| below 1.7.0 | < 1.8.0 |
| 1.7.0 and above | >= 2.0.0 |
| >= 1.11.0 | >= v4.0.0 |

| Packer Plugin Version | Anka CLI Version |
| --- | --- |
| 2.x | 2.x |
| >= 3.x | >= 3.0.0 |

## Installing with `packer init`

1. Add a packer block to your .pkr.hcl like this:

    ```
    packer {
      required_plugins {
        veertu-anka = {
          version = "= v4.x.x"
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
    * The `~/.config/packer/plugins` directory.
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

**Guest networking:** By default, **veertu-anka-vm-clone** and **veertu-anka-vm-create** run a guest `ping` loop against `8.8.8.8` after start (and after `boot_delay`) before provisioners. Set `wait_for_networking = false` on the source to skip that step. See the builder docs for details.

### Create a base VM

```hcl
source "veertu-anka-vm-create" "anka-packer-base-macos" {
  installer = "/Applications/Install macOS Big Sur.app/"
  vm_name   = "anka-packer-base-macos"
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

This creates a base VM from the `.app` or `.ipsw` in `installer`, then pushes it to your default registry with tag `veertu-registry-push-test`.

> If you do not set `vm_name`, the plugin builds a name from the installer, for example `anka-packer-base-12.6-21G115`.

> **`hw_uuid`, `port_forwarding_rules`, and several other settings are ignored on the created base VM.** Use `veertu-anka-vm-clone` to set those values.

### Thin OCI tags (Anka 3.10.0+)

Use `thin_oci_pulls` on `veertu-anka-vm-clone` to layer thin OCI tags onto `source_vm_name` before the build. Use `thin = true` on `veertu-anka-registry-push` to push only the files this build added (`remote` must be an OCI registry URL).

Public bottles work today from GHCR (Homebrew). Large tools such as Xcode are tags **you** push to your own OCI registry after you install them in a VM. See [Anka OCI tags](https://docs.veertu.com/anka/templates-and-registries/using-an-oci-registry/oci-tags/).

#### Homebrew bottles from GHCR (`skip_clone`)

With `skip_clone = true`, the builder changes `source_vm_name` in place. You do not need `remote_vm`. Pull order matters. Optional `transform` values become `anka pull --transform` options.

> Anka does not relocate the `@@HOMEBREW_PREFIX@@` / `@@HOMEBREW_CELLAR@@` placeholders in Homebrew bottles. Only bottles that link no other Homebrew libraries (`cellar :any_skip_relocation`, for example `cmake` and `ninja`) run after a thin pull. Bottles such as `jq` (which links `oniguruma`) do not.

```hcl
source "veertu-anka-vm-clone" "anka-packer-skip-clone-with-thin-oci-tags" {
  source_vm_name = "anka-packer-base-macos"
  skip_clone     = true

  thin_oci_pulls {
    url       = "oci://ghcr.io/homebrew/core/cmake:4.3.4"
    transform = ["s|cmake/4.3.4|usr/local"]
  }

  thin_oci_pulls {
    url       = "oci://ghcr.io/homebrew/core/ninja:1.13.2"
    transform = ["s|ninja/1.13.2|opt/homebrew"]
  }
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-skip-clone-with-thin-oci-tags",
  ]

  provisioner "shell" {
    inline = [
      "/usr/local/bin/cmake --version",
      "/opt/homebrew/bin/ninja --version",
      "mkdir -p /Users/Shared/anka-packer-skip-clone-thin",
      "echo 'built by packer skip_clone + thin OCI' > /Users/Shared/anka-packer-skip-clone-thin/build.txt",
    ]
  }

  post-processor "veertu-anka-registry-push" {
    remote = "oci://registry.example.com/v2/veertu/"
    tag    = "anka-packer-skip-clone-thin-1"
    thin   = true
  }
}
```

Full file: [examples/skip-clone-with-thin-oci-tags.pkr.hcl](./examples/skip-clone-with-thin-oci-tags.pkr.hcl)

#### Xcode from your OCI registry, then push the next thin layer

There is no public Xcode OCI tag on GHCR. Install Xcode once, stop the VM, and push a thin tag (Anka `push_mode` `2` / `anka push --thin`). Then pull that tag in Packer and push only this build's layer:

```hcl
source "veertu-anka-vm-clone" "anka-packer-with-xcode" {
  source_vm_name = "27.0.1"
  skip_clone     = true

  // Tag you pushed earlier, for example:
  //   ANKA_PUSH_MODE=2 anka push 27.0.1 -t xcode-27.1 oci://registry.example.com/v2/
  thin_oci_pulls {
    url = "oci://registry.example.com/v2/veertu/27.0.1:xcode-27.1"
  }
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-with-xcode",
  ]

  provisioner "shell" {
    inline = [
      "xcodebuild -version",
    ]
  }

  post-processor "veertu-anka-registry-push" {
    remote = "oci://registry.example.com/v2/veertu/"
    tag    = "fastlane-2.228.0"
    thin   = true
  }
}
```

#### Clone after a thin pull

Same `thin_oci_pulls` + `thin = true` pattern, but the builder clones first. Example (Homebrew defaults; override the pull URL with `-var` if needed): [examples/clone-existing-with-thin-oci-tags.pkr.hcl](./examples/clone-existing-with-thin-oci-tags.pkr.hcl)

Needs Anka 3.10.0+, network access to the pull registry (GHCR or yours), and an OCI `remote` for the thin push. See [vm-clone](./docs/builders/vm-clone.mdx) (`thin_oci_pulls`, `skip_clone`) and [anka-registry-push](./docs/post-processors/anka-registry-push.mdx) (`thin`).

### Clone an existing VM

If the template/tag is not local, the builder pulls it from the registry, then clones it:

```hcl
source "veertu-anka-vm-clone" "anka-packer-from-source" {
  vm_name        = "anka-packer-from-source"
  source_vm_name = "anka-packer-base-macos"
  always_fetch   = true
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source",
  ]
}
```

```bash
❯ PKR_VAR_source_vm_tag="v1" PACKER_LOG=1 packer build -var 'source_vm_name=anka-packer-base-macos' examples/clone-existing-with-port-forwarding-rules.pkr.hcl
. . .
2021/04/07 14:11:52 packer-plugin-veertu-anka plugin: 2021/04/07 14:11:52 Searching for anka-packer-base-macos locally...
2021/04/07 14:11:52 packer-plugin-veertu-anka plugin: 2021/04/07 14:11:52 Executing anka --machine-readable show anka-packer-base-macos
2021/04/07 14:11:53 packer-plugin-veertu-anka plugin: 2021/04/07 14:11:53 Could not find anka-packer-base-macos locally, looking in anka registry...
2021/04/07 14:11:53 packer-plugin-veertu-anka plugin: 2021/04/07 14:11:53 Executing anka --machine-readable registry pull --tag v1 anka-packer-base-macos
```

> In `.pkrvars.hcl`, use `variable` blocks. Set values with `packer build -var 'foo=bar'` or `PKR_VAR_foo=bar`. See [Packer variables](https://www.packer.io/docs/templates/hcl_templates/variables#assigning-values-to-build-variables).

> More templates (port forwarding and other options) are in the [examples directory](./examples).

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

### Run the plugin locally

```bash
go run -ldflags="-X main.version=$(./scripts/plugin-version.bash --dev) -X main.commit=$(git rev-parse HEAD)" main.go
# Or set an explicit version: VERSION=4.7.0 make go.build
```

### Building, Linting, and Testing

```bash
make all && make install
```

<!-- We recommend using goreleaser to perform all of the building, linting, and testing:

```bash
PACKER_CI_PROJECT_API_VERSION=$(go run . describe 2>/dev/null | jq -r '.api_version') goreleaser build --single-target --snapshot --clean
``` -->

When testing with an example HCL:

```bash
ANKA_LOG_LEVEL=debug ANKA_DELETE_LOGS=0 PACKER_LOG=1 packer build examples/create-from-installer.pkr.hcl
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
