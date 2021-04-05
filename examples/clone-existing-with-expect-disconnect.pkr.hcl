source "veertu-anka-vm-clone" "anka-packer-from-source-with-expect-disconnect" {
  vm_name = "anka-packer-from-source-with-expect-disconnect"
  source_vm_name = "anka-packer-base-macos"
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source-with-expect-disconnect",
  ]

  provisioner "shell" {
    inline = [
      "set -x",
      "echo PRE REBOOT",
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