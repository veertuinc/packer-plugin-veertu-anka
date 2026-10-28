# Releasing

All steps for releasing are internal to Veertu and are not public.

## Official version release

1. Create version branch `release/v<version>` for both `github-repo-packer-plugin-veertu-anka` and the internal `anka-packer` repo
2. Commit and push both branches (no `VERSION` file — the branch name and publish tag carry the version)
3. Make your changes
4. Run build job in Jenkins (uses `VERSION` from `release/v<version>` branch name)
5. Run test job in Jenkins
6. If tests pass, merge version branch into `edge` (anka-packer) and `edge` (packer-plugin-veertu-anka)
7. Run release job in Jenkins (`edge` branch): set `PLUGIN_VERSION` to `v<version>` (or point `PACKER_BUILD_ARTIFACT_JOB_NAME` at `…/release/v<version>`). That job creates/pushes the GitHub tag and uploads artifacts.

## Edge continuous pre-release

`Jenkinsfile-edge` on anka-packer runs **build → test → release** and publishes a movable GitHub tag `edge` (always pre-release).

1. Create a Jenkins job with Script Path `Jenkinsfile-edge` (defaults call `build-packer-plugin/edge`, `test-packer-plugin/edge`, `release-packer-plugin/edge`).
2. Trigger that job when plugin `edge` changes (webhook, poll, or manual). anka-packer `edge` checks out plugin `edge`.
3. Build uses version `0.0.0-edge.<shortsha>` for zip names.
4. After tests succeed, release sets `PLUGIN_VERSION=edge`, force-moves tag `edge`, and uploads the build artifacts.

## Version overrides

- Build/publish: set `VERSION` or `PROJECT_VERSION` (optional leading `v`)
- Local: `make VERSION=4.7.0 go.releaser` or tag HEAD (`git tag v4.7.0`) before build
- Untagged local builds fall back to `0.0.0-dev.<shortsha>`
