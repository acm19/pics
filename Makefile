BINARY_NAME_CLI=pics
BINARY_NAME_UI=pics-ui

# Tool versions
GORELEASER_VERSION=v2.4.8
WAILS_VERSION=v2.11.0
EXIFTOOL_VERSION=13.45
JPEGOPTIM_VERSION=1.5.6

# Tool commands
GORELEASER := go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)
WAILS := go run github.com/wailsapp/wails/v2/cmd/wails@$(WAILS_VERSION)

.DEFAULT_GOAL := help

.PHONY: help
.SILENT: help
help:
	echo "Available targets:"
	echo ""
	echo "Build:"
	echo "  make build            - Build CLI application (alias for build-cli)"
	echo "  make build-cli        - Build CLI application"
	echo "  make build-ui         - Build UI application (downloads binaries if needed)"
	echo "  make build-all        - Build both CLI and UI applications"
	echo "  make local-install    - Build and install CLI to ~/bin"
	echo ""
	echo "Development:"
	echo "  make dev-ui           - Run UI in development mode"
	echo "  make run ARGS=\"...\"   - Run CLI with arguments (e.g., make run ARGS=\"parse /src /dst\")"
	echo "  make test             - Run all tests"
	echo "  make tidy             - Tidy all go modules"
	echo "  make clean            - Clean all build artifacts and temporary files"
	echo ""
	echo "Release:"
	echo "  make release-snapshot - Create release snapshot"
	echo "  make release-test     - Validate release configuration"
	echo ""
	echo "Binary downloads:"
	echo "  make download-binaries - Download all platform binaries"
	echo ""
	echo "Infrastructure (AWS):"
	echo "  make infra-deploy     - Deploy infrastructure"
	echo "  make infra-status     - Show infrastructure status"
	echo "  make infra-outputs    - Show infrastructure outputs"
	echo "  make infra-bucket-name - Show S3 bucket name"
	echo "  make infra-validate   - Validate infrastructure configuration"
	echo "  make infra-empty-bucket - Empty S3 bucket"
	echo "  make infra-delete     - Delete infrastructure"
	echo "  make infra-help       - Show infrastructure help"

.PHONY: build
build: build-cli

.PHONY: clean
.SILENT: clean
clean:
	echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME_CLI)
	rm -f $(BINARY_NAME_UI)
	rm -rf apps/ui/build/bin
	rm -rf dist/
	echo "Cleaning downloaded binaries..."
	rm -rf apps/ui/build/resources
	echo "Cleaning runtime extracted binaries..."
	rm -rf /tmp/pics-ui-tools
	echo "Cleaning temporary directories..."
	rm -rf /tmp/pics-*
	echo "✓ Clean complete"

.PHONY: build-cli
.SILENT: build-cli
build-cli:
	echo "Building CLI..."
	cd apps/cli && go build -ldflags="-s -w" -o ../../$(BINARY_NAME_CLI)
	echo "✓ Built $(BINARY_NAME_CLI)"

.PHONY: local-install
.SILENT: local-install
local-install: build-cli
	echo "Installing $(BINARY_NAME_CLI) to ~/bin..."
	mkdir -p ~/bin
	cp $(BINARY_NAME_CLI) ~/bin/$(BINARY_NAME_CLI)
	echo "✓ Installed $(BINARY_NAME_CLI) to ~/bin/$(BINARY_NAME_CLI)"

.PHONY: build-ui
.SILENT: build-ui
build-ui: \
	apps/ui/build/resources/windows/exiftool.exe \
	apps/ui/build/resources/windows/jpegoptim.exe \
	apps/ui/build/resources/darwin/exiftool \
	apps/ui/build/resources/darwin/jpegoptim \
	apps/ui/build/resources/linux/exiftool \
	apps/ui/build/resources/linux/jpegoptim
	echo "Building UI..."
	cd apps/ui && $(WAILS) build -tags webkit2_41
	echo "✓ Built $(BINARY_NAME_UI)"

.PHONY: build-all
.SILENT: build-all
build-all: build-cli build-ui
	echo "✓ Built all applications"

