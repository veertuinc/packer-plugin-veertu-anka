# Veertu Anka Builder (for packer.io)

The builder does not manage images. Once it creates an image, it is up to you to use it or delete it.

## Running

```bash
mkdir -p ~/.packer.d/plugins
go build -o ~/.packer.d/plugins/packer-builder-veertu-anka
packer build examples/macos-sierra.json
```
