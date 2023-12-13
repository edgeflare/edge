.PHONY: all build-ui build build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 build-windows-arm64 docker clean help

# Defaults
BINARY_NAME := edge
IMAGE_NAME := edgeflare/edge

# Default target executed when no arguments are given to make
all: build-ui build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 build-windows-arm64

# Build UI
build-ui:
	@echo "Building UI..."
	@cd ui && npm ci && npm run build && cd ..

# Function to determine binary suffix
define binary_suffix
$(if $(findstring linux-amd64,$(1)-$(2)),,-$(1)-$(2)$(if $(findstring windows,$(1)),.exe,))
endef

# Build the application for the specified OS and architecture
build:
	@echo "Building application for $(GOOS)/$(GOARCH)..."
	@mkdir -p ./bin
	$(eval BINARY_OS_ARCH_SUFFIX := $(call binary_suffix,$(GOOS),$(GOARCH)))
	@GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags='-w -s -extldflags "-static"' -a -o ./bin/$(BINARY_NAME)$(BINARY_OS_ARCH_SUFFIX)

# Build for Linux AMD64
build-linux-amd64: build-ui
	@$(MAKE) build GOOS=linux GOARCH=amd64

# Build for Linux ARM64
build-linux-arm64: build-ui
	@$(MAKE) build GOOS=linux GOARCH=arm64

# Build for Darwin AMD64
build-darwin-amd64: build-ui
	@$(MAKE) build GOOS=darwin GOARCH=amd64

# Build for Darwin ARM64
build-darwin-arm64: build-ui
	@$(MAKE) build GOOS=darwin GOARCH=arm64

# Build for Windows AMD64
build-windows-amd64: build-ui
	@$(MAKE) build GOOS=windows GOARCH=amd64

# Build for Windows ARM64
build-windows-arm64: build-ui
	@$(MAKE) build GOOS=windows GOARCH=arm64

# Build Docker image
docker: build-ui
	@echo "Building Docker image..."
	@docker build -t $(IMAGE_NAME) .

# Clean up
clean:
	@echo "Cleaning up..."
	@rm -rf ./bin
	@rm -rf ./ui/dist

# Help
help:
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all                 Build for all specified platforms"
	@echo "  build-ui            Build the UI"
	@echo "  build-linux-amd64   Build for Linux AMD64"
	@echo "  build-darwin-arm64  Build for Darwin ARM64"
	@echo "  build-windows-amd64 Build for Windows AMD64"
	@echo "  docker              Build Docker image"
	@echo "  clean               Clean up the build"
	@echo "  help                Show this help message"
