variable "vm_name" {
  type = string
  default = "anka-packer-base-macos"
}

variable "installer_app" {
  type = string
  default = "/Applications/Install macOS Big Sur.app/"
}

variable "vcpu_count" {
  type = string
  default = ""
}

source "veertu-anka-vm-create" "base" {
  installer_app = "${var.installer_app}"
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