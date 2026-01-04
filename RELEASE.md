# Release Process

This document describes the release process for Headjack, covering the CLI and container images.

## Overview

Headjack uses [release-please](https://github.com/googleapis/release-please) to automate releases. The system manages two independent components:

| Component | Tag Format | Changelog |
|-----------|------------|-----------|
| CLI | `v1.0.0` | `CHANGELOG.md` |
| base image | `images/base/v1.0.0` | `images/base/CHANGELOG.md` |

## Release Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Developer Workflow                             │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   1. Commit with Conventional Commits format                            │
│      └── feat: add new command                                          │
│      └── fix(images/base): update Node.js                               │
│                                                                          │
│   2. Push to master                                                      │
│                         │                                                │
│                         ▼                                                │
│   ┌─────────────────────────────────────────┐                           │
│   │     release-please.yml workflow         │                           │
│   │                                         │                           │
│   │  • Analyzes commits since last release  │                           │
│   │  • Creates/updates release PR           │                           │
│   │  • Updates CHANGELOG.md                 │                           │
│   │  • Bumps version in manifest            │                           │
│   └─────────────────────────────────────────┘                           │
│                         │                                                │
│                         ▼                                                │
│   3. Review and merge release PR                                        │
│                         │                                                │
│                         ▼                                                │
│   ┌─────────────────────────────────────────┐                           │
│   │     release-please creates:             │                           │
│   │     • Git tag (v1.0.0 or images/*/v*)   │                           │
│   │     • GitHub Release                    │                           │
│   └─────────────────────────────────────────┘                           │
│                         │                                                │
│           ┌─────────────┴─────────────┐                                 │
│           ▼                           ▼                                 │
│   ┌───────────────┐           ┌───────────────┐                         │
│   │  CLI Release  │           │ Image Release │                         │
│   │  (v* tag)     │           │ (images/*/v*) │                         │
│   └───────────────┘           └───────────────┘                         │
│           │                           │                                  │
│           ▼                           ▼                                 │
│   ┌───────────────┐           ┌───────────────┐                         │
│   │ release.yml   │           │ images.yml    │                         │
│   │ • GoReleaser  │           │ • Docker Bake │                         │
│   │ • Homebrew    │           │ • Cosign      │                         │
│   └───────────────┘           └───────────────┘                         │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Conventional Commits

All commits must follow the [Conventional Commits](https://www.conventionalcommits.org/) specification. The commit type determines version bumps:

| Commit Type | Version Bump | Example |
|-------------|--------------|---------|
| `fix:` | Patch (0.0.x) | `fix: resolve container cleanup issue` |
| `feat:` | Minor (0.x.0) | `feat: add new run command` |
| `feat!:` or `BREAKING CHANGE:` | Major (x.0.0) | `feat!: redesign CLI interface` |

### Changelog Sections

Commits are grouped into changelog sections based on type:

| Type | Section | Visible |
|------|---------|---------|
| `feat` | Features | Yes |
| `fix` | Bug Fixes | Yes |
| `perf` | Performance | Yes |
| `refactor` | Code Refactoring | Yes |
| `revert` | Reverts | Yes |
| `build` | Build System | Yes |
| `chore` | Miscellaneous | Yes |
| `docs` | Documentation | Hidden |
| `style` | Styles | Hidden |
| `test` | Tests | Hidden |
| `ci` | Continuous Integration | Hidden |

### Scoping Commits to Components

Release-please attributes commits to components based on file paths. For clarity, you can use scopes:

```bash
# CLI changes (touches Go files in root)
git commit -m "feat: add instance list command"

# Image changes (touches files in images/)
git commit -m "feat(images/base): add ripgrep to base image"
```

## CLI Releases

CLI releases are handled by [GoReleaser](https://goreleaser.com/) when a `v*` tag is pushed.

### What Gets Built

| Platform | Architecture | CGO |
|----------|--------------|-----|
| darwin (macOS) | amd64, arm64 | Enabled |
| linux | amd64, arm64 | Disabled |

### Release Artifacts

- **Binaries**: `hjk` executable for each platform
- **Archives**: `headjack_{version}_{os}_{arch}.tar.gz`
- **Checksums**: `checksums.txt` with SHA256 hashes
- **GitHub Release**: Created with changelog from commits
- **Homebrew Cask**: Updated in [GilmanLab/tap](https://github.com/GilmanLab/tap)

### Version Injection

Version information is injected at build time via ldflags:

```go
// internal/version/version.go
var (
    Version = "dev"   // Set to tag version (e.g., "1.0.0")
    Commit  = "none"  // Set to git commit SHA
    Date    = "unknown" // Set to build date
)
```

### Workflow: `.github/workflows/release.yml`

Triggers on `v*` tags and runs GoReleaser with:
- `GITHUB_TOKEN` for GitHub release creation
- `HOMEBREW_TAP_TOKEN` for Homebrew cask updates

## Image Releases

Container images are built and published when `images/*/v*` tags are pushed.

### Image Variant

| Variant | Base | Features |
|---------|------|----------|
| `base` | Ubuntu 24.04 | Dev tools, agent CLIs, version managers |

### Image Tags

Each release creates two tags:

```
ghcr.io/gilmanlab/headjack:base          # Latest
ghcr.io/gilmanlab/headjack:base-v1.0.0   # Versioned
```

### Build Features

- **Multi-architecture**: linux/amd64 and linux/arm64
- **Signing**: Keyless signing with [Cosign](https://github.com/sigstore/cosign)
- **SBOM**: SPDX JSON format, attested to the image
- **Registry**: GitHub Container Registry (ghcr.io)

### Workflow: `.github/workflows/images.yml`

Triggers on:
- `images/base/v*` tags

Jobs:
1. **lint**: Validates Dockerfiles with hadolint
2. **prepare**: Extracts variant and version from tag
3. **build**: Builds multi-arch images with Docker Bake
4. **sign**: Signs images and attests SBOMs with Cosign

## Configuration Files

### release-please-config.json

Defines components, release types, and changelog configuration:

```json
{
  "separate-pull-requests": true,
  "packages": {
    ".": { "component": "cli", "include-component-in-tag": false },
    "images/base": { "component": "images/base", "include-component-in-tag": true }
  }
}
```

### .release-please-manifest.json

Tracks current versions for each component:

```json
{
  ".": "1.0.0",
  "images/base": "1.0.0"
}
```

### .goreleaser.yaml

Configures CLI builds, archives, and Homebrew cask publishing.

### docker-bake.hcl

Defines Docker build targets and multi-platform configuration.

## Manual Release Override

To force a specific version, add `Release-As:` to a commit message:

```bash
git commit -m "feat: major redesign

Release-As: 2.0.0"
```

## Verifying Releases

### CLI Release

```bash
# Download and verify checksum
curl -LO https://github.com/GilmanLab/headjack/releases/download/v1.0.0/checksums.txt
curl -LO https://github.com/GilmanLab/headjack/releases/download/v1.0.0/headjack_1.0.0_darwin_arm64.tar.gz
sha256sum -c checksums.txt --ignore-missing

# Check version
hjk version
```

### Container Images

```bash
# Verify image signature
cosign verify ghcr.io/gilmanlab/headjack:base \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp "github.com/GilmanLab/headjack"

# Verify SBOM attestation
cosign verify-attestation ghcr.io/gilmanlab/headjack:base \
  --type spdxjson \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp "github.com/GilmanLab/headjack"
```

## Troubleshooting

### Release PR Not Created

1. Ensure commits follow Conventional Commits format
2. Check that commits touch files in the component's path
3. Verify the `release-please.yml` workflow ran successfully

### Wrong Component Gets Release

Release-please uses file paths to attribute commits. Ensure your changes are in the correct directory:
- CLI: Root Go files (`*.go`, `internal/`, `cmd/`)
- Images: `images/base/`

### Release PR Has Wrong Version

Use `Release-As: X.Y.Z` in a commit message to override automatic versioning.

### Workflow Failed After Tag

1. Check the workflow logs in GitHub Actions
2. For CLI: Verify `HOMEBREW_TAP_TOKEN` secret is set
3. For images: Check Docker build logs and registry permissions

## Secrets Required

| Secret | Used By | Purpose |
|--------|---------|---------|
| `GITHUB_TOKEN` | All workflows | GitHub API access (automatic) |
| `HOMEBREW_TAP_TOKEN` | release.yml | Push to GilmanLab/tap repository |
