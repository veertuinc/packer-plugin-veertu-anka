variable "source_vm_name" {
  type = string
  default = "anka-packer-base-macos"
}

variable "display_controller" {
  type = string
  default = "pg"
}

variable "vm_name" {
  type = string
  default = "anka-packer-from-source-with-display_controller"
}

source "veertu-anka-vm-clone" "anka-packer-from-source-with-display_controller" {
  vm_name = "${var.vm_name}"
  source_vm_name = "${var.source_vm_name}"
  vcpu_count = 10
  display_controller = "${var.display_controller}"
  stop_vm = true
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-from-source-with-display_controller",
  ]

}