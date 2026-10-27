// Requires Anka 3.10.0 or later on the host.
// 1. Pulls thin OCI tags onto source_vm_name (in order, each with --thin and optional --transform).
// 2. Clones source_vm_name and runs the provisioners.
// 3. Pushes only the build's own layer as a thin OCI tag.

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
  default = "http://127.0.0.1:15455/v2/veertu/anka-packer-thin-tool:tool-1"
}

variable "thin_oci_pull_transforms" {
  type    = list(string)
  default = []
}

variable "oci_push_remote" {
  type    = string
  default = "http://127.0.0.1:15455/v2/veertu/"
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
