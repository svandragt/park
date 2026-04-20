.PHONY: build test vet install clean

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

install:
	go install .

clean:
	go clean ./...
