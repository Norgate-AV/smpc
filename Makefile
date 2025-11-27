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
