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

### Run the compiled binary

```bash
./parse-pics SOURCE_DIR TARGET_DIR
```

### Run without compiling

```bash
make run ARGS="/path/to/source /path/to/target"
```

### Arguments

- `SOURCE_DIR` - Directory containing subdirectories with media files.
- `TARGET_DIR` - Directory where organised files will be placed.

### Environment Variables

- `RATE` - JPEG compression quality (0-100, default: 50).
- `DEBUG` - Enable debug logging (set to any non-empty value).

### Examples

```bash
# Basic usage with default settings (quality 50)
./parse-pics /path/to/source /path/to/target

# Custom compression quality
RATE=75 ./parse-pics /path/to/source /path/to/target

# Enable debug logging
DEBUG=1 ./parse-pics /path/to/source /path/to/target

# Combine options
DEBUG=1 RATE=80 ./parse-pics /path/to/source /path/to/target
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
DEBUG=1 ./parse-pics /source /target
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
