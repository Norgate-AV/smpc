APP_NAME := smpc
TARGET=$(APP_NAME).exe

SRC_DIR := .
BUILD_DIR := bin
DIST_DIR := dist
COVERAGE_DIR := .coverage
GOBIN := $(shell go env GOBIN)
INSTALL_DIR := $(if $(GOBIN),$(GOBIN),$(shell go env GOPATH)/bin)

GO_MODULE := github.com/Norgate-AV/$(APP_NAME)

# Version information (from git tags and commit)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME := $(shell git log -1 --format=%cI 2>/dev/null || echo unknown)

CGO_ENABLED := 0
BUILD_TAGS := netgo osusergo
LDFLAGS_BASE := -s -w -buildid= -X $(GO_MODULE)/internal/version.version=$(VERSION) \
								-X $(GO_MODULE)/internal/version.commit=$(COMMIT) \
								-X $(GO_MODULE)/internal/version.date=$(BUILD_TIME)
LDFLAGS := -ldflags "$(LDFLAGS_BASE) -extldflags '-static'"

.PHONY: build
build: clean
	CGO_ENABLED=$(CGO_ENABLED) go build \
	$(LDFLAGS) \
	-tags "$(BUILD_TAGS)" \
	-trimpath \
	-o $(BUILD_DIR)/$(TARGET) \
	$(SRC_DIR)

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR) $(DIST_DIR) $(COVERAGE_DIR)

.PHONY: install
install: build
	cp $(BUILD_DIR)/$(TARGET) $(INSTALL_DIR)/$(TARGET)

.PHONY: test
test:
	go test ./... -v

.PHONY: test-coverage
test-coverage:
	@mkdir -p $(COVERAGE_DIR)
	go test ./... -coverprofile=$(COVERAGE_DIR)/coverage.out
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html

.PHONY: test-integration
test-integration:
	go test ./... -tags=integration -v

.PHONY: fmt
fmt:
	go tool goimports -w -local github.com/Norgate-AV/smpc ./cmd ./internal ./test

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint: fmt vet
	go tool golangci-lint run

.PHONY: vuln
vuln:
	go tool govulncheck ./...







