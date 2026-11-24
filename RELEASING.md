# Releasing

All steps for releasing are internal to Veertu and are not public.

The wrapper and Jenkinsfile are [veertuinc/anka-packer-wrapper](https://github.com/veertuinc/anka-packer-wrapper), branch `edge`. The plugin submodule is `veertuinc/packer-plugin-veertu-anka`, branch `edge`. `anka-packer-test` is the private test suite.

## Jenkins job `packer-plugin-veertu-anka`

One Pipeline job. It is not a multibranch job. SCM is `veertuinc/anka-packer-wrapper` branch `edge`. Script Path is `Jenkinsfile`.

One run does **docs check → resolve release → build → test → GitHub release**.

The job checks out plugin `edge`. It reads version tags on that commit. A version tag is `vMAJOR.MINOR.PATCH` with no suffix. The tag named `edge` is ignored.

Required GitHub release assets:

- darwin arm64 zip
- darwin amd64 zip
- linux amd64 zip
- `SHA256SUMS`
- `SHA256SUMS.sig`

| Plugin `edge` HEAD | Build version | After tests pass |
| --- | --- | --- |
| One version tag, and any required asset is missing | `X.Y.Z` | Create the GitHub release when it is missing. Upload each missing asset. Full release. The git tag stays on the commit you pushed. |
| Any other commit | `0.0.0-dev` | Replace the GitHub pre-release `edge` and move git tag `edge` to this commit. |

The job checks GitHub again after the tests. If every required asset is already on the version release, the job replaces the `edge` pre-release with the artifacts from this run.

A failed docs check, build, or test publishes nothing. A version release that is missing a required asset stays unreleased. The next green run uploads the missing files.

`SKIP_TESTS` skips the test stages. `SKIP_RELEASE` skips notarization and GitHub publish.

A push to `anka-packer-wrapper` `edge` can start the job. A person can also start it. A tag push on the plugin repo does not start the job. The version tag must already be on plugin `origin/edge` HEAD when the job runs. A tag on an older commit publishes the `edge` pre-release.

More than one version tag on that commit fails the job before the build.

`Notify Integration Release (Tag)` still runs when the `v*` tag is pushed. That is before this job uploads artifacts.

## Jenkins item change

Remove the multibranch job `packer-plugin-veertu-anka`. Create one Pipeline job with the same name.

- SCM: `git@github.com:veertuinc/anka-packer-wrapper.git`, branch `edge`.
- Script Path: `Jenkinsfile`.
- Do not discover `release/*`.
- Do not create `packer-plugin-veertu-anka-release`.

Do this in Jenkins after `edge` contains this `Jenkinsfile`.

## Version release

1. Push the plugin commit to `edge`.
2. Tag that same commit, for example `v4.7.1`, and push the tag.
3. Start `packer-plugin-veertu-anka`, or push `anka-packer-wrapper` `edge` so the job starts.
4. When the tests pass, the job publishes `v4.7.1` because the GitHub release has no required assets.

A security patch uses this same path. Do not create a `release/vX.Y.Z` branch. Do not run `./scripts/create-release-branch.bash`.

## Version overrides

- Build/publish: the job sets `VERSION` from the decision file.
- Local: `make VERSION=4.7.0 go.releaser` or tag HEAD (`git tag v4.7.0`) before build.
- Untagged local builds fall back to `0.0.0-dev`.
