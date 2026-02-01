#!/bin/bash
# Test Release Script
#
# Creates a test release to verify the GitHub Actions release pipeline.
# Use cleanup-test-release.sh to remove the test release afterwards.
#
# Usage:
#   ./scripts/test-release.sh [TAG]
#
# Arguments:
#   TAG  Optional tag name (default: v0.0.1-test)
#
# Prerequisites:
#   - Git repository with push access
#   - GitHub CLI (gh) installed and authenticated
#
# What this does:
#   1. Creates an annotated tag locally
#   2. Pushes the tag to origin
#   3. Triggers the release workflow
#
# After running:
#   1. Monitor GitHub Actions - all 4 jobs should complete
#   2. Verify release page has:
#      - 6 CLI archives (3 OS × 2 arch)
#      - 3 UI archives (Linux, Windows, macOS)
#      - checksums.txt
#      - Updated installation instructions
#   3. Download and test binaries on respective platforms
#   4. Run ./scripts/cleanup-test-release.sh to clean up

set -e

TAG="${1:-v0.0.1-test}"

echo "Creating test release: $TAG"
echo ""

# Check if tag already exists locally
if git tag -l "$TAG" | grep -q "$TAG"; then
    echo "Error: Tag $TAG already exists locally."
    echo "Run ./scripts/cleanup-test-release.sh first or choose a different tag."
    exit 1
fi

# Check if tag already exists on remote
if git ls-remote --tags origin | grep -q "refs/tags/$TAG"; then
    echo "Error: Tag $TAG already exists on remote."
    echo "Run ./scripts/cleanup-test-release.sh first or choose a different tag."
    exit 1
fi

# Create and push tag
echo "Creating annotated tag..."
git tag -a "$TAG" -m "Test release"

echo "Pushing tag to origin..."
git push origin "$TAG"

echo ""
echo "Test release triggered successfully!"
echo ""
echo "Next steps:"
echo "  1. Monitor the workflow: gh run watch"
echo "  2. Or view in browser: gh run list --workflow=release.yml"
echo "  3. When complete, check release: gh release view $TAG"
echo "  4. Clean up with: ./scripts/cleanup-test-release.sh $TAG"
