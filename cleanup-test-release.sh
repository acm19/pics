#!/bin/bash
# Cleanup script for test releases

TAG="v0.0.1-test"

echo "Cleaning up test release: $TAG"

# Delete GitHub release (allow to fail if it doesn't exist)
echo "Deleting GitHub release..."
gh release delete "$TAG" -y 2>/dev/null || echo "Release not found or already deleted"

# Delete remote tag (allow to fail)
echo "Deleting remote tag..."
git push origin :"$TAG" 2>/dev/null || echo "Remote tag not found or already deleted"

# Delete local tag (allow to fail)
echo "Deleting local tag..."
git tag -d "$TAG" 2>/dev/null || echo "Local tag not found or already deleted"

echo "Cleanup complete"