.PHONY: dev-ui
.SILENT: dev-ui
dev-ui:
	echo "Starting UI in development mode..."
	cd apps/ui && $(WAILS) dev -tags webkit2_41

.PHONY: run
.SILENT: run
run:
	cd apps/cli && go run . $(ARGS)

.PHONY: test
.SILENT: test
test:
	echo "Running tests..."
	go test -v ./...
	cd apps/cli && go test -v ./...
	cd apps/ui && go test -v ./...
	echo "✓ All tests passed"

.PHONY: tidy
.SILENT: tidy
tidy:
	echo "Tidying modules..."
	go mod tidy
	cd apps/cli && go mod tidy
	cd apps/ui && go mod tidy
	echo "✓ Modules tidied"

# Windows binaries
apps/ui/build/resources/windows/exiftool.exe:
	@echo "Downloading exiftool for Windows..."
	@mkdir -p apps/ui/build/resources/windows
	@TMPDIR=$$(mktemp -d /tmp/pics-exiftool-windows.XXXXXX) && \
		cd $$TMPDIR && \
		curl -L -o exiftool.zip "https://sourceforge.net/projects/exiftool/files/exiftool-$(EXIFTOOL_VERSION)_64.zip/download" && \
		unzip -q exiftool.zip && \
		chmod -R u+w . && \
		cp "exiftool-$(EXIFTOOL_VERSION)_64/exiftool(-k).exe" $(CURDIR)/apps/ui/build/resources/windows/exiftool.exe && \
		cd /tmp && rm -rf $$TMPDIR
	@echo "✓ exiftool.exe downloaded"

apps/ui/build/resources/windows/jpegoptim.exe:
	@echo "Downloading jpegoptim for Windows..."
	@mkdir -p apps/ui/build/resources/windows
	@TMPDIR=$$(mktemp -d /tmp/pics-jpegoptim-windows.XXXXXX) && \
		cd $$TMPDIR && \
		curl -L -o jpegoptim.zip "https://github.com/XhmikosR/jpegoptim-windows/releases/download/$(JPEGOPTIM_VERSION)-rel1/jpegoptim-$(JPEGOPTIM_VERSION)-rel1-win64-msvc-2022-mozjpeg331-static-ltcg.zip" && \
		unzip -q jpegoptim.zip && \
		chmod -R u+w . && \
		cp jpegoptim-*/jpegoptim.exe $(CURDIR)/apps/ui/build/resources/windows/jpegoptim.exe && \
		cd /tmp && rm -rf $$TMPDIR
	@echo "✓ jpegoptim.exe downloaded"

# macOS binaries
apps/ui/build/resources/darwin/exiftool:
	@echo "Downloading exiftool for macOS..."
	@mkdir -p apps/ui/build/resources/darwin
	@TMPDIR=$$(mktemp -d /tmp/pics-exiftool-darwin.XXXXXX) && \
		cd $$TMPDIR && \
		curl -L -o exiftool.tar.gz "https://exiftool.org/Image-ExifTool-$(EXIFTOOL_VERSION).tar.gz" && \
		tar -xzf exiftool.tar.gz && \
		chmod -R u+w . && \
		cp -r Image-ExifTool-$(EXIFTOOL_VERSION)/lib $(CURDIR)/apps/ui/build/resources/darwin/ && \
		cp Image-ExifTool-$(EXIFTOOL_VERSION)/exiftool $(CURDIR)/apps/ui/build/resources/darwin/exiftool && \
		chmod +x $(CURDIR)/apps/ui/build/resources/darwin/exiftool && \
		cd /tmp && rm -rf $$TMPDIR
	@echo "✓ exiftool downloaded"

apps/ui/build/resources/darwin/jpegoptim:
	@echo "Downloading jpegoptim for macOS..."
	@mkdir -p apps/ui/build/resources/darwin
	@TMPDIR=$$(mktemp -d /tmp/pics-jpegoptim-darwin.XXXXXX) && \
		cd $$TMPDIR && \
		curl -L -o jpegoptim.zip "https://github.com/tjko/jpegoptim/releases/download/v$(JPEGOPTIM_VERSION)/jpegoptim-$(JPEGOPTIM_VERSION)-x64-osx.zip" && \
		unzip -q jpegoptim.zip && \
		chmod -R u+w . && \
		cp jpegoptim $(CURDIR)/apps/ui/build/resources/darwin/jpegoptim && \
		chmod +x $(CURDIR)/apps/ui/build/resources/darwin/jpegoptim && \
		cd /tmp && rm -rf $$TMPDIR
	@echo "✓ jpegoptim downloaded"

