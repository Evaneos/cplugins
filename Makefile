VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X github.com/Evaneos/cplugins/cmd.version=$(VERSION)"

.PHONY: build test e2e-tests

build:
	go build $(LDFLAGS) -o cplugins .
	go install $(LDFLAGS) .

test:
	go test ./...

e2e-tests:
	go test ./e2e/... -v -timeout 300s
