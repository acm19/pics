#!/bin/bash

# Script to automatically remove duplicate files based on original_name comparison
# Usage: ./remove_duplicate_files.sh [directory]

# Set the directory to scan (default to current directory)
TARGET_DIR="${1:-.}"

# Check if directory exists
if [ ! -d "$TARGET_DIR" ]; then
  echo "Error: Directory '$TARGET_DIR' does not exist"
  exit 1
fi

echo "Scanning for duplicate files in: $TARGET_DIR"
echo "=========================================="

# Associative array to store original_name -> filepath mappings
declare -A original_name_map

# Counters
total_files=0
duplicates_removed=0

# Find all files
while IFS= read -r -d '' file; do
  ((total_files++))
  
  # Get original filename from EXIF data (if available), it's inserted by my renaming script, so it should be there
  original_name=$(exiftool -OriginalFileName "$file" | awk '{print $5}' | awk -F'.' '{print $1}')

  # Check if original_name exists
  if [[ -n "${original_name_map[$original_name]}" ]]; then
    # Duplicate found - remove it
    echo "Duplicate: $file (same as ${original_name_map[$original_name]})"
    rm "$file"
    ((duplicates_removed++))
  else
    # Store this file as the original
    original_name_map[$original_name]="$file"
  fi
  
done < <(find "$TARGET_DIR" -maxdepth 1 -type f -print0 | sort -z)

# Print summary
echo ""
echo "=========================================="
echo "Summary:"
echo "  Total files scanned: $total_files"
echo "  Unique files: $((total_files - duplicates_removed))"
echo "  Duplicates removed: $duplicates_removed"
echo "=========================================="
