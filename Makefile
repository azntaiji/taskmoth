GOLANGCI_LINT_VERSION := v2.15.0
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: build test vet lint fmt run check clean

build:
	go build -o bin/taskmoth ./cmd/taskmoth

test:
	go test ./...

vet:
	go vet ./...

lint:
	$(GOLANGCI_LINT) run

fmt:
	$(GOLANGCI_LINT) fmt

run:
	go run ./cmd/taskmoth $(ARGS)

# Everything a phase must pass before it is called done.
check: build vet test lint

clean:
	rm -rf bin
