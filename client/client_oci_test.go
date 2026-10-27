package client

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestIsOCIRemote(t *testing.T) {
	remoteCases := []struct {
		remote      string
		isOCIRemote bool
	}{
		{"oci://registry.example.com/v2/tools:xcode-26.1", true},
		{"OCI://registry.example.com/tools", true},
		{"oci://ghcr.io/homebrew/core/cmake:4.3.4", true},
		{"https://registry.example.com/v2/tools:xcode-26.1", true},
		{"http://127.0.0.1:15455/v2/veertu/", true},
		{"http://anka.registry:8089", false},
		{"http://127.0.0.1:18083", false},
		{"https://registry.example.com/tools", false},
		{"my-registry-name", false},
		{"", false},
	}
	for _, remoteCase := range remoteCases {
		t.Run(remoteCase.remote, func(t *testing.T) {
			assert.Equal(t, remoteCase.isOCIRemote, IsOCIRemote(remoteCase.remote))
		})
	}
}

func TestValidateOCIRemoteTLSOptions(t *testing.T) {
	ociRemote := "oci://registry.example.com/v2/tools"

	t.Run("OCI remote without TLS options is valid", func(t *testing.T) {
		assert.NilError(t, ValidateOCIRemoteTLSOptions(RegistryParams{Remote: ociRemote}))
	})

	t.Run("OCI remote with any TLS option is rejected", func(t *testing.T) {
		for _, registryParams := range []RegistryParams{
			{Remote: ociRemote, NodeCertPath: "/cert.pem"},
			{Remote: ociRemote, NodeKeyPath: "/key.pem"},
			{Remote: ociRemote, CaRootPath: "/ca.pem"},
			{Remote: ociRemote, IsInsecure: true},
		} {
			assert.ErrorContains(t, ValidateOCIRemoteTLSOptions(registryParams), "only for Anka Registry remotes")
		}
	})

	t.Run("Anka Registry remote keeps TLS options", func(t *testing.T) {
		assert.NilError(t, ValidateOCIRemoteTLSOptions(RegistryParams{
			Remote:       "https://anka.registry:8089",
			NodeCertPath: "/cert.pem",
			NodeKeyPath:  "/key.pem",
			CaRootPath:   "/ca.pem",
			IsInsecure:   true,
		}))
	})
}

func TestOCIPullCommandArgs(t *testing.T) {
	t.Run("full pull of a source tag", func(t *testing.T) {
		assert.DeepEqual(t,
			[]string{"pull", "--tag", "vanilla", "macos-base", "oci://registry.example.com/v2/"},
			ociPullCommandArgs(OCIPullParams{VMID: "macos-base", Remote: "oci://registry.example.com/v2/", Tag: "vanilla"}),
		)
	})

	t.Run("thin pull keeps every transform in order", func(t *testing.T) {
		assert.DeepEqual(t,
			[]string{
				"pull", "--thin",
				"--transform", "s|cmake/4.3.4|usr/local",
				"--transform", "s|bin|usr/local/bin",
				"macos-base", "oci://ghcr.io/homebrew/core/cmake:4.3.4",
			},
			ociPullCommandArgs(OCIPullParams{
				VMID:       "macos-base",
				Remote:     "oci://ghcr.io/homebrew/core/cmake:4.3.4",
				Thin:       true,
				Transforms: []string{"s|cmake/4.3.4|usr/local", "s|bin|usr/local/bin"},
			}),
		)
	})
}

func TestOCIPushCommandArgs(t *testing.T) {
	t.Run("full push with only a tag", func(t *testing.T) {
		assert.DeepEqual(t,
			[]string{"push", "--tag", "vanilla", "built-vm", "oci://registry.example.com/v2/"},
			ociPushCommandArgs(OCIPushParams{VMID: "built-vm", Remote: "oci://registry.example.com/v2/", Tag: "vanilla"}),
		)
	})

	t.Run("thin push with every option", func(t *testing.T) {
		assert.DeepEqual(t,
			[]string{
				"push", "--tag", "xcode-26.1",
				"--description", "Xcode 26.1",
				"--remote-vm", "tools",
				"--thin", "--force",
				"built-vm", "oci://registry.example.com/v2/veertu",
			},
			ociPushCommandArgs(OCIPushParams{
				VMID:        "built-vm",
				Remote:      "oci://registry.example.com/v2/veertu",
				Tag:         "xcode-26.1",
				Description: "Xcode 26.1",
				RemoteVM:    "tools",
				Thin:        true,
				Force:       true,
			}),
		)
	})
}
