# Releasing

All steps for releasing are internal to Veertu and are not public.

## Preferred: unified Jenkins job `packer-plugin-veertu-anka`

Multibranch Pipeline on the internal `anka-packer` repo, Script Path `Jenkinsfile`. One run does **docs check → build → test → GitHub release**.

### Edge continuous pre-release

1. Push to anka-packer `edge` (plugin submodule tracks plugin `edge`).
2. Job `packer-plugin-veertu-anka/edge` runs automatically (SCM trigger / scan).
3. Build uses version `0.0.0-dev` (Packer path-install only allows prerelease `dev`; commit is in binary metadata).
4. After tests succeed, the job publishes movable GitHub tag `edge` (always pre-release) and uploads the build artifacts.

### Official version release

1. Create version branch `release/v<version>` for both the plugin repo and anka-packer; commit and push.
2. Job `packer-plugin-veertu-anka/release%2Fv<version>` runs build and test.
3. If tests pass, the Console shows **Release**. Press it to publish GitHub tag `v<version>` and upload artifacts. Abort marks the job ABORTED and does **not** publish.
4. Merge the version branch into `edge` (both repos) when you want edge to include that work.

### Create the Jenkins job (once)

1. New Item → Multibranch Pipeline → name `packer-plugin-veertu-anka`.
2. Branch Sources: Bitbucket/Git `git@bitbucket.org:veertunew/anka-packer.git` (same credentials as the other packer jobs).
3. Build Configuration → Mode: by Jenkinsfile → Script Path: `Jenkinsfile`.
4. Discover `edge` and `release/*` (filter as needed).
5. Scan / save. Wire Bitbucket webhook or periodic scan like the other packer multibranch jobs.

## Legacy jobs (still available)

`build-packer-plugin`, `test-packer-plugin`, and `release-packer-plugin` (`Jenkinsfile-build` / `-test` / `-release`) remain for manual or one-off runs. Prefer `packer-plugin-veertu-anka` for normal edge and release flow.

## Version overrides

- Build/publish: set `VERSION` or `PROJECT_VERSION` (optional leading `v`)
- Local: `make VERSION=4.7.0 go.releaser` or tag HEAD (`git tag v4.7.0`) before build
- Untagged local builds fall back to `0.0.0-dev`
