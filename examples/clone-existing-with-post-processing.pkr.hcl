variable "source_vm_name" {
  type = string
  default = "anka-packer-base-macos"
}

variable "vm_name" {
  type = string
  default = "anka-packer-from-source-with-post-processing"
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

variables {
  OSVersion = ""
  DarwinVersion = ""
}

source "veertu-anka-vm-clone" "anka-packer-from-source-with-post-processing" {
  vm_name = "${var.vm_name}"
  source_vm_name = "${var.source_vm_name}"
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source-with-post-processing",
  ]

  post-processor "veertu-anka-registry-push" {
    tag         = var.tag != "" ? var.tag : "${build.OSVersion}-${build.DarwinVersion}"
    description = "Xcode 14.1, Fastlane X.X, Go, Brew, Git"
    force       = var.force
    remote      = var.remote
  }
}