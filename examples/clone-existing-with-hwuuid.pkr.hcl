locals {
  hw_uuid = ""
}

source "veertu-anka-vm-clone" "anka-packer-from-source-with-hwuuid" {
  vm_name = "anka-packer-from-source-with-hwuuid"
  source_vm_name = "anka-packer-base-macos"
  hw_uuid = "${local.hw_uuid}"
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source-with-hwuuid",
  ]

  provisioner "shell" {
    inline = [
      "sleep 5",
      "echo hello world",
      "echo llamas rock"
    ]
  }
}