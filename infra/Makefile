STACK_NAME ?= pics-backup
TEMPLATE_FILE = s3-backup-bucket.yml
BUCKET_NAME ?=
ENABLE_VERSIONING ?= false
GLACIER_TRANSITION_DAYS ?= 90

.PHONY: deploy
deploy:
	@echo "Deploying CloudFormation stack: $(STACK_NAME)"
	@aws cloudformation deploy \
		--stack-name $(STACK_NAME) \
		--template-file $(TEMPLATE_FILE) \
		--parameter-overrides \
			BucketName=$(BUCKET_NAME) \
			EnableVersioning=$(ENABLE_VERSIONING) \
			TransitionToGlacierDays=$(GLACIER_TRANSITION_DAYS) \
		--no-fail-on-empty-changeset

.PHONY: empty-bucket
empty-bucket:
	@echo "Emptying S3 bucket..."
	@BUCKET_NAME=$$(aws cloudformation describe-stacks \
		--stack-name $(STACK_NAME) \
		--query 'Stacks[0].Outputs[?OutputKey==`BucketName`].OutputValue' \
		--output text 2>/dev/null); \
	if [ -n "$$BUCKET_NAME" ]; then \
		echo "Bucket name: $$BUCKET_NAME"; \
		echo "Deleting all object versions and delete markers..."; \
		while true; do \
			VERSIONS=$$(aws s3api list-object-versions \
				--bucket "$$BUCKET_NAME" \
				--max-items 1000 \
				--query '{Objects: Versions[].{Key:Key,VersionId:VersionId}, DeleteMarkers: DeleteMarkers[].{Key:Key,VersionId:VersionId}}' \
				--output json 2>/dev/null || echo '{}'); \
			OBJECTS=$$(echo "$$VERSIONS" | python3 -c "import sys, json; data = json.load(sys.stdin); objs = (data.get('Objects') or []) + (data.get('DeleteMarkers') or []); print(json.dumps(objs))" 2>/dev/null || echo '[]'); \
			OBJECTS=$${OBJECTS:-[]}; \
			if [ "$$OBJECTS" = "[]" ] || [ -z "$$OBJECTS" ]; then \
				echo "No more objects to delete"; \
				break; \
			fi; \
			echo "Deleting batch of $$(echo "$$OBJECTS" | python3 -c "import sys, json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "unknown") objects..."; \
			echo "$$OBJECTS" | python3 -c "import sys, json; objs = json.load(sys.stdin); print(json.dumps({'Objects': objs, 'Quiet': True}))" > /tmp/delete-objects.json 2>/dev/null; \
			aws s3api delete-objects \
				--bucket "$$BUCKET_NAME" \
				--delete file:///tmp/delete-objects.json \
				--output json > /dev/null 2>&1 || true; \
			rm -f /tmp/delete-objects.json; \
		done; \
		echo "Bucket emptied successfully"; \
	else \
		echo "Stack not found or bucket name not available"; \
	fi

.PHONY: delete
delete: empty-bucket
	@echo "Deleting CloudFormation stack: $(STACK_NAME)"
	@aws cloudformation delete-stack --stack-name $(STACK_NAME)
	@echo "Waiting for stack deletion..."
	@aws cloudformation wait stack-delete-complete --stack-name $(STACK_NAME)
	@echo "Stack deleted successfully"

.PHONY: status
status:
	@aws cloudformation describe-stacks \
		--stack-name $(STACK_NAME) \
		--query 'Stacks[0].StackStatus' \
		--output text

.PHONY: outputs
outputs:
	@aws cloudformation describe-stacks \
		--stack-name $(STACK_NAME) \
		--query 'Stacks[0].Outputs' \
		--output table

.PHONY: bucket-name
bucket-name:
	@aws cloudformation describe-stacks \
		--stack-name $(STACK_NAME) \
		--query 'Stacks[0].Outputs[?OutputKey==`BucketName`].OutputValue' \
		--output text

.PHONY: validate
validate:
	@echo "Validating CloudFormation template..."
	@aws cloudformation validate-template --template-body file://$(TEMPLATE_FILE)
	@echo "Template is valid"

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  deploy              - Deploy or update the CloudFormation stack"
	@echo "  empty-bucket        - Empty all objects from the S3 bucket"
	@echo "  delete              - Empty bucket and delete the CloudFormation stack"
	@echo "  status              - Show stack status"
	@echo "  outputs             - Show stack outputs"
	@echo "  bucket-name         - Show bucket name only"
	@echo "  validate            - Validate the CloudFormation template"
	@echo ""
	@echo "Variables:"
	@echo "  STACK_NAME          - CloudFormation stack name (default: parse-pics-backup)"
	@echo "  BUCKET_NAME         - S3 bucket name (default: auto-generated)"
	@echo "  ENABLE_VERSIONING   - Enable versioning (default: false)"
	@echo "  GLACIER_TRANSITION_DAYS - Days before Glacier transition (default: 90)"
	@echo ""
	@echo "Examples:"
	@echo "  make deploy"
	@echo "  make deploy BUCKET_NAME=my-backup-bucket"
	@echo "  make deploy ENABLE_VERSIONING=true"
	@echo "  make bucket-name"
	@echo "  make delete"
