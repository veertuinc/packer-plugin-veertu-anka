variable "source_vm_name" {
  type = string
  default = "anka-packer-base-macos"
}

variable "vm_name" {
  type = string
  default = "anka-packer-from-source"
}

source "veertu-anka-vm-clone" "anka-packer-from-source" {
  vm_name = "${var.vm_name}"
  source_vm_name = "${var.source_vm_name}"
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source",
  ]

  provisioner "shell-local" {
    command = "touch /tmp/test.file"
  }

  provisioner "file" {
    source      = "/tmp/test.file"
    destination = "/Users/anka/test1.file"
  }
  
  provisioner "shell" {
    inline = [
      "ls -laht /Users/anka/",
      "ls -laht /Users/anka/test1.file"
    ]
  }
}