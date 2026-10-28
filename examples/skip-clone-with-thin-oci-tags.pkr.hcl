// Requires Anka 3.10.0 or later on the host, and network access to ghcr.io.
// In-place build (no clone) that layers thin Homebrew bottles from GHCR onto
// source_vm_name, provisions, then pushes only this build's layer as a thin OCI tag.
// Because skip_clone keeps the local VM name equal to the registry template name,
// remote_vm is not needed.
//
// Typical flow:
// 1. Pull thin OCI tags onto source_vm_name (order matters; optional --transform).
// 2. Start source_vm_name, run provisioners, stop/suspend.
// 3. Push a thin OCI tag that holds only the files this build added.
//
// cmake uses --transform to flatten the bottle under /usr/local (Anka docs pattern).
// jq depends on oniguruma: pull oniguruma first, and leave both bottles un-transformed
// so Anka installs them into the Homebrew prefix (/opt/homebrew) where Mach-O
// @@HOMEBREW_PREFIX@@ / @@HOMEBREW_CELLAR@@ placeholders resolve.

variable "source_vm_name" {
  type    = string
  default = "anka-packer-base-macos"
}

variable "oci_push_remote" {
  type    = string
  default = "http://127.0.0.1:15455/v2/veertu/"
}

variable "oci_push_tag" {
  type    = string
  default = "anka-packer-skip-clone-thin-1"
}

variable "force" {
  type    = bool
  default = false
}

source "veertu-anka-vm-clone" "anka-packer-skip-clone-with-thin-oci-tags" {
  source_vm_name = var.source_vm_name
  skip_clone     = true

  // Pulled onto source_vm_name before start, in this order.
  thin_oci_pulls {
    url       = "oci://ghcr.io/homebrew/core/cmake:4.3.4"
    transform = ["s|cmake/4.3.4|usr/local"]
  }

  thin_oci_pulls {
    url = "oci://ghcr.io/homebrew/core/oniguruma:6.9.10"
  }

  thin_oci_pulls {
    url = "oci://ghcr.io/homebrew/core/jq:1.8.2"
  }

  vm_label {
    name  = "packer-flow"
    value = "skip-clone-thin-oci"
  }
}

build {
  sources = [
    "source.veertu-anka-vm-clone.anka-packer-skip-clone-with-thin-oci-tags",
  ]

  provisioner "shell" {
    inline = [
      "/usr/local/bin/cmake --version",
      "/opt/homebrew/bin/jq --version",
      "mkdir -p /Users/Shared/anka-packer-skip-clone-thin",
      "echo 'built by packer skip_clone + thin OCI' > /Users/Shared/anka-packer-skip-clone-thin/build.txt",
    ]
  }

  post-processor "veertu-anka-registry-push" {
    remote = var.oci_push_remote
    tag    = var.oci_push_tag
    thin   = true
    force  = var.force
  }
}
