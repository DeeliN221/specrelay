# Release automation

## Purpose

Publish tested, complete, verifiable SpecRelay releases from version tags. The [Chinese specification](../../zh-CN/specs/release-automation/spec.md) records the earlier wording.

### Requirement: Release trigger and notes

Pushing a `v*` tag MUST start the GitHub Actions Release workflow. The tag identifies the release version. The release body MUST use the matching section in `CHANGELOG.md`, falling back to `Unreleased`. The extraction script also accepts the translated legacy heading. A release without usable notes MUST fail rather than publish an empty body.

### Requirement: Test before publication

Both CI and Release MUST build a local SpecRelay binary and run `go test ./...`. A failed test MUST prevent release packaging and publication. README MUST document the same local reproduction commands used by GitHub Actions.

### Requirement: Complete build matrix

The release MUST build Linux, macOS, and Windows for both amd64 and arm64. Unix archives contain `specrelay`; Windows archives contain `specrelay.exe`. Binaries MUST embed the release version. Module paths and linker symbols MUST use `github.com/DeeliN221/specrelay` consistently.

### Requirement: Asset integrity

A release MUST contain six platform/architecture archives, `sha256sums.txt`, and the complete `CHANGELOG.md`. All builds MUST succeed before publication. The checksum manifest MUST cover every binary archive so users and the updater can verify downloads.

### Requirement: Existing release updates

Publication MUST use the workflow's checked-out version and artifacts rather than stale local outputs. Updating release assets must retain the complete platform matrix and matching checksum manifest.

### Requirement: Local verification

The release contract is covered by `tests/specs/release-automation/test_monorepo_cli_release_contract.sh` and the Go suite. Release-note extraction MUST be testable with a temporary changelog containing an exact version section and an Unreleased fallback, without depending on historical release entries.
