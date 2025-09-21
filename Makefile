# PPatcher Makefile
# Build PPatcher client executables for different platforms with custom configurations

# Default configuration file
CONFIG_FILE ?= config.json

# Default platforms to build for
PLATFORMS ?= windows/amd64,linux/amd64

# Build flags
CLEAN ?= false
DEBUG ?= false

# Colors for terminal output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m

.PHONY: help build build-windows build-linux build-all clean install-deps create-config check-deps test

# Default target
all: build

## Show help
help:
	@echo "PPatcher Build System"
	@echo ""
	@echo "Usage: make [target] [CONFIG_FILE=path] [PLATFORMS=list] [CLEAN=true/false] [DEBUG=true/false]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
	@echo ""
	@echo "Variables:"
	@echo "  CONFIG_FILE    Path to config.json file (default: config.json)"
	@echo "  PLATFORMS      Comma-separated list of platforms (default: windows/amd64,linux/amd64)"
	@echo "  CLEAN          Clean build before building (default: false)"
	@echo "  DEBUG          Build in debug mode (default: false)"
	@echo ""
	@echo "Examples:"
	@echo "  make build CONFIG_FILE=my-config.json"
	@echo "  make build PLATFORMS=windows/amd64,linux/amd64 CLEAN=true"
	@echo "  make build-windows DEBUG=true"
	@echo "  make create-config CONFIG_FILE=my-config.json"

## Install required dependencies
install-deps:
	@echo -e "$(BLUE)Installing dependencies...$(NC)"
	@if ! command -v wails >/dev/null 2>&1; then \
		echo "Installing Wails CLI..."; \
		go install github.com/wailsapp/wails/v2/cmd/wails@latest; \
	fi
	@if [ ! -d "frontend/node_modules" ]; then \
		echo "Installing frontend dependencies..."; \
		cd frontend && npm install; \
	fi
	@echo -e "$(GREEN)Dependencies installed successfully!$(NC)"

## Check if all dependencies are available
check-deps:
	@echo -e "$(BLUE)Checking dependencies...$(NC)"
	@if ! command -v wails >/dev/null 2>&1; then \
		echo -e "$(RED)❌ Wails CLI not found!$(NC)"; \
		echo "Install with: go install github.com/wailsapp/wails/v2/cmd/wails@latest"; \
		exit 1; \
	fi
	@if ! command -v go >/dev/null 2>&1; then \
		echo -e "$(RED)❌ Go not found!$(NC)"; \
		echo "Install from: https://golang.org/dl/"; \
		exit 1; \
	fi
	@if ! command -v npm >/dev/null 2>&1; then \
		echo -e "$(RED)❌ npm not found!$(NC)"; \
		echo "Install Node.js from: https://nodejs.org/"; \
		exit 1; \
	fi
	@echo -e "$(GREEN)✅ All dependencies are available$(NC)"

## Create a sample config file
create-config:
	@if [ -z "$(CONFIG_FILE)" ]; then \
		echo -e "$(RED)❌ CONFIG_FILE not specified$(NC)"; \
		echo "Usage: make create-config CONFIG_FILE=path/to/config.json"; \
		exit 1; \
	fi
	@mkdir -p $(dir $(CONFIG_FILE))
	@echo '{' > $(CONFIG_FILE)
	@echo '  "backend": "http://localhost:3000",' >> $(CONFIG_FILE)
	@echo '  "executable": "your-game-executable",' >> $(CONFIG_FILE)
	@echo '  "colorPalette": "green",' >> $(CONFIG_FILE)
	@echo '  "mode": "production",' >> $(CONFIG_FILE)
	@echo '  "outputName": "ppatcher",' >> $(CONFIG_FILE)
	@echo '  "version": "1.0.0",' >> $(CONFIG_FILE)
	@echo '  "description": "PPatcher Client"' >> $(CONFIG_FILE)
	@echo '}' >> $(CONFIG_FILE)
	@echo -e "$(GREEN)✅ Sample config created at: $(CONFIG_FILE)$(NC)"
	@echo -e "$(BLUE)Please edit the config file with your specific settings:$(NC)"
	@echo "  - backend: Your patch server URL"
	@echo "  - executable: Path to your game executable"
	@echo "  - colorPalette: UI color theme (green, blue, red, etc.)"
	@echo "  - mode: Build mode (production or dev)"
	@echo "  - outputName: Name of the output executable"

## Build for specified platforms
build: check-deps
	@if [ ! -f "$(CONFIG_FILE)" ]; then \
		echo -e "$(RED)❌ Config file not found: $(CONFIG_FILE)$(NC)"; \
		echo "Create one with: make create-config CONFIG_FILE=$(CONFIG_FILE)"; \
		exit 1; \
	fi
	@echo -e "$(BLUE)Building PPatcher client...$(NC)"
	@echo "Config file: $(CONFIG_FILE)"
	@echo "Platforms: $(PLATFORMS)"
	@./build-client.sh --config "$(CONFIG_FILE)" --platforms "$(PLATFORMS)" \
		$(if $(filter true,$(CLEAN)),--clean) \
		$(if $(filter true,$(DEBUG)),--debug)

## Build for Windows only
build-windows: check-deps
	@$(MAKE) build PLATFORMS=windows/amd64

## Build for Linux only  
build-linux: check-deps
	@$(MAKE) build PLATFORMS=linux/amd64

## Build for Windows and Linux (64-bit)
build-all: check-deps
	@$(MAKE) build PLATFORMS=windows/amd64,linux/amd64

## Build for all common platforms
build-multiplatform: check-deps
	@$(MAKE) build PLATFORMS=windows/amd64,windows/arm64,linux/amd64,linux/arm64,darwin/amd64,darwin/arm64

## Clean build directory
clean:
	@echo -e "$(YELLOW)Cleaning build directory...$(NC)"
	@rm -rf build/bin/*
	@echo -e "$(GREEN)✅ Build directory cleaned$(NC)"

## Run frontend in development mode
dev-frontend:
	@echo -e "$(BLUE)Starting frontend development server...$(NC)"
	@cd frontend && npm run dev

## Run wails in development mode
dev: install-deps
	@echo -e "$(BLUE)Starting Wails development server...$(NC)"
	@wails dev

## Run tests (if any)
test:
	@echo -e "$(BLUE)Running tests...$(NC)"
	@go test ./... -v

## Build and run server for development
run-server:
	@echo -e "$(BLUE)Starting patch server...$(NC)"
	@cd server && go run main.go

## Show available build targets/platforms
targets:
	@echo "Available build targets:"
	@echo "  windows/amd64"
	@echo "  windows/arm64"
	@echo "  linux/amd64"
	@echo "  linux/arm64"
	@echo "  darwin/amd64"
	@echo "  darwin/arm64"