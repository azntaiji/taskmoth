GOLANGCI_LINT_VERSION := v2.15.0
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GORELEASER_VERSION := v2.19.0
GORELEASER := go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)

.PHONY: build test vet lint fmt run gallery golden check release-check clean

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

# Widget gallery for comparing against Figma (tab / shift+tab between pages).
gallery:
	go run ./cmd/taskmoth --gallery $(ARGS)

# Rewrite golden test files after an intended visual change; review the diff.
golden:
	go test ./internal/ui/... -update

# Everything a phase must pass before it is called done.
check: build vet test lint

# Validate .goreleaser.yaml and build every release target into dist/
# without publishing (the release workflow publishes on vX.Y.Z tags).
release-check:
	$(GORELEASER) check
	$(GORELEASER) release --snapshot --clean

clean:
	rm -rf bin dist
