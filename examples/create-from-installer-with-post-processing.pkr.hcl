source "veertu-anka-vm-create" "anka-packer-base-macos-post-processor" {
  installer_app = "/Applications/Install macOS Big Sur.app/"
  vm_name = "anka-packer-base-macos-post-processor"
}

build {
  sources = [
    "source.veertu-anka-vm-create.anka-packer-base-macos-post-processor"
  ]

  provisioner "shell" {
    inline = [
      "sleep 5",
      "echo hello world",
      "echo llamas rock"
    ]
  }

  post-processor "veertu-anka-registry-push" {
    tag = "veertu-registry-push-test"
  }
}