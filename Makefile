.PHONY: build test clean

BINARY_NAME=parse-pics

build:
	go build -o $(BINARY_NAME) .

test:
	go test -v ./...

clean:
	rm -f $(BINARY_NAME)
