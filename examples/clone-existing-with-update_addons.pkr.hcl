variable "source_vm_name" {
  type = string
  default = "anka-packer-base-macos"
}

variable "vm_name" {
  type = string
  default = "anka-packer-from-source-with-update_addons"
}

source "veertu-anka-vm-clone" "anka-packer-from-source-with-update_addons" {
  vm_name = "${var.vm_name}"
  source_vm_name = "${var.source_vm_name}"
  update_addons = true
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source-with-update_addons",
  ]

  provisioner "shell" {
    inline = [
      "echo hello world",
      "echo llamas rock"
    ]
  }
}