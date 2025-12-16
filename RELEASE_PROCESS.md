# Release Process for Terraform Provider Platform

## Overview

The `releasePlatformProvider.sh` script automates the process of creating a new release for the terraform-provider-platform.

## Prerequisites

- Clean working tree (no uncommitted changes)
- Access to push to the repository
- Git configured with appropriate credentials

## Usage

### Interactive Mode (Recommended)

```bash
./releasePlatformProvider.sh
```

The script will:
1. Fetch and display the latest version from GitHub
2. Prompt you to enter the new version number
3. Ask for confirmation at each step

### Non-Interactive Mode

For CI/CD or automation:

```bash
NEW_VERSION=2.2.8 ./releasePlatformProvider.sh -y
```

or

```bash
export NEW_VERSION=2.2.8
./releasePlatformProvider.sh -y
```

The `-y` flag automatically answers "yes" to all prompts.

## What the Script Does

The script performs the following steps:

1. **Version Check**: Fetches the latest stable version from GitHub
2. **Input Validation**: Validates the new version follows SemVer (e.g., 2.2.8)
3. **Safety Checks**:
   - Ensures working tree is clean
   - Verifies the tag doesn't already exist
4. **Git Workflow**:
   - Checks out the default branch (main)
   - Pulls the latest code
   - Creates a new release branch (e.g., `v2.2.8`)
   - Pushes the branch to origin
   - Creates a new tag (e.g., `v2.2.8`)
   - Pushes the tag to origin

## Version Format

Versions must follow SemVer format: `MAJOR.MINOR.PATCH`

Examples:
- ✅ `2.2.8`
- ✅ `v2.2.8` (will be normalized to `v2.2.8`)
- ✅ `3.0.0`
- ❌ `2.2` (missing patch version)
- ❌ `2.2.8-beta` (pre-release versions not supported by this script)

## Release Workflow

Once the tag is pushed, the GitHub Actions workflow (`.github/workflows/release.yml`) will:

1. Trigger automatically on tag push
2. Build the provider for multiple platforms
3. Sign the release with GPG
4. Create a GitHub release
5. Upload artifacts to the release
6. Publish to Terraform Registry and OpenTofu Registry

## Example Session

```bash
$ ./releasePlatformProvider.sh

--- Fetching Latest Stable Provider Versions ---
Latest version for terraform-provider-platform: v2.2.7
-------------------------------------

Using provider: terraform-provider-platform
Please enter the new version number (e.g., 1.2.3): 2.2.8

--- Starting release process for provider 'terraform-provider-platform' and version v2.2.8 ---

About to checkout branch 'main'...
Proceed to checkout 'main'? (y/n) y

About to pull latest code from 'main'...
Proceed to pull from 'main'? (y/n) y

About to create and checkout new release branch: v2.2.8...
Proceed to create branch 'v2.2.8'? (y/n) y

About to push new branch to origin: v2.2.8...
Proceed to push branch 'v2.2.8' to origin? (y/n) y

About to create new tag: v2.2.8...
Proceed to create tag 'v2.2.8'? (y/n) y

About to push new tag to origin: v2.2.8...
Proceed to push tag 'v2.2.8' to origin? (y/n) y

--- Release process completed successfully for terraform-provider-platform! ---
```

## Pre-Release Checklist

Before running the release script:

1. **Update CHANGELOG.md**
   - Add new version header with date and tested versions
   - Document all changes (features, bug fixes, breaking changes)
   - Include issue/PR references

2. **Update Documentation**
   - Run `make generate` to regenerate documentation
   - Verify examples are up to date

3. **Run Tests**
   ```bash
   make test
   make acceptance
   ```

4. **Verify Build**
   ```bash
   make build
   ```

## Troubleshooting

### Working Tree Has Uncommitted Changes

**Error**: "Your working tree has uncommitted changes."

**Solution**: 
- Commit or stash your changes
- Or answer "y" when prompted to proceed anyway (not recommended)

### Tag Already Exists

**Error**: "Tag v2.2.8 already exists locally or on origin."

**Solution**:
- Choose a different version number
- Or delete the existing tag if it was created in error:
  ```bash
  git tag -d v2.2.8
  git push origin :refs/tags/v2.2.8
  ```

### Permission Denied

**Error**: Unable to push to origin

**Solution**:
- Verify you have push access to the repository
- Check your Git credentials
- Ensure you're authenticated with GitHub

## Manual Release (Alternative)

If you prefer to do it manually without the script:

```bash
# 1. Checkout and update main branch
git checkout main
git pull --ff-only

# 2. Create release branch
git checkout -b v2.2.8

# 3. Push branch
git push -u origin v2.2.8

# 4. Create and push tag
git tag v2.2.8
git push origin tag v2.2.8
```

## Post-Release

After the release is created:

1. Monitor the GitHub Actions workflow for successful completion
2. Verify the release appears on the [Releases page](https://github.com/jfrog/terraform-provider-platform/releases)
3. Verify the provider is available on:
   - [Terraform Registry](https://registry.terraform.io/providers/jfrog/platform)
   - [OpenTofu Registry](https://registry.opentofu.org/providers/jfrog/platform)
4. Update documentation if needed
5. Announce the release to stakeholders

## Versioning Guidelines

This project follows [Semantic Versioning](https://semver.org/):

- **MAJOR**: Incompatible API changes, breaking changes
- **MINOR**: New features, backwards-compatible additions
- **PATCH**: Bug fixes, backwards-compatible fixes

### When to Increment

| Change Type | Version Bump |
|-------------|--------------|
| Breaking schema change | MAJOR |
| New resource/data source | MINOR |
| New attribute (optional) | MINOR |
| Bug fix | PATCH |
| Documentation update | PATCH |
| Dependency update (non-breaking) | PATCH |

## Notes

- The script auto-detects the default branch (main or master)
- Each step requires confirmation unless `-y` flag is used
- The script will exit immediately if any command fails (set -e)
- Tags pushed to GitHub trigger the automated release workflow
- Both Terraform and OpenTofu registries are updated automatically


