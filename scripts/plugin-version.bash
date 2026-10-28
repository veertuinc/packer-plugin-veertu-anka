#!/usr/bin/env bash
# Print the plugin version WITHOUT a leading v.
#
# Priority:
#   1. VERSION env
#   2. PROJECT_VERSION env
#   3. Exact semver-like tag on HEAD (prefer v-prefixed)
#   4. With --dev: 0.0.0-dev (Packer plugins install only allows prerelease "dev")
#   5. Otherwise exit 1
set -euo pipefail

ALLOW_DEV=false
if [[ "${1:-}" == "--dev" ]]; then
  ALLOW_DEV=true
fi

normalize() {
  local raw="${1:-}"
  raw="$(echo "${raw}" | tr -d '[:space:]')"
  if [[ -z "${raw}" ]]; then
    return 1
  fi
  echo "${raw#v}"
}

if [[ -n "${VERSION:-}" ]]; then
  normalize "${VERSION}"
  exit 0
fi
if [[ -n "${PROJECT_VERSION:-}" ]]; then
  normalize "${PROJECT_VERSION}"
  exit 0
fi

if git rev-parse --git-dir >/dev/null 2>&1; then
  preferred=""
  fallback=""
  # shellcheck disable=SC2013
  for tag in $(git tag --points-at HEAD 2>/dev/null | grep -E '^v?[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$' || true); do
    if [[ "${tag}" == v* ]]; then
      preferred="${tag}"
      break
    fi
    if [[ -z "${fallback}" ]]; then
      fallback="${tag}"
    fi
  done
  if [[ -n "${preferred}" ]]; then
    normalize "${preferred}"
    exit 0
  fi
  if [[ -n "${fallback}" ]]; then
    normalize "${fallback}"
    exit 0
  fi

  if [[ "${ALLOW_DEV}" == true ]]; then
    echo "0.0.0-dev"
    exit 0
  fi
fi

echo "ERROR: cannot resolve plugin version. Set VERSION or PROJECT_VERSION, or tag HEAD (e.g. v4.7.0)." >&2
exit 1
