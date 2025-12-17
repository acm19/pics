BINARY_NAME=parse-pics

.PHONY: build
build:
	go build -o $(BINARY_NAME) .

.PHONY: run
# Example: make run ARGS="/source /target"
run:
	go run . $(ARGS)

.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	rm -f $(BINARY_NAME)

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
