variable "vm_name" {
  type = string
  default = "anka-packer-base-macos"
}

variable "installer" {
  type = string
  default = "/Applications/Install macOS Big Sur.app/"
}

variable "vcpu_count" {
  type = string
  default = ""
}

source "veertu-anka-vm-create" "base" {
  installer = "${var.installer}"
  vm_name = "${var.vm_name}"
  vcpu_count = "${var.vcpu_count}"
}

build {
  sources = [
    "source.veertu-anka-vm-create.base"
  ]

  provisioner "shell" {
    inline = [
      "echo hello world",
      "echo llamas rock"
    ]
  }
}