# Linux binaries
apps/ui/build/resources/linux/exiftool:
	@echo "Downloading exiftool for Linux..."
	@mkdir -p apps/ui/build/resources/linux
	@TMPDIR=$$(mktemp -d /tmp/pics-exiftool-linux.XXXXXX) && \
		cd $$TMPDIR && \
		curl -L -o exiftool.tar.gz "https://exiftool.org/Image-ExifTool-$(EXIFTOOL_VERSION).tar.gz" && \
		tar -xzf exiftool.tar.gz && \
		chmod -R u+w . && \
		cp -r Image-ExifTool-$(EXIFTOOL_VERSION)/lib $(CURDIR)/apps/ui/build/resources/linux/ && \
		cp Image-ExifTool-$(EXIFTOOL_VERSION)/exiftool $(CURDIR)/apps/ui/build/resources/linux/exiftool && \
		chmod +x $(CURDIR)/apps/ui/build/resources/linux/exiftool && \
		cd /tmp && rm -rf $$TMPDIR
	@echo "✓ exiftool downloaded"

apps/ui/build/resources/linux/jpegoptim:
	@echo "Downloading jpegoptim for Linux..."
	@mkdir -p apps/ui/build/resources/linux
	@TMPDIR=$$(mktemp -d /tmp/pics-jpegoptim-linux.XXXXXX) && \
		cd $$TMPDIR && \
		curl -L -o jpegoptim.zip "https://github.com/tjko/jpegoptim/releases/download/v$(JPEGOPTIM_VERSION)/jpegoptim-$(JPEGOPTIM_VERSION)-x64-linux.zip" && \
		unzip -q jpegoptim.zip && \
		chmod -R u+w . && \
		cp jpegoptim $(CURDIR)/apps/ui/build/resources/linux/jpegoptim && \
		chmod +x $(CURDIR)/apps/ui/build/resources/linux/jpegoptim && \
		cd /tmp && rm -rf $$TMPDIR
	@echo "✓ jpegoptim downloaded"

# Convenience target to download all binaries
.PHONY: download-binaries
download-binaries: \
	apps/ui/build/resources/windows/exiftool.exe \
	apps/ui/build/resources/windows/jpegoptim.exe \
	apps/ui/build/resources/darwin/exiftool \
	apps/ui/build/resources/darwin/jpegoptim \
	apps/ui/build/resources/linux/exiftool \
	apps/ui/build/resources/linux/jpegoptim
	@echo ""
	@echo "✓ All binaries ready!"

.PHONY: release-snapshot
.SILENT: release-snapshot
release-snapshot:
	echo "Creating release snapshot..."
	$(GORELEASER) release --snapshot --clean
	echo "✓ Release snapshot created"

.PHONY: release-test
.SILENT: release-test
release-test:
	echo "Validating release configuration..."
	$(GORELEASER) check
	echo "✓ Release configuration valid"

# Infrastructure targets
.PHONY: infra-deploy
infra-deploy:
	$(MAKE) -C infra deploy

.PHONY: infra-empty-bucket
infra-empty-bucket:
	$(MAKE) -C infra empty-bucket

.PHONY: infra-delete
infra-delete:
	$(MAKE) -C infra delete

.PHONY: infra-status
infra-status:
	$(MAKE) -C infra status

.PHONY: infra-outputs
infra-outputs:
	$(MAKE) -C infra outputs

.PHONY: infra-bucket-name
infra-bucket-name:
	$(MAKE) -C infra bucket-name

.PHONY: infra-validate
infra-validate:
	$(MAKE) -C infra validate

.PHONY: infra-help
infra-help:
	$(MAKE) -C infra help
