# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# Details on using this Integration template can be found at https://github.com/hashicorp/integration-template
#
# Canonical copy for editing (integration layout expects the file next to `.web-docs/components/`; see packer-plugin-scaffolding).
# `make generate` copies this file to `.web-docs/metadata.hcl`. `readme_location` is written for that path (`./README.md` → `.web-docs/README.md`).
integration {
  name = "Anka"
  description = "This is a packer plugin for building macOS VM templates and tags using the Anka Virtualization CLI"
  identifier = "packer/veertuinc/veertu-anka"
  flags = [
    # Remove if the plugin does not conform to the HCP Packer requirements.
    #
    # Please refer to our docs if you want your plugin to be compatible with
    # HCP Packer: https://developer.hashicorp.com/packer/docs/plugins/creation/hcp-support
    # "hcp-ready",
    # This signals that the plugin is unmaintained and will eventually not be
    # working with a future version of Packer.
    #
    # On the integrations, this will end-up as an icon on the plugin's main card.
    # "archived",
  ]
  docs {
    process_docs = true
    readme_location = "./README.md"
    external_url = "https://github.com/veertuinc/packer-plugin-veertu-anka"
  }
  license {
    type = "MIT License"
    url = "https://github.com/veertuinc/packer-plugin-veertu-anka/blob/main/LICENSE"
  }
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
