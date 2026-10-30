# Releasing

All steps for releasing are internal to Veertu and are not public.

## Preferred: unified Jenkins job `packer-plugin-veertu-anka`

Multibranch Pipeline on the internal `anka-packer` repo, Script Path `Jenkinsfile`. One run does **docs check → build → test → GitHub release**.

### Edge continuous pre-release

1. Push to anka-packer `edge` (plugin submodule tracks plugin `edge`).
2. Job `packer-plugin-veertu-anka/edge` runs automatically (SCM trigger / scan).
3. Build uses version `0.0.0-dev` (Packer path-install only allows prerelease `dev`; commit is in binary metadata).
4. After tests succeed, the job publishes movable GitHub tag `edge` (always pre-release) and uploads the build artifacts. Set job param **`SKIP_RELEASE`** to stop after build + test with no publish.

### Official version release

Jenkins maps the anka-packer branch name to the same plugin branch name (`release/v<version>` on both). Create those branches first, then let the job publish.

#### 1. Create the release branches

Use the same version string on both repos (example: `4.7.0` → branch `release/v4.7.0`). From a clean anka-packer checkout on disk:

```bash
./scripts/create-release-branch.bash 4.7.0
# Local branches only:
# ./scripts/create-release-branch.bash 4.7.0 --no-push
```

The script refreshes `edge` on the plugin and anka-packer, creates `release/v<version>` on both, points the submodule at the plugin tip, and pushes (unless `--no-push`).

#### 2. Build, test, and publish

1. Job `packer-plugin-veertu-anka/release%2Fv<version>` runs docs check → build → test (SCM trigger / multibranch scan).
2. If tests pass, the Console shows **Release**. Press it to publish GitHub tag `v<version>` as a **full release** (not pre-release) and upload artifacts. The release body lists commits since the previous `v*` tag. Abort marks the job ABORTED and does **not** publish. Set job param **`SKIP_RELEASE`** to skip the confirm + publish stages (build + test only).
3. A human may edit the GitHub release notes after publish if the auto commit list needs more detail.
4. Merge `release/v<version>` into `edge` (both repos) when you want edge to include that work.

## Version overrides

- Build/publish: set `VERSION` or `PROJECT_VERSION` (optional leading `v`)
- Local: `make VERSION=4.7.0 go.releaser` or tag HEAD (`git tag v4.7.0`) before build
- Untagged local builds fall back to `0.0.0-dev`
