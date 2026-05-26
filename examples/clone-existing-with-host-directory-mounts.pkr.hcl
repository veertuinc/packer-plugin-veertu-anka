variable "source_vm_name" {
  type = string
  default = "anka-packer-base-macos"
}

variable "source_vm_tag" {
  type = string
  default = "latest"
}

variable "vm_name" {
  type = string
  default = "anka-packer-from-source-with-host-directory-mounts"
}

variable "host_path" {
  type = string
}

source "veertu-anka-vm-clone" "anka-packer-from-source-with-host-directory-mounts" {
  vm_name = "${var.vm_name}"
  source_vm_name = "${var.source_vm_name}"
  source_vm_tag = "${var.source_vm_tag}"
  host_directory_mounts {
    host_path = "${var.host_path}"
    guest_folder_name = "packer-mount"
  }
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source-with-host-directory-mounts",
  ]

  provisioner "shell" {
    inline = [
      "echo hello world",
      "echo llamas rock"
    ]
  }
}
