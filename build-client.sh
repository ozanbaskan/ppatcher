#!/bin/bash

# PPatcher Build Script
# This script allows users to build PPatcher client executables for Windows and Linux
# with custom configurations.

set -e

# Default values
CONFIG_FILE="config.json"
PLATFORMS="windows/amd64,linux/amd64"
CLEAN=false
DEBUG=false
HELP=false
CREATE_CONFIG=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to show help
show_help() {
    echo "PPatcher Build Script"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -c, --config FILE        Path to config.json file (default: config.json)"
    echo "  -p, --platforms LIST     Comma-separated list of platforms (default: windows/amd64,linux/amd64)"
    echo "  -C, --clean             Clean build directory before building"
    echo "  -d, --debug             Build in debug mode"
    echo "  -h, --help              Show this help message"
    echo "  --create-config FILE    Create a sample config file"
    echo "  --targets               Show available build targets"
    echo ""
    echo "Available platforms:"
    echo "  windows/amd64, windows/arm64, linux/amd64, linux/arm64, darwin/amd64, darwin/arm64"
    echo ""
    echo "Examples:"
    echo "  $0 --config=my-config.json"
    echo "  $0 --platforms=windows/amd64,linux/amd64 --clean"
    echo "  $0 --create-config=my-config.json"
    echo "  $0 --targets"
}

# Function to show available targets
show_targets() {
    echo "Available build targets:"
    echo "  windows/amd64"
    echo "  windows/arm64"
    echo "  linux/amd64"
    echo "  linux/arm64"
    echo "  darwin/amd64"
    echo "  darwin/arm64"
}

# Function to create sample config
create_sample_config() {
    local config_file="$1"
    local config_dir=$(dirname "$config_file")
    
    # Create directory if it doesn't exist
    if [[ "$config_dir" != "." ]]; then
        mkdir -p "$config_dir"
    fi
    
    cat > "$config_file" << 'EOF'
{
  "backend": "http://localhost:3000",
  "executable": "your-game-executable",
  "colorPalette": "green",
  "mode": "production",
  "outputName": "ppatcher",
  "version": "1.0.0",
  "description": "PPatcher Client"
}
EOF
    
    print_success "Sample config created at: $config_file"
    print_info "Please edit the config file with your specific settings:"
    print_info "  - backend: Your patch server URL"
    print_info "  - executable: Path to your game executable"
    print_info "  - colorPalette: UI color theme (green, blue, red, etc.)"
    print_info "  - mode: Build mode (production or dev)"
    print_info "  - outputName: Name of the output executable"
}

# Function to check prerequisites
check_prerequisites() {
    # Check if wails is installed
    if ! command -v wails &> /dev/null; then
        print_error "Wails CLI not found!"
        print_info "Please install wails with: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
        exit 1
    fi
    
    # Check if go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go not found!"
        print_info "Please install Go from: https://golang.org/dl/"
        exit 1
    fi
    
    # Check if npm is installed (for frontend build)
    if ! command -v npm &> /dev/null; then
        print_error "npm not found!"
        print_info "Please install Node.js and npm from: https://nodejs.org/"
        exit 1
    fi
    
    print_success "All prerequisites are met"
}

# Function to setup environment
setup_environment() {
    print_info "Setting up build environment..."
    
    # Install frontend dependencies if needed
    if [[ ! -d "frontend/node_modules" ]]; then
        print_info "Installing frontend dependencies..."
        cd frontend
        npm install
        cd ..
    fi
    
    # Create build directory
    mkdir -p build/bin
    
    print_success "Environment setup complete"
}

