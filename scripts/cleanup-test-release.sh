#!/bin/bash
# Cleanup Test Release Script
#
# Removes a test release created by test-release.sh.
# All commands are allowed to fail (for idempotency).
#
# Usage:
#   ./scripts/cleanup-test-release.sh [TAG]
#
# Arguments:
#   TAG  Optional tag name (default: v0.0.1-test)
#
# Prerequisites:
#   - GitHub CLI (gh) installed and authenticated
#
# What this does:
#   1. Deletes the GitHub release (if exists)
#   2. Deletes the remote tag (if exists)
#   3. Deletes the local tag (if exists)

set -u

TAG="${1:-v0.0.1-test}"

echo "Cleaning up test release: $TAG"
echo ""

# Delete GitHub release (allow to fail if it doesn't exist)
echo "Deleting GitHub release..."
gh release delete "$TAG" -y 2>/dev/null || echo "  Release not found or already deleted"

# Delete remote tag (allow to fail)
echo "Deleting remote tag..."
git push origin :"$TAG" 2>/dev/null || echo "  Remote tag not found or already deleted"

# Delete local tag (allow to fail)
echo "Deleting local tag..."
git tag -d "$TAG" 2>/dev/null || echo "  Local tag not found or already deleted"

echo ""
echo "Cleanup complete"
