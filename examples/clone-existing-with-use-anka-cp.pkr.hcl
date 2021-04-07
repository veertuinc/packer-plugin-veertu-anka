variable "source_vm_name" {
  type = string
  default = "anka-packer-base-macos"
}

variable "vm_name" {
  type = string
  default = "anka-packer-from-source-with-use-anka-cp"
}

source "veertu-anka-vm-clone" "anka-packer-from-source-with-use-anka-cp" {
  vm_name = "${var.vm_name}"
  source_vm_name = "${var.source_vm_name}"
  use_anka_cp = true
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source-with-use-anka-cp",
  ]

  provisioner "shell" {
    inline = [
      "sleep 5",
      "echo hello world",
      "echo llamas rock"
    ]
  }
}