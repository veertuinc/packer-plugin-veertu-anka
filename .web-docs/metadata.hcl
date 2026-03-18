integration {
  name = "Veertu Anka"
  description = "Packer plugin for creating and cloning Anka macOS VM templates, then pushing them to the Anka Build Cloud Registry."
  identifier = "packer/veertuinc/veertu-anka"
  flags = ["hcp-ready"]

  component {
    type = "builder"
    name = "veertu-anka-vm-clone"
    slug = "clone"
  }

  component {
    type = "builder"
    name = "veertu-anka-vm-create"
    slug = "create"
  }

  component {
    type = "post-processor"
    name = "veertu-anka-registry-push"
    slug = "veertu-anka-registry-push"
  }
}