variable "source_vm_name" {
  type    = string
  default = "anka-packer-base-macos"
}

variable "tag" {
  type    = string
  default = ""
}

variable "force" {
  type    = bool
  default = false
}

variable "remote" {
  type    = string
  default = ""
}

source "veertu-anka-vm-clone" "anka-packer-skip-clone-with-post-processing" {
  source_vm_name = "${var.source_vm_name}"
  skip_clone     = true
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-skip-clone-with-post-processing",
  ]

  provisioner "shell" {
    inline = [
      "touch /Users/Shared/anka-packer-skip-clone"
    ]
  }

  post-processor "veertu-anka-registry-push" {
    tag    = var.tag != "" ? var.tag : "${build.OSVersion}-${build.DarwinVersion}"
    force  = var.force
    remote = var.remote
  }
}
