Type: `veertu-anka-registry-push`

The `veertu-anka-registry-push` Packer Post Processor is able to push your created Anka VM templates to the [Anka Build Cloud Registry](https://veertu.com/anka-build/) through the [Anka Virtualization](https://veertu.com/technology/) package.

This post-processor is part of the [Veertu Anka plugin](https://github.com/veertuinc/packer-plugin-veertu-anka). To install this plugin using `packer init`, add the following Packer block to your hcl template:

```hcl
packer {
  required_plugins {
    veertu-anka = {
      version = "= 4.x.x"
      source  = "github.com/veertuinc/veertu-anka"
    }
  }
}
```

## Configuration Reference

There are many configuration options available for the post-processor. They are
segmented below into two categories: required and optional parameters.

### _**Required Configuration**_

* `type` (String) **Must be `veertu-anka-registry-push`; no other types**

### _**Optional Configuration**_

* `cacert` (String) Path to a CA Root certificate.

* `cert` (String) Path to your node certificate (if certificate authority is enabled).

* `description` (String) The description of the tag.

* `insecure` (Boolean) Skip TLS verification.

* `key` (String) Path to your node certificate key if the client/node certificate doesn't contain one.

  > `cacert`, `cert`, `insecure`, and `key` are only for an Anka Registry. The build fails if you set them when `remote` is an OCI registry URL, because `anka push` does not accept them.

* `local` (Boolean) Assign a tag to your local template and avoid pushing to the Registry.

* `remote` (String) The registry URL or name to target for registry operation (will use your default configuration if not set).

  > (Anka 3.9.0+) To push to an OCI registry, set `remote` to an OCI registry URL: an `oci://` URL, or an `http(s)://` URL with `/v2` in its path (for example `oci://registry.example.com/v2/veertu/`). See [Push to an OCI registry](#push-to-an-oci-registry).

* `thin` (Boolean) (Anka 3.10.0+) Push only the layer that this build added, as a thin OCI tag (`anka push --thin`). Requires `remote` to be an OCI registry URL. Cannot be used with `local`. Defaults to `false` (a full OCI tag).

* `remote_vm` (String) The name of a registry template you want to push the local template onto.

  > To push a new tag onto the registry template that you started from, without `remote_vm`, set `skip_clone = true` on the `veertu-anka-vm-clone` builder. The builder then provisions `source_vm_name` in place, and the post-processor pushes it with its own name. Example: [skip-clone-with-post-processing.pkr.hcl](https://github.com/veertuinc/packer-plugin-veertu-anka/blob/edge/examples/skip-clone-with-post-processing.pkr.hcl)

* `tag` (String) The name of the tag to push (will default as 'latest' if not set).

* `force` (Boolean) Whether or not to forcefully push, regardless of a tag already existing

* `delete_template_post_push` (Boolean) When `true`, after a **successful remote registry push** (`local` must be `false`) the post-processor runs `anka delete --yes` on the **local** VM template from the Anka builder (`artifact` name). It does **not** run for `local = true` (local tagging only). Defaults to `false`. Use this to avoid leaving duplicate local templates when reusing the same `vm_name` across builds. If deletion fails, the post-processor returns an error even though the push already succeeded.

## Other 

When using `packer build -force`, the post-processor will issue a [revert API call](https://docs.veertu.com/anka/anka-build-cloud/working-with-registry-and-api/#revert) to remove the existing tag before pushing the new.

### Push to an OCI registry

When `remote` is an OCI registry URL, the post-processor runs `anka push` instead of `anka registry push`:

* It does not use the Anka Registry list and revert API. An OCI registry does not have it.
* `force = true` or `packer build -force` adds `--force`. An OCI push with the same tag name replaces the tag.
* The repository is the last segment of the `remote` path after `/v2/`: `oci://registry.example.com/v2/veertu/tools` pushes to `veertu/tools`. If the path ends with `/`, the repository is the local VM name. `remote_vm` sets the repository name and overrides the path.
* With `thin = true`, the tag holds only the files that this build added. Pull it onto a VM that already has the base, for example with the clone builder's `thin_oci_pulls`.
* The Anka Build Cloud Controller pulls only from an Anka Registry. For a Controller template, push a full tag to the Anka Registry.

Registry credentials come from the host environment, for example `ANKA_REGISTRY_AUTH_TOKEN`. A remote name from `anka registry add` always uses `anka registry push`, even when that remote is an OCI registry.

```hcl
post-processor "veertu-anka-registry-push" {
  remote = "oci://registry.example.com/v2/veertu/"
  tag    = "xcode-26.1"
  thin   = true
}
```

Examples:
- [clone-existing-with-thin-oci-tags.pkr.hcl](https://github.com/veertuinc/packer-plugin-veertu-anka/blob/edge/examples/clone-existing-with-thin-oci-tags.pkr.hcl) (clone, then thin push)
- [skip-clone-with-thin-oci-tags.pkr.hcl](https://github.com/veertuinc/packer-plugin-veertu-anka/blob/edge/examples/skip-clone-with-thin-oci-tags.pkr.hcl) (`skip_clone`, multiple thin pulls, then thin push)

## Example

Here is an example that uses the file and shell provisioners.

Also see:

* [clone-existing-with-post-processing.pkr.hcl](https://github.com/veertuinc/packer-plugin-veertu-anka/blob/edge/examples/clone-existing-with-post-processing.pkr.hcl)
* [create-from-installer-with-post-processing.pkr.hcl](https://github.com/veertuinc/packer-plugin-veertu-anka/blob/edge/examples/create-from-installer-with-post-processing.pkr.hcl)

```hcl

variable "source_vm_name" {
  type = string
  default = "anka-packer-base-macos"
}

variable "vm_name" {
  type = string
  default = "anka-packer-from-source"
}

source "veertu-anka-vm-clone" "clone" {
  vm_name = "${var.vm_name}"
  source_vm_name = "${var.source_vm_name}"
}

build {
  sources = [
    "source.veertu-anka-vm-clone.clone",
  ]
  
  provisioner "file" {
    destination = "/private/tmp/"
    source      = "./examples/ansible"
  }
  provisioner "shell" {
    inline = [
      "[[ ! -d /tmp/ansible ]] && exit 100",
      "touch /tmp/ansible/test1"
    ]
  }

  post-processor "veertu-anka-registry-push" {
    tag                    = "v2"
    delete_template_post_push = true
  }
}

```
