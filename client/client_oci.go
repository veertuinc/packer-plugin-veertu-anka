package client

import (
	"errors"
	"net/url"
	"strings"
)

// IsOCIRemote reports whether remote is an OCI registry URL, using the same rule as the Anka CLI:
// an oci:// URL, or any URL whose path contains /v2. Remote names from `anka registry add` are not URLs and return false.
func IsOCIRemote(remote string) bool {
	parsedRemote, err := url.Parse(remote)
	if err != nil || parsedRemote.Scheme == "" || parsedRemote.Host == "" {
		return false
	}
	if strings.EqualFold(parsedRemote.Scheme, "oci") {
		return true
	}
	return strings.Contains(parsedRemote.Path, "/v2")
}

// ValidateOCIRemoteTLSOptions rejects cert, key, cacert, and insecure for an OCI remote: `anka push` and `anka pull` have no such options.
func ValidateOCIRemoteTLSOptions(registryParams RegistryParams) error {
	if !IsOCIRemote(registryParams.Remote) {
		return nil
	}
	if registryParams.NodeCertPath != "" || registryParams.NodeKeyPath != "" || registryParams.CaRootPath != "" || registryParams.IsInsecure {
		return errors.New("cert, key, cacert, and insecure are only for Anka Registry remotes; remove them when remote is an OCI registry URL")
	}
	return nil
}

// OCIPullParams describes one `anka pull` from an OCI registry onto a local VM.
type OCIPullParams struct {
	VMID       string
	Remote     string
	Tag        string
	Thin       bool
	Transforms []string
}

func ociPullCommandArgs(pullParams OCIPullParams) []string {
	cmdArgs := []string{"pull"}
	if pullParams.Tag != "" {
		cmdArgs = append(cmdArgs, "--tag", pullParams.Tag)
	}
	if pullParams.Thin {
		cmdArgs = append(cmdArgs, "--thin")
	}
	for _, transform := range pullParams.Transforms {
		cmdArgs = append(cmdArgs, "--transform", transform)
	}
	return append(cmdArgs, pullParams.VMID, pullParams.Remote)
}

func (c *AnkaClient) OCIPull(pullParams OCIPullParams, outputStreamer chan string) error {
	_, err := runAnkaCommandWithProgress(outputStreamer, ociPullCommandArgs(pullParams)...)
	return err
}

// OCIPushParams describes one `anka push` of a local VM to an OCI registry.
type OCIPushParams struct {
	VMID        string
	Remote      string
	Tag         string
	Description string
	RemoteVM    string
	Thin        bool
	Force       bool
}

func ociPushCommandArgs(pushParams OCIPushParams) []string {
	cmdArgs := []string{"push", "--tag", pushParams.Tag}
	if pushParams.Description != "" {
		cmdArgs = append(cmdArgs, "--description", pushParams.Description)
	}
	if pushParams.RemoteVM != "" {
		cmdArgs = append(cmdArgs, "--remote-vm", pushParams.RemoteVM)
	}
	if pushParams.Thin {
		cmdArgs = append(cmdArgs, "--thin")
	}
	if pushParams.Force {
		cmdArgs = append(cmdArgs, "--force")
	}
	return append(cmdArgs, pushParams.VMID, pushParams.Remote)
}

func (c *AnkaClient) OCIPush(pushParams OCIPushParams, outputStreamer chan string) error {
	_, err := runAnkaCommandWithProgress(outputStreamer, ociPushCommandArgs(pushParams)...)
	return err
}
