# Releasing

All steps for releasing are internal to Veertu and are not public.

`anka-packer` holds the Jenkins `Jenkinsfile` and the plugin submodule. `anka-packer-test` is the private test suite. Both are moving from Bitbucket to private GitHub repos:

| Role | Bitbucket (live until the migration finishes) | GitHub destination | Branch |
| --- | --- | --- | --- |
| Wrapper and Jenkinsfile | `veertunew/anka-packer` | `veertuinc/anka-packer` | `edge` |
| Private test suite | `veertunew/anka-packer-test` | `veertuinc/anka-packer-test` | `master` |

Until Jenkins uses the GitHub remotes, the Bitbucket remotes stay the live SCM.

## Preferred: unified Jenkins job `packer-plugin-veertu-anka`

Multibranch Pipeline on `anka-packer`, Script Path `Jenkinsfile`. One run does **docs check → build → test → GitHub release**.

### Edge continuous pre-release

1. Push to anka-packer `edge` (plugin submodule tracks plugin `edge`).
2. Job `packer-plugin-veertu-anka/edge` runs automatically (SCM trigger / scan).
3. Build uses version `0.0.0-dev` (Packer path-install only allows prerelease `dev`; commit is in binary metadata).
4. After tests succeed, the job publishes movable GitHub tag `edge` (always pre-release) and uploads the build artifacts. Set job param **`SKIP_RELEASE`** to stop after build + test with no publish.

### Official version release

This path is for a human feature release. A security patch does not use it. Follow [Automation](#automation). Do not create `release/vX.Y.Z` for a security patch.

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

## Automation

The bot follows only this section for a security patch.

A security patch is cut from `edge` on this public repo and from `edge` on `anka-packer`. There is no `release/vX.Y.Z` branch. Do not run `./scripts/create-release-branch.bash`. Do not merge a release branch back. The commit is already on `edge`.

The next security version is the latest full-release `v*` tag plus one patch. Ignore the movable pre-release tag `edge` when you choose the number. On 2026-11-09 the latest full release is `v4.7.0`, so the version string is `4.7.1` and the tag is `v4.7.1`.

1. Pull-request check `Check PR / goreleaser` must have passed on the `edge` commit before you start Jenkins. No GitHub workflow runs on push to `edge`. Do not wait for one.
2. Do not trigger `packer-plugin-veertu-anka/edge`. That job publishes the movable pre-release tag `edge`. Do not move tag `edge`.
3. Do not dispatch `Notify Integration Release (Manual)`. That workflow is for documentation only.
4. Do not trigger `packer-plugin-veertu-anka/release%2FvX.Y.Z`. That job exists only for a `release/vX.Y.Z` branch, and a security patch has no such branch.
5. Read private `veertuinc/anka-packer` (branch `edge`) and `veertuinc/anka-packer-test` (branch `master`). They are migrating from Bitbucket `veertunew/anka-packer` and `veertunew/anka-packer-test`. If the bot cannot read `veertuinc/anka-packer`, stop and notify. Do not create the GitHub Release from this public repo to get around that.
6. Trigger Jenkins job `packer-plugin-veertu-anka-release` (`https://jenkins.veertu.com/job/packer-plugin-veertu-anka-release/`). Set `VERSION` to `X.Y.Z` with no leading `v`. Leave `SKIP_RELEASE` unset. If that job does not exist, stop and notify. Do not substitute the `edge` job or a `release/v*` job.
7. Wait for docs check, build, and test. The test suite comes from `anka-packer-test`. If any of those fail, stop. Do not ask Nathan to submit **Release**.
8. When the build is waiting for input, post in Slack channel `C01SDPZ2R6Z`. Mention Nathan as `<@U01DZ9N8UJV>` and ask him to submit the **Release** input on that build. Include the build URL and the version. Do not submit the input. Do not abort the build. Abort does not publish.
9. Poll the build after that post. When it finishes, verify the result. `SUCCESS` publishes tag `vX.Y.Z` on this repo as a full GitHub Release (not a pre-release) and uploads the artifacts. `FAILURE`, `UNSTABLE`, or `ABORTED` does not publish. Stop and notify. Do not push the tag yourself. Confirm that tag `edge` did not move.
10. Workflow `Notify Integration Release (Tag)` runs from the `v*` tag push. Report its result. Do not re-tag if it fails.

### Jenkins job to create

Job `packer-plugin-veertu-anka-release` does not exist yet. The multibranch item `packer-plugin-veertu-anka` is a different job. Its `edge` branch publishes tag `edge`. Its `release/v*` branch needs a release branch. Neither one is the security-patch job.

Create one Pipeline job:

1. Name: `packer-plugin-veertu-anka-release`.
2. SCM: `veertuinc/anka-packer`, branch `edge`. Until that repo is the live remote, use `git@bitbucket.org:veertunew/anka-packer.git`, branch `edge`.
3. Script Path: `Jenkinsfile` in `anka-packer`. The `edge` branch of that file today publishes tag `edge` and does not wait for **Release**. Change it for this job: wait for **Release**, then publish tag `v${VERSION}` as a full release. Do not move tag `edge`.
4. Parameters: `VERSION` (string, `X.Y.Z`, no `v`) and `SKIP_RELEASE` (boolean, default false). `SKIP_RELEASE` skips confirm and publish. Build and test still run.
5. Tests clone `veertuinc/anka-packer-test` (branch `master`). Until that repo is the live remote, clone Bitbucket `veertunew/anka-packer-test`, branch `master`.
6. The **Release** input stays manual. The bot triggers the job and verifies the result. Nathan submits the input.

When the GitHub migration is done, point the existing multibranch job `packer-plugin-veertu-anka` at `veertuinc/anka-packer`. Leave its `edge` branch as the pre-release path only.

## Version overrides

- Build/publish: set `VERSION` or `PROJECT_VERSION` (optional leading `v`)
- Local: `make VERSION=4.7.0 go.releaser` or tag HEAD (`git tag v4.7.0`) before build
- Untagged local builds fall back to `0.0.0-dev`