# buildForTarget builds for a specific target platform
build_platform() {
    local platform="$1"
    local config_file="$2"
    local clean="$3"
    local debug="$4"
    
    local os=$(echo "$platform" | cut -d'/' -f1)
    local arch=$(echo "$platform" | cut -d'/' -f2)
    
    # Read config to get output name
    local output_name="ppatcher"
    if [[ -f "$config_file" ]]; then
        if command -v jq &> /dev/null; then
            local config_output_name=$(jq -r '.outputName // "ppatcher"' "$config_file")
            if [[ "$config_output_name" != "null" ]]; then
                output_name="$config_output_name"
            fi
        fi
    fi
    
    # Set output filename
    local output_file="${output_name}-${os}-${arch}"
    if [[ "$os" == "windows" ]]; then
        output_file="${output_file}.exe"
    fi
    
    print_info "Building for $platform..."
    
    # Build wails command
    local wails_cmd="wails build"
    
    if [[ "$clean" == "true" ]]; then
        wails_cmd="$wails_cmd --clean"
    fi
    
    if [[ "$debug" == "true" ]]; then
        wails_cmd="$wails_cmd --debug"
    fi
    
    wails_cmd="$wails_cmd --platform $platform --o $output_file"
    
    print_info "Running: $wails_cmd"
    
    # Execute build command and capture result
    if $wails_cmd > /tmp/wails_build.log 2>&1; then
        local built_file="build/bin/$output_file"
        if [[ -f "$built_file" ]]; then
            print_success "Build successful for $platform: $built_file"
            return 0
        else
            print_error "Build completed but output file not found: $built_file"
            return 1
        fi
    else
        print_error "Build failed for $platform"
        if [[ -f /tmp/wails_build.log ]]; then
            print_error "Build output:"
            cat /tmp/wails_build.log
        fi
        return 1
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--config)
            CONFIG_FILE="$2"
            shift 2
            ;;
        --config=*)
            CONFIG_FILE="${1#*=}"
            shift
            ;;
        -p|--platforms)
            PLATFORMS="$2"
            shift 2
            ;;
        --platforms=*)
            PLATFORMS="${1#*=}"
            shift
            ;;
        -C|--clean)
            CLEAN=true
            shift
            ;;
        -d|--debug)
            DEBUG=true
            shift
            ;;
        -h|--help)
            HELP=true
            shift
            ;;
        --create-config)
            CREATE_CONFIG="$2"
            shift 2
            ;;
        --create-config=*)
            CREATE_CONFIG="${1#*=}"
            shift
            ;;
        --targets)
            show_targets
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Show help if requested
if [[ "$HELP" == "true" ]]; then
    show_help
    exit 0
fi

# Create sample config if requested
if [[ -n "$CREATE_CONFIG" ]]; then
    create_sample_config "$CREATE_CONFIG"
    exit 0
fi

# Check if config file exists
if [[ ! -f "$CONFIG_FILE" ]]; then
    print_error "Config file not found: $CONFIG_FILE"
    print_info "Create one with: $0 --create-config=$CONFIG_FILE"
    exit 1
fi

# Main build process
print_info "Starting PPatcher build process..."
print_info "Config file: $CONFIG_FILE"
print_info "Platforms: $PLATFORMS"
print_info "Clean: $CLEAN"
print_info "Debug: $DEBUG"

# Check prerequisites
check_prerequisites

# Setup environment
setup_environment

# Backup original config and use build config
if [[ "$CONFIG_FILE" != "config.json" ]]; then
    if [[ -f "config.json" ]]; then
        cp "config.json" "config.json.bak"
    fi
    cp "$CONFIG_FILE" "config.json"
fi

# Ensure we restore the original config on exit
cleanup() {
    if [[ -f "config.json.bak" ]]; then
        mv "config.json.bak" "config.json"
    fi
}
trap cleanup EXIT

# Build for each platform
successful_builds=0
failed_builds=0
built_files=()

IFS=',' read -ra PLATFORM_ARRAY <<< "$PLATFORMS"
for platform in "${PLATFORM_ARRAY[@]}"; do
    platform=$(echo "$platform" | xargs) # trim whitespace
    
    if build_platform "$platform" "$CONFIG_FILE" "$CLEAN" "$DEBUG"; then
        ((successful_builds++))
        
        output_name="ppatcher"
        if [[ -f "$CONFIG_FILE" ]]; then
            if command -v jq &> /dev/null; then
                config_output_name=$(jq -r '.outputName // "ppatcher"' "$CONFIG_FILE")
                if [[ "$config_output_name" != "null" ]]; then
                    output_name="$config_output_name"
                fi
            fi
        fi
        
        os=$(echo "$platform" | cut -d'/' -f1)
        arch=$(echo "$platform" | cut -d'/' -f2)
        output_file="${output_name}-${os}-${arch}"
        if [[ "$os" == "windows" ]]; then
            output_file="${output_file}.exe"
        fi
        built_files+=("build/bin/$output_file")
    else
        ((failed_builds++))
    fi
done

# Print summary
echo ""
echo "============================================================="
echo "BUILD SUMMARY"
echo "============================================================="

for file in "${built_files[@]}"; do
    print_success "✅ $(basename "$file"): SUCCESS ($file)"
done

if [[ $failed_builds -gt 0 ]]; then
    print_warning "❌ $failed_builds build(s) failed"
fi

echo ""
print_info "Results: $successful_builds successful, $failed_builds failed"

if [[ $successful_builds -gt 0 ]]; then
    echo ""
    print_success "🎉 Build process completed! Your executables are in build/bin/"
    print_info "You can now distribute these executables to your users."
fi

# Exit with error code if any builds failed
if [[ $failed_builds -gt 0 ]]; then
    exit 1
fi

exit 0