# For full specification on the configuration of this file visit:
# https://github.com/hashicorp/integration-template#metadata-configuration
integration {
  name = "Anka"
  description = "TODO"
  identifier = "packer/BrandonRomano/veertu-anka"
  component {
    type = "builder"
    name = "Anka VM Create"
    slug = "vm-create"
  }
  component {
    type = "builder"
    name = "Anka VM Clone"
    slug = "vm-clone"
  }
  component {
    type = "post-processor"
    name = "Anka Build Cloud Registry Push"
    slug = "anka-registry-push"
  }
}
