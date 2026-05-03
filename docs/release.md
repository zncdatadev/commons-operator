# Release Guide

This document describes the standard release process for commons-operator.

## Overview

The release process follows a branch-based workflow:

- **Main branch** (`main`): The default development branch where new features and bug fixes are merged.
- **Release branch** (`release-x.y`): A long-lived branch for a minor version series (e.g., `release-0.4`). Created from `main` and used to stabilize and release versions within that series.

## Release Process

### 1. Create Release Branch (if not exists)

Create a release branch from the latest `main`:

```bash
git checkout main
git pull --rebase upstream main
git checkout -b release-0.x
git push upstream release-0.x
```

### 2. Prepare Version Changes

Create a local branch for version changes and update all relevant files:

```bash
git checkout main
git pull --rebase upstream main
git checkout -b bump/release-x.y.z
```

#### Files to Update

| File | Field | Description |
|------|-------|-------------|
| `Makefile` | `VERSION` | Application version (e.g., `VERSION ?= 0.4.0`) |
| `deploy/helm/commons-operator/Chart.yaml` | `version` | Helm chart version |
| `deploy/helm/commons-operator/Chart.yaml` | `appVersion` | Helm chart app version |

> **Tip:** You can refer to the previous version's changes on the release branch (e.g., `git diff v0.3.0..release-0.3`) to see exactly which files were modified.

### 3. Submit and Merge PR

Push the version change branch and create a Pull Request targeting the release branch:

```bash
git push origin bump/release-x.y.z
```

Create a PR from `bump/release-x.y.z` to `release-0.x`. Wait for CI to pass and code review before merging.

### 4. Tag and Publish

After the PR is merged, tag the release and push to trigger the automated release workflow:

```bash
git checkout release-0.x
git pull --rebase upstream release-0.x
git tag vx.y.z upstream/release-0.x
git push upstream vx.y.z
```

This triggers the [Release workflow](.github/workflows/release.yml) which runs the following jobs:

- **Markdown Lint** — Lints markdown files under `docs/` and `README.*.md`
- **Golang Lint** — Runs golangci-lint
- **Golang Test** — Runs unit tests
- **E2E Test** — Runs end-to-end tests across multiple Kubernetes versions (1.33, 1.34, 1.35)
- **Chainsaw Test** — Runs Chainsaw E2E tests across multiple Kubernetes versions
- **Release Image** — Builds and pushes multi-arch Docker image to `quay.io/zncdatadev/commons-operator:<version>`, and signs the image with Cosign
- **Chart Lint & Test** — Validates and tests the Helm chart
- **Release Chart** — Publishes the Helm chart to `quay.io/kubedoopcharts/commons-operator:<version>` (OCI registry) and updates the [kubedoop-helm-charts](https://github.com/zncdatadev/kubedoop-helm-charts) index

## Versioning Convention

commons-operator follows [Semantic Versioning](https://semver.org/):

- **Patch** (x.y.Z): Bug fixes, no API changes
- **Minor** (x.Y.z): New features, backward-compatible API changes
- **Major** (X.y.z): Breaking API changes

## Example

Here is an example of releasing version `0.4.0` on the `release-0.4` branch:

```bash
# Step 1: Create release branch (skip if already exists)
git checkout main
git pull --rebase upstream main
git checkout -b release-0.4
git push upstream release-0.4

# Step 2: Prepare version changes
git checkout main
git pull --rebase upstream main
git checkout -b bump/release-0.4.0

# Update Makefile VERSION
sed -i 's/VERSION ?= 0.0.0-dev/VERSION ?= 0.4.0/' Makefile

# Update Chart.yaml
sed -i 's/version: 0.0.0-dev/version: 0.4.0/' deploy/helm/commons-operator/Chart.yaml
sed -i 's/appVersion: 0.0.0-dev/appVersion: 0.4.0/' deploy/helm/commons-operator/Chart.yaml

git add -A
git commit -m "release: 0.4.0"
git push origin bump/release-0.4.0

# Step 3: Create PR to release-0.4, review and merge

# Step 4: Tag and publish
git checkout release-0.4
git pull --rebase upstream release-0.4
git tag v0.4.0 upstream/release-0.4
git push upstream v0.4.0
```

## Troubleshooting

### Chart release failed

If the `chart-lint-test` or `release-chart` job fails, check the workflow logs for details. Common issues include:

- **CRDs out of sync**: Run `make manifests` and `make helm-crd-sync` to regenerate CRDs, then commit the changes.
- **Chart YAML validation errors**: Ensure `Chart.yaml` has correct `version` and `appVersion` fields.
- **Previous tag not found**: For the first release on a new release branch, the workflow automatically detects this and marks all charts as changed.

### Force re-trigger a release

If the release workflow fails and you need to re-trigger after fixing the issue, delete and re-create the tag:

```bash
git push upstream :refs/tags/vx.y.z
git tag vx.y.z upstream/release-0.x
git push upstream vx.y.z
```
