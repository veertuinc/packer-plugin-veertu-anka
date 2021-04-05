variables {
  OSVersion = ""
  DarwinVersion = ""
}

source "veertu-anka-vm-clone" "anka-packer-from-source-with-post-processing" {
  vm_name = "anka-packer-from-source-with-post-processing"
  source_vm_name = "anka-packer-base-macos"
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source-with-post-processing",
  ]

  post-processor "veertu-anka-registry-push" {
    tag = "${build.OSVersion}-${build.DarwinVersion}"
  }
}