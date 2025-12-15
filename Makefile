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
