source "veertu-anka-vm-clone" "anka-packer-from-source" {
  vm_name = "anka-packer-from-source"
  source_vm_name = "anka-packer-base-macos"
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source",
  ]
}