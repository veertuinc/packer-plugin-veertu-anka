source "veertu-anka-vm-clone" "anka-packer-from-source-with-use-anka-cp" {
  vm_name = "anka-packer-from-source-with-use-anka-cp"
  source_vm_name = "anka-packer-base-macos"
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