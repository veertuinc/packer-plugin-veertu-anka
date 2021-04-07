variable "vm_name" {
  type = string
  default = "anka-packer-base-macos"
}

source "veertu-anka-vm-create" "anka-packer-base-macos" {
  installer_app = "/Applications/Install macOS Big Sur.app/"
  vm_name = "${var.vm_name}"
}

build {
  sources = [
    "source.veertu-anka-vm-create.anka-packer-base-macos"
  ]

  provisioner "shell" {
    inline = [
      "sleep 5",
      "echo hello world",
      "echo llamas rock"
    ]
  }
}