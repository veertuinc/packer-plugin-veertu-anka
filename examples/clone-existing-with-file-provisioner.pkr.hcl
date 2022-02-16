variable "source_vm_name" {
  type = string
  default = "anka-packer-base-macos"
}

variable "vm_name" {
  type = string
  default = "anka-packer-from-source"
}

variable "host_path" {
  type = string
  default = "/tmp/"
}

source "veertu-anka-vm-clone" "anka-packer-from-source" {
  vm_name = "${var.vm_name}"
  source_vm_name = "${var.source_vm_name}"
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source",
  ]
  provisioner "file" {
    destination = "/private/tmp/"
    source      = "${var.host_path}/examples/ansible"
  }
  provisioner "shell" {
    inline = [
      "[[ ! -d /tmp/ansible ]] && exit 100",
      "touch /tmp/ansible/test1"
    ]
  }
  provisioner "file" {
    destination = "./"
    direction   = "download"
    source      = "/private/tmp/ansible/test1"
  }
  provisioner "shell-local" {
    inline = [
      "[[ ! -f ./test1 ]] && exit 200",
      "rm -f ./test1"
    ]
  }
}