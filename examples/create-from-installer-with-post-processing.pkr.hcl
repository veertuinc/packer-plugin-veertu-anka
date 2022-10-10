variable "vm_name" {
  type = string
  default = "anka-packer-base-macos-post-processor"
}

variable "installer" {
  type = string
  default = "/Applications/Install macOS Big Sur.app/"
}

source "veertu-anka-vm-create" "anka-packer-base-macos-post-processor" {
  installer = "${var.installer}"
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