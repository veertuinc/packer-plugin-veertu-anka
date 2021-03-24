source "veertu-anka-vm-clone" "anka-packer-from-source-with-port-rules" {
  vm_name = "anka-packer-from-source-with-port-rules"
  source_vm_name = "anka-packer-base-macos"
  // port_forwarding_rules = [
  //   {
  //     port_forwarding_guest_port = 80
  //     port_forwarding_host_port = 12345
  //     port_forwarding_rule_name = "website"
  //   },
  //   {
  //     port_forwarding_guest_port = 8080
  //   }
  // ]
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