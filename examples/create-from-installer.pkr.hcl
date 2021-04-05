source "veertu-anka-vm-create" "anka-packer-base-macos" {
  installer_app = "/Applications/Install macOS Big Sur.app/"
  vm_name = "anka-packer-base-macos"
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