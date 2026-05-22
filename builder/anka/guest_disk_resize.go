package anka

// guestAPFSResizeContainerShellCommand grows the guest APFS container to fill the
// virtual disk after `anka modify set hard-drive`. Older macOS templates use
// disk0s2; newer releases report APFS container identifiers via diskutil info /.
const guestAPFSResizeContainerShellCommand = `APFS_CONTAINER=""; if diskutil info disk0s2 >/dev/null 2>&1; then APFS_CONTAINER=disk0s2; else APFS_CONTAINER=$(diskutil info / 2>/dev/null | awk '/APFS Container:/ {print $NF; exit}'); fi; if [ -z "${APFS_CONTAINER}" ]; then APFS_CONTAINER=$(diskutil info / 2>/dev/null | awk '/APFS Physical Store:/ {print $NF; exit}'); fi; if [ -z "${APFS_CONTAINER}" ]; then APFS_CONTAINER=$(diskutil apfs list 2>/dev/null | awk '/^\+-- Container / {print $3; exit}'); fi; if [ -z "${APFS_CONTAINER}" ] || ! diskutil info "${APFS_CONTAINER}" >/dev/null 2>&1; then echo "Could not determine APFS container to resize" >&2; exit 1; fi; diskutil apfs resizeContainer "${APFS_CONTAINER}" 0`
