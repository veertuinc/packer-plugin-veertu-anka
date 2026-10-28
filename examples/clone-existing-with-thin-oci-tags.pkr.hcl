// Requires Anka 3.10.0 or later on the host, and network access to the pull registry.
// Defaults pull a real Homebrew bottle from GHCR. Robot CI overrides thin_oci_pull_*
// and oci_push_* with a temporary Zot registry.
//
// 1. Pull thin OCI tags onto source_vm_name (in order; optional --transform).
// 2. Clone source_vm_name and run the provisioners.
// 3. Push only the build's own layer as a thin OCI tag.

variable "source_vm_name" {
  type    = string
  default = "anka-packer-base-macos"
}

variable "vm_name" {
  type    = string
  default = "anka-packer-from-source-with-thin-oci-tags"
}

variable "thin_oci_pull_url" {
  type    = string
  default = "oci://ghcr.io/homebrew/core/cmake:4.3.4"
}

variable "thin_oci_pull_transforms" {
  type    = list(string)
  default = ["s|cmake/4.3.4|usr/local"]
}

variable "oci_push_remote" {
  type    = string
  default = "oci://registry.example.com/v2/veertu/"
}

variable "oci_push_tag" {
  type    = string
  default = "anka-packer-build-1"
}

variable "force" {
  type    = bool
  default = false
}

source "veertu-anka-vm-clone" "anka-packer-from-source-with-thin-oci-tags" {
  vm_name        = var.vm_name
  source_vm_name = var.source_vm_name

  thin_oci_pulls {
    url       = var.thin_oci_pull_url
    transform = var.thin_oci_pull_transforms
  }
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source-with-thin-oci-tags",
  ]

  provisioner "shell" {
    inline = [
      // With the Homebrew default this is /usr/local/bin/cmake; Robot CI overrides the pull URL.
      "echo 'built by packer' > ~/anka-packer-thin-oci-build.txt",
    ]
  }

  post-processor "veertu-anka-registry-push" {
    remote = var.oci_push_remote
    tag    = var.oci_push_tag
    thin   = true
    force  = var.force
  }
}
