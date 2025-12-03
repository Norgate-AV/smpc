TARGET=smpc.exe

ENTRY_POINT=main.go

BUILD_DIR=bin

.PHONY: build
build: clean
	go build -o $(BUILD_DIR)/$(TARGET) $(ENTRY_POINT)

.PHONY: clean
clean:
	@powershell -Command "if (Test-Path $(BUILD_DIR)) { Remove-Item -Recurse -Force $(BUILD_DIR) }"

.PHONY: install
install: build
	go install ./...

.PHONY: test
test:
	go test ./... -v

.PHONY: test-coverage
test-coverage:
	@powershell -Command "if (-not (Test-Path .coverage)) { New-Item -ItemType Directory -Path .coverage }"
	go test ./... -coverprofile=.coverage/coverage.out
	go tool cover -html=.coverage/coverage.out -o .coverage/coverage.html

.PHONY: test-short
test-short:
	go test ./... -short -v

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







