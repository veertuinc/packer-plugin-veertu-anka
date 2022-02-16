variable "vm_name" {
  type = string
  default = "anka-packer-base-macos-post-processor"
}

variable "installer_app" {
  type = string
  default = "/Applications/Install macOS Big Sur.app/"
}

source "veertu-anka-vm-create" "anka-packer-base-macos-post-processor" {
  installer_app = "${var.installer_app}"
  vm_name = "${var.vm_name}"
}

build {
  sources = [
    "source.veertu-anka-vm-create.anka-packer-base-macos-post-processor"
  ]
  provisioner "shell" {
    inline = [
      "echo hello world",
      "echo llamas rock"
    ]
  }
  post-processor "veertu-anka-registry-push" {
    tag = "veertu-registry-push-test"
  }
}