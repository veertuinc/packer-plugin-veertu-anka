variable "source_vm_name" {
  type = string
  default = "anka-packer-base-macos"
}

variable "vm_name" {
  type = string
  default = "anka-packer-from-source-with-labels"
}

source "veertu-anka-vm-clone" "anka-packer-from-source-with-labels" {
  vm_name = "${var.vm_name}"
  source_vm_name = "${var.source_vm_name}"

  vm_label {
    name  = "env"
    value = "ci"
  }
  vm_label {
    name  = "team"
    value = "veertu"
  }
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source-with-labels",
  ]

  provisioner "shell" {
    inline = [
      "echo hello world",
      "echo llamas rock"
    ]
  }
}
