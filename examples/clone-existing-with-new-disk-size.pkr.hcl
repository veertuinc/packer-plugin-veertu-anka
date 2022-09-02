variable "source_vm_name" {
  type = string
  default = "anka-packer-base-macos"
}

variable "source_vm_tag" {
  type = string
  default = "latest"
}

variable "vm_name" {
  type = string
  default = "anka-packer-from-source-with-new-disk-size"
}

source "veertu-anka-vm-clone" "anka-packer-from-source-with-new-disk-size" {
  vm_name = "${var.vm_name}"
  source_vm_name = "${var.source_vm_name}"
  source_vm_tag = "${var.source_vm_tag}"
  vcpu_count = 8
  ram_size = "10G"
  disk_size = "200G"
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source-with-new-disk-size",
  ]

  provisioner "shell" {
    inline = [
      "echo hello world",
      "echo llamas rock"
    ]
  }
}