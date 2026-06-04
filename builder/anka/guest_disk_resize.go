package anka

// guestAPFSResizeContainerShellCommand grows the guest APFS container to fill the
// virtual disk after `anka modify set hard-drive`. The container is resolved from
// diskutil info on the root volume, with diskutil apfs list as a fallback.
const guestAPFSResizeContainerShellCommand = `DISKUTIL_INFO=$(diskutil info / 2>/dev/null); APFS_CONTAINER=$(echo "${DISKUTIL_INFO}" | awk '/APFS Container:/ {print $NF; exit}'); if [ -z "${APFS_CONTAINER}" ]; then APFS_CONTAINER=$(echo "${DISKUTIL_INFO}" | awk '/APFS Physical Store:/ {print $NF; exit}'); fi; if [ -z "${APFS_CONTAINER}" ]; then APFS_CONTAINER=$(diskutil apfs list 2>/dev/null | awk '/^\+-- Container / {print $3; exit}'); fi; if [ -z "${APFS_CONTAINER}" ] || ! diskutil info "${APFS_CONTAINER}" >/dev/null 2>&1; then echo "Could not determine APFS container to resize" >&2; exit 1; fi; diskutil apfs resizeContainer "${APFS_CONTAINER}" 0`
