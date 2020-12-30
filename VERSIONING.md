# How our versioning works

> We do **not** use a strict semantic versioning pattern

We use a well known pattern: MAJOR.MINOR.PATCH

MAJOR changes are for versions that break backwards compatibility
MINOR are for new features and bug fixes, while maintaining backwards compatibility
PATCH are for bug fixes only, while maintaining backwards compatibility

For some projects, we add a short git-sha to the version string, denoted with a '-' (ex. 3.1.0-ab2d12).
This appendix has no meaning for precedence or does not mean anything regarding changes implemented.