package anka

import "fmt"

// guestNetworkProbeTarget is a well-known public IP used to verify default-route reachability from the macOS guest.
const guestNetworkProbeTarget = "8.8.8.8"

// guestNetworkReadinessMaxAttempts is how many 1-second intervals we try before failing the readiness check.
const guestNetworkReadinessMaxAttempts = 60

// guestNetworkReadinessShCommand returns Command for client.RunParams.
// The anka runner feeds strings.Join(Command, " ") to `sh -s` on stdin, so the script must be a single argv element.
func guestNetworkReadinessShCommand() []string {
	script := fmt.Sprintf(
		`i=0; while [ "$i" -lt %d ]; do ping -c 1 %s >/dev/null 2>&1 && exit 0; i=$((i+1)); sleep 1; done; echo "anka packer: guest network readiness check timed out after %d attempts" >&2; exit 1`,
		guestNetworkReadinessMaxAttempts,
		guestNetworkProbeTarget,
		guestNetworkReadinessMaxAttempts,
	)
	return []string{script}
}
