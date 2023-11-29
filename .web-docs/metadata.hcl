# For full specification on the configuration of this file visit:
# https://github.com/hashicorp/integration-template#metadata-configuration
integration {
  name = "Anka"
  description = "A plugin for building images that work with Veertu's Anka macOS Virtualization tool."
  identifier = "packer/veertuinc/veertu-anka"
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
