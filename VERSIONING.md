# How our versioning works

> We do **not** use a strict semantic versioning pattern

We use a well known pattern: MAJOR.MINOR.PATCH

MAJOR changes are for versions that break backwards compatibility
MINOR are for new features and bug fixes, while maintaining backwards compatibility
PATCH are for bug fixes only, while maintaining backwards compatibility

For some projects, we add a short git-sha to the version string, denoted with a '-' (ex. 3.1.0-ab2d12).
This appendix has no meaning for precedence or does not mean anything regarding changes implemented.

## Source of the version

There is no committed `VERSION` file. At build and publish time the version comes from, in order:

1. `VERSION` or `PROJECT_VERSION` environment variable (optional leading `v`)
2. An exact semver-like git tag on `HEAD` (prefer `vMAJOR.MINOR.PATCH`)
3. For local/dev/edge builds only: `0.0.0-dev` (Packer `plugins install --path` rejects any other prerelease; the git commit is passed via ldflags)

Official GitHub releases use a tag such as `v4.7.0`. Jenkins release creates that tag when it publishes.
