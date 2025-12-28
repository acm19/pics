# Release Process

This document describes how to create a new release of `pics`.

## Prerequisites

- [GoReleaser](https://goreleaser.com/install/) installed for local testing.
- Push access to the repository.
- Ability to create tags.

## Version Numbering

This project follows [Semantic Versioning](https://semver.org/):
- `v1.0.0` - Major release (breaking changes).
- `v1.1.0` - Minor release (new features, backwards compatible).
- `v1.0.1` - Patch release (bug fixes).

## Testing a Release Locally

Before creating a release, test the build process locally:

```bash
# Validate GoReleaser configuration
make release-test

# Build snapshot (doesn't publish)
make release-snapshot

# Check the built binaries
ls -lh dist/

# Test a binary
./dist/pics_linux_amd64_v1/pics --version
```

## Creating a Release

1. **Ensure all changes are committed and pushed to `main`.**
   ```bash
   git status
   git push origin main
   ```

2. **Create and push a tag.**
   ```bash
   # Create an annotated tag
   git tag -a v1.0.0 -m "Release v1.0.0"

   # Push the tag
   git push origin v1.0.0
   ```

3. **GitHub Actions will automatically:**
   - Run all tests.
   - Build binaries for:
     - Linux (amd64, arm64).
     - macOS (amd64, arm64).
     - Windows (amd64, arm64).
   - Generate checksums.
   - Create changelog from git commits.
   - Create a GitHub release with all assets.

4. **Monitor the release.**
   - Go to: https://github.com/acm19/pics/actions
   - Check the "Release" workflow status.
   - Once complete, verify: https://github.com/acm19/pics/releases

## What Gets Released

Each release includes:
- **Binaries**: Cross-platform executables (6 variants).
- **Archives**: `.tar.gz` (Linux/macOS) and `.zip` (Windows).
- **Checksums**: `checksums.txt` for verifying downloads.
- **Changelog**: Auto-generated from commit messages.
- **README**: Installation instructions.

## Release Checklist

- [ ] All tests pass locally (`make test`).
- [ ] Version follows semantic versioning.
- [ ] Changelog-worthy commits have clear messages.
- [ ] No uncommitted changes.
- [ ] Tag pushed to remote.
- [ ] GitHub Actions workflow completes successfully.
- [ ] Release assets uploaded correctly.
- [ ] Release notes are accurate.

## Troubleshooting

**Release failed?**
- Check GitHub Actions logs for errors.
- Delete the release and tag (see "Deleting a Release" below).
- Fix the issue and try again.

**Need to update a release?**
- Delete the GitHub release (keeps the tag).
- Re-run the workflow or push the tag again.
- GoReleaser will recreate the release.

## Deleting a Release

**Important**: Deleting a git tag does NOT automatically delete the GitHub release. You must delete them separately.

### Complete Removal (Release + Tag)

1. **Delete the GitHub release first.**
   ```bash
   # Using GitHub CLI
   gh release delete v1.0.0 --yes

   # Or via web UI:
   # Go to https://github.com/acm19/pics/releases
   # Click the release → Delete
   ```

2. **Then delete the tag.**
   ```bash
   # Delete local tag
   git tag -d v1.0.0

   # Delete remote tag
   git push origin :refs/tags/v1.0.0
   ```

### Just Update Release Assets (Keep Tag)

If you only want to update the release without changing the tag:

1. Delete the GitHub release (via `gh release delete v1.0.0`).
2. Keep the tag.
3. Re-trigger the workflow or manually create a new release.

## Rolling Back

If you need to roll back a release:
1. Follow "Deleting a Release" steps above.
2. Users should download the previous version.
3. Consider creating a new patch release with the fix instead.
