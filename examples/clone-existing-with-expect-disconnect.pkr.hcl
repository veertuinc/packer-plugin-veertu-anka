variable "source_vm_name" {
  type = string
  default = "anka-packer-base-macos"
}

variable "vm_name" {
  type = string
  default = "anka-packer-from-source-with-expect-disconnect"
}

source "veertu-anka-vm-clone" "anka-packer-from-source-with-expect-disconnect" {
  vm_name = "${var.vm_name}"
  source_vm_name = "${var.source_vm_name}"
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source-with-expect-disconnect",
  ]

  provisioner "shell" {
    inline = [
      "set -x",
      "echo PRE REBOOT",
      "sleep 5",
      "sudo reboot",
      "echo SHOULD NOT SEE THIS ECHO"
    ]
    expect_disconnect = true
  }

  provisioner "shell" {
    inline = [
      "set -x",
      "echo REBOOTED"
    ]
  }
}