# Variables
BINARY_NAME=ggmp
VERSION=1.0.0
MAIN_PATH=./cmd/ggmp/main.go
SEED_PATH=./cmd/seed/main.go
LIST_PATH=./cmd/list/main.go

# Build directories
BUILD_DIR=./build
WINDOWS_DIR=$(BUILD_DIR)/windows
MACOS_DIR=$(BUILD_DIR)/macos
LINUX_DIR=$(BUILD_DIR)/linux

.PHONY: all build clean

# Default target
all: build

# Build releases for all platforms
build: windows macos linux

# Windows build
windows:
	@mkdir -p $(WINDOWS_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(WINDOWS_DIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64.exe $(MAIN_PATH)

# macOS builds (Apple Silicon and Intel)
macos:
	@mkdir -p $(MACOS_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(MACOS_DIR)/$(BINARY_NAME)-$(VERSION)-macos-arm64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 go build -o $(MACOS_DIR)/$(BINARY_NAME)-$(VERSION)-macos-amd64 $(MAIN_PATH)

# Linux build
linux:
	@mkdir -p $(LINUX_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(LINUX_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64 $(MAIN_PATH)

# Clean up build directory
clean:
	rm -rf $(BUILD_DIR)

# Run tests
test:
	go test ./...

# Run the application (uses local build)
run: 
	go run $(MAIN_PATH)

reset:
	rm -f ggmp.db

rerun: reset run

seed:
	go run ${SEED_PATH}

list:
	go run ${LIST_PATH}