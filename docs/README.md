This is a [Packer](https://www.packer.io/) Plugin for building images that work with [Veertu's Anka macOS Virtualization tool](https://veertu.com/).

### Installation

To install this plugin, copy and paste this code into your Packer configuration, then run [`packer init`](https://www.packer.io/docs/commands/init).

```hcl
packer {
  required_plugins {
    veertu-anka = {
      version = "= 3.2.0"
      source  = "github.com/veertuinc/veertu-anka"
    }
  }
}
```

Alternatively, you can use `packer plugins install` to manage installation of this plugin.

```sh
$ packer plugins install github.com/veertuinc/veertu-anka
```

### Components
~> For use with the post-processor, it's important to use `anka registry add` to [set your default registry on the machine building your templates/tags](https://docs.veertu.com/anka/apple/command-line-reference/#registry-add).

#### Builders
- [veertu-anka-vm-clone](/packer/integrations/veertuinc/veertu-anka/latest/components/builder/clone) - Packer builder is able to clone existing Anka VM Templates for use with the [Anka Virtualization](https://veertu.com/technology/) package and the [Anka Build Cloud](https://veertu.com/anka-build/). The builder takes a source VM name, clones it, and then runs any provisioning necessary on the new VM Template before stopping or suspending it.
- [veertu-anka-vm-create- ](/packer/integrations/veertuinc/veertu-anka/latest/components/builder/create) Packer builder is able to create new Anka VM Templates for use with the
[Anka Virtualization](https://veertu.com/technology/) package and the [Anka Build Cloud](https://veertu.com/anka-build/). The builder takes the path to macOS installer .app 
and installs that macOS version inside of an Anka VM Template.

#### Post-Processors
- [veertu-anka-registry-push](/packer/integrations/veertuinc/veertu-anka/latest/components/post-processor/veertu-anka-registry-push) Packer Post Processor is able to push your created Anka VM templates to 
the [Anka Build Cloud Registry](https://veertu.com/anka-build/) through the [Anka Virtualization](https://veertu.com/technology/) package.
