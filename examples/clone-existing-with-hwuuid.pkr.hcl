variable "source_vm_name" {
  type = string
  default = "anka-packer-base-macos"
}

variable "hw_uuid" {
  type = string
  default = ""
}

variable "vm_name" {
  type = string
  default = "anka-packer-from-source-with-hwuuid"
}

source "veertu-anka-vm-clone" "anka-packer-from-source-with-hwuuid" {
  vm_name = "${var.vm_name}"
  source_vm_name = "${var.source_vm_name}"
  hw_uuid = "${var.hw_uuid}"
  vcpu_count = 10
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source-with-hwuuid",
  ]

  provisioner "shell" {
    inline = [
      "echo hello world",
      "echo llamas rock"
    ]
  }
}