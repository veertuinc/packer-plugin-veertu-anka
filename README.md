# Veertu Anka Builder (for packer.io)

The builder does not manage images. Once it creates an image, it is up to you to use it or delete it.

## Running

```bash
mkdir -p ~/.packer.d/plugins
go build -o ~/.packer.d/plugins/packer-builder-veertu-anka
packer build examples/macos-sierra.json
```

## Development

```bash
make packer-test
```

If you've already built a base macOS VM, you can use:

```bash
make packer-test SOURCE_VM_NAME=macos-10.12.3-base
```