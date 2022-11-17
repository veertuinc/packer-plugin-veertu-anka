variable "source_vm_name" {
  type = string
  default = "anka-packer-base-macos"
}

variable "vm_name" {
  type = string
  default = "anka-packer-from-source-with-post-processing"
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
    tag = "${build.OSVersion}-${build.DarwinVersion}"
    description = "Xcode 14.1, Fastlane X.X, Go, Brew, Git"
  }
}