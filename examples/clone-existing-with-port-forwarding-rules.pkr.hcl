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
  default = "anka-packer-from-source-with-port-rules"
}

source "veertu-anka-vm-clone" "anka-packer-from-source-with-port-rules" {
  vm_name = "${var.vm_name}"
  source_vm_name = "${var.source_vm_name}"
  source_vm_tag = "${var.source_vm_tag}"
  port_forwarding_rules {
    port_forwarding_guest_port = 80
    port_forwarding_host_port = 12345
    port_forwarding_rule_name = "website"
  }
  port_forwarding_rules {
    port_forwarding_guest_port = 8080
  }
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source-with-port-rules",
  ]

  provisioner "shell" {
    inline = [
      "sleep 5",
      "echo hello world",
      "echo llamas rock"
    ]
  }
}