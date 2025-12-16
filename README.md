# Parse Pics

A Go application for organising and compressing photos and videos. Replicates the functionality of the original `parse-pics.sh` bash script with improved error handling and structured logging.

## Features

- Copies media files (JPG, MOV) from source subdirectories.
- Optional JPEG compression with configurable quality.
- Organises files into date-based directories (YYYY MM Month DD).
- Moves videos to separate subdirectories.
- Renames JPEGs sequentially.
- Preserves file modification times.
- Structured logging with debug mode.

## Requirements

- Go 1.21 or later.
- `jpegoptim` - for JPEG compression with EXIF preservation.
- AWS credentials configured (for S3 backup feature) - via environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`) or `~/.aws/credentials` file.

### Installing jpegoptim

**Ubuntu/Debian:**
```bash
sudo apt install jpegoptim
```

**macOS:**
```bash
brew install jpegoptim
```

**Fedora/RHEL:**
```bash
sudo dnf install jpegoptim
```

**Arch Linux:**
```bash
sudo pacman -S jpegoptim
```

## Building

```bash
make build
```

This creates a `parse-pics` binary in the current directory.

## Usage

### Parse and organise media files

```bash
# Using compiled binary
./parse-pics parse SOURCE_DIR TARGET_DIR

# Using make
make run ARGS="parse /path/to/source /path/to/target"
```

**Arguments:**
- `SOURCE_DIR` - Directory containing subdirectories with media files.
- `TARGET_DIR` - Directory where organised files will be placed.

### Rename a date-based directory

```bash
# Using compiled binary
./parse-pics rename DIRECTORY NAME

# Using make
make run ARGS="rename '/path/to/2025 12 December 15' Vacation"
```

**Arguments:**
- `DIRECTORY` - Path to the date-based directory (format: YYYY MM Month DD [current-name]).
- `NAME` - New name to append or replace after the date.

**Examples:**
```bash
# Add name to unnamed directory
./parse-pics rename "/pics/2025 12 December 15" "Vacation"
# Result: /pics/2025 12 December 15 Vacation/
#         Images: 2025_12_December_15_Vacation_00001.jpg

# Replace existing name
./parse-pics rename "/pics/2025 12 December 15 OldName" "NewName"
# Result: /pics/2025 12 December 15 NewName/
#         Images: 2025_12_December_15_NewName_00001.jpg
```

### Backup directories to S3

```bash
# Using compiled binary
./parse-pics backup SOURCE_DIR BUCKET

# Using make
make run ARGS="backup /path/to/organised/pics my-backup-bucket"
```

**Arguments:**
- `SOURCE_DIR` - Directory containing date-based subdirectories to backup.
- `BUCKET` - S3 bucket name where archives will be uploaded.

**How it works:**
- Creates tar.gz archives of each subdirectory in a temporary location (`/tmp/<random>_pic`).
- Counts images and videos in each directory and includes counts in the S3 object key.
- Checks if objects already exist in S3 using MD5 hash comparison.
- Skips upload if identical archive already exists.
- Fails with error if object exists but hash differs (manual intervention required).
- Uploads new archives to S3 with format: `directory-name (X images, Y videos).tar.gz`.
- Processes directories in parallel (configurable, default 5).
- Automatically cleans up temporary files after each upload.

**Examples:**
```bash
# Basic backup with default concurrency (5)
./parse-pics backup /pics/organised my-backup-bucket

# Custom concurrency level
MAX_CONCURRENT=3 ./parse-pics backup /pics/organised my-backup-bucket

# With debug logging
DEBUG=1 ./parse-pics backup /pics/organised my-backup-bucket
```

**S3 object naming:**
Archives are named with image and video counts:
- `2025 12 December 15 Vacation (42 images, 3 videos).tar.gz`
- `2025 11 November 20 (15 images, 0 videos).tar.gz`

### Environment Variables

- `RATE` - JPEG compression quality (0-100, default: 50).
- `DEBUG` - Enable debug logging (set to any non-empty value).
- `MAX_CONCURRENT` - Maximum concurrent backup operations for S3 backup (default: 5).

### Examples

```bash
# Basic usage with default settings (quality 50)
./parse-pics parse /path/to/source /path/to/target

# Custom compression quality
RATE=75 ./parse-pics parse /path/to/source /path/to/target

# Enable debug logging
DEBUG=1 ./parse-pics parse /path/to/source /path/to/target

# Combine options
DEBUG=1 RATE=80 ./parse-pics parse /path/to/source /path/to/target
```

## Logging

The application uses structured logging with two levels:

### Info Level (Default)

Shows major operations:
- Creating temporary directory.
- Copying media files.
- Compressing JPEGs.
- Organising by date.
- Final organisation.
- Summary statistics.

```
time=2025-12-15T10:30:00.000Z level=INFO msg="Starting media parsing" source=/source target=/target
time=2025-12-15T10:30:01.000Z level=INFO msg="Copying media files" source=/source target=/target/tmp_image
time=2025-12-15T10:30:05.000Z level=INFO msg="Compressing JPEGs" quality=50
time=2025-12-15T10:30:10.000Z level=INFO msg="Processing complete"
```

### Debug Level

Shows detailed operations including individual files:
- Processing each subdirectory.
- Copying individual files.
- Compressing individual files.

```bash
# Enable with DEBUG environment variable
DEBUG=1 ./parse-pics parse /source /target
```

```
time=2025-12-15T10:30:02.000Z level=DEBUG msg="Processing subdirectory" dir=vacation
time=2025-12-15T10:30:02.000Z level=DEBUG msg="Copying file" from=/source/vacation/IMG_001.JPG to=/target/tmp_image/vacation-IMG_001.JPG
time=2025-12-15T10:30:05.000Z level=DEBUG msg="Found JPEG files" count=42
time=2025-12-15T10:30:05.000Z level=DEBUG msg="Compressing file" path=/target/tmp_image/vacation-IMG_001.JPG
```

## How It Works

1. **Validation**: Checks that source and target directories exist.
2. **Copy**: Copies all JPG and MOV files from source subdirectories to a temporary directory, prefixing filenames with their subdirectory name.
3. **Compress** (optional): Re-encodes JPEG files at the specified quality level.
4. **Organise by Date**: Moves files into date-based directories based on file modification time.
5. **Final Organisation**:
   - Moves MOV files into `videos` subdirectories.
   - Renames JPG files sequentially (e.g., `2025_12_December_15_00001.jpg`).
6. **Cleanup**: Removes temporary directory.

## Configuration Options

Compression can be disabled in code by modifying `ParseOptions`:

```go
opts := DefaultParseOptions()
opts.CompressJPEGs = false  // Skip compression
```

By default, compression is enabled with quality 50.

## Development

### Running Tests

```bash
make test
```

### Clean Build Artifacts

```bash
make clean
```

## Original Script

This replaces the functionality of `parse-pics.sh` with the following improvements:

- No external dependencies (no `jpegoptim` required).
- Better error handling and reporting.
- Structured logging.
- Type safety.
- Easier to test and maintain.
