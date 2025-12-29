BINARY_NAME_CLI=pics
BINARY_NAME_UI=pics-ui
GORELEASER := go run github.com/goreleaser/goreleaser/v2@latest

.PHONY: build
build: build-cli

.PHONY: build-cli
build-cli:
	go build -o $(BINARY_NAME_CLI) ./apps/cli

.PHONY: build-ui
build-ui:
	cd apps/ui && wails build

.PHONY: build-all
build-all: build-cli build-ui

.PHONY: dev-ui
dev-ui:
	cd apps/ui && wails dev

.PHONY: run
# Example: make run ARGS="parse /source /target"
run:
	go run ./apps/cli $(ARGS)

.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	rm -f $(BINARY_NAME_CLI)
	rm -rf apps/ui/build
	rm -rf dist/

.PHONY: release-snapshot
release-snapshot:
	$(GORELEASER) release --snapshot --clean

.PHONY: release-test
release-test:
	$(GORELEASER) check

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
