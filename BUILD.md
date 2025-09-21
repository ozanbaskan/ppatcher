# PPatcher Build System

This document explains how to build custom PPatcher client executables for Windows and Linux with your own configurations.

## Overview

The PPatcher build system allows you to create customized client executables that can connect to your own patch server and launch your specific game or application. Users can configure:

- Backend server URL
- Executable path to launch
- UI color palette
- Build mode (production/development)
- Output executable name

## Prerequisites

Before building, ensure you have the following installed:

1. **Go** (version 1.19 or later) - [Download](https://golang.org/dl/)
2. **Node.js and npm** - [Download](https://nodejs.org/)
3. **Wails CLI** - Install with: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Quick Prerequisites Setup

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Install frontend dependencies (run from project root)
cd frontend && npm install && cd ..
```

Or use the Makefile:

```bash
make install-deps
```

## Configuration

### Creating a Configuration File

Create a configuration file (e.g., `my-config.json`) with your specific settings:

```json
{
  "backend": "https://your-server.com",
  "executable": "path/to/your/game.exe",
  "colorPalette": "blue",
  "mode": "production",
  "outputName": "my-patcher",
  "version": "1.0.0",
  "description": "My Game Patcher"
}
```

#### Configuration Options

- **`backend`**: URL of your patch server (e.g., `"https://patches.yourgame.com"`)
- **`executable`**: Path to the executable the patcher should launch (relative to the patcher location)
- **`colorPalette`**: UI color theme - `"green"`, `"blue"`, `"red"`, etc.
- **`mode`**: Build mode - `"production"` or `"dev"`
- **`outputName`**: Name for the output executable (without extension)
- **`version`**: Version string for the executable
- **`description`**: Description for the executable metadata

### Quick Config Creation

```bash
# Using the build script
./build-client.sh --create-config=my-config.json

# Using Makefile
make create-config CONFIG_FILE=my-config.json
```

## Building

There are three ways to build PPatcher clients:

### 1. Using the Build Script (Recommended)

The build script `build-client.sh` provides the most user-friendly interface:

```bash
# Basic build (Windows and Linux 64-bit)
./build-client.sh --config=my-config.json

# Build for specific platforms
./build-client.sh --config=my-config.json --platforms=windows/amd64,linux/amd64

# Clean build with debug mode
./build-client.sh --config=my-config.json --clean --debug

# Show available options
./build-client.sh --help

# Show available platforms
./build-client.sh --targets
```

#### Build Script Options

- `-c, --config FILE`: Path to config file (default: `config.json`)
- `-p, --platforms LIST`: Comma-separated list of platforms (default: `windows/amd64,linux/amd64`)
- `-C, --clean`: Clean build directory before building
- `-d, --debug`: Build in debug mode
- `-h, --help`: Show help
- `--create-config FILE`: Create sample config file
- `--targets`: Show available build targets

### 2. Using Makefile

The Makefile provides convenient targets for common build scenarios:

```bash
# Basic build
make build CONFIG_FILE=my-config.json

# Build for Windows only
make build-windows CONFIG_FILE=my-config.json

# Build for Linux only
make build-linux CONFIG_FILE=my-config.json

# Build for both Windows and Linux
make build-all CONFIG_FILE=my-config.json

# Build for all platforms
make build-multiplatform CONFIG_FILE=my-config.json

# Clean build
make build CONFIG_FILE=my-config.json CLEAN=true

# Debug build
make build CONFIG_FILE=my-config.json DEBUG=true

# Show help
make help
```

### 3. Using Wails CLI Directly

For advanced users, you can use wails directly:

```bash
# Copy your config to config.json
cp my-config.json config.json

# Build for Windows 64-bit
wails build --platform windows/amd64 --o my-patcher-windows-amd64.exe

# Build for Linux 64-bit
wails build --platform linux/amd64 --o my-patcher-linux-amd64

# Clean build
wails build --clean --platform windows/amd64 --o my-patcher-windows-amd64.exe
```

## Supported Platforms

The build system supports the following platforms:

- `windows/amd64` - Windows 64-bit (Intel/AMD)
- `windows/arm64` - Windows 64-bit (ARM)
- `linux/amd64` - Linux 64-bit (Intel/AMD)
- `linux/arm64` - Linux 64-bit (ARM)
- `darwin/amd64` - macOS 64-bit (Intel)
- `darwin/arm64` - macOS 64-bit (Apple Silicon)

## Output

Built executables are placed in the `build/bin/` directory with the following naming convention:

- Windows: `{outputName}-{os}-{arch}.exe`
- Linux/macOS: `{outputName}-{os}-{arch}`

For example:
- `my-patcher-windows-amd64.exe`
- `my-patcher-linux-amd64`

## Example Workflows

### Building for Game Distribution

1. Set up your patch server
2. Create a configuration file:
   ```bash
   ./build-client.sh --create-config=game-patcher-config.json
   ```
3. Edit the config with your settings:
   ```json
   {
     "backend": "https://patches.yourgame.com",
     "executable": "game/yourgame.exe",
     "colorPalette": "blue",
     "mode": "production",
     "outputName": "yourgame-patcher",
     "version": "2.1.0",
     "description": "Your Game Patcher"
   }
   ```
4. Build for your target platforms:
   ```bash
   ./build-client.sh --config=game-patcher-config.json --platforms=windows/amd64,linux/amd64
   ```

### Development Workflow

1. Use development mode for testing:
   ```json
   {
     "backend": "http://localhost:3000",
     "executable": "testprogram",
     "mode": "dev"
   }
   ```
2. Build with debug symbols:
   ```bash
   ./build-client.sh --config=dev-config.json --debug
   ```

### Multi-Environment Setup

Create different configs for different environments:

```bash
# Production
./build-client.sh --create-config=prod-config.json
# Edit for production settings

# Staging  
./build-client.sh --create-config=staging-config.json
# Edit for staging settings

# Development
./build-client.sh --create-config=dev-config.json
# Edit for development settings

# Build all
./build-client.sh --config=prod-config.json
./build-client.sh --config=staging-config.json  
./build-client.sh --config=dev-config.json
```

## Troubleshooting

### Common Issues

1. **"wails command not found"**
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```

2. **"npm not found"**
   - Install Node.js from https://nodejs.org/

3. **Frontend build fails**
   ```bash
   cd frontend && npm install && cd ..
   ```

4. **Permission denied on Linux/macOS**
   ```bash
   chmod +x build-client.sh
   ```

5. **Build fails with "config not found"**
   - Ensure your config file exists and is valid JSON
   - Use `--create-config` to generate a sample

### Getting Help

- Run `./build-client.sh --help` for build script options
- Run `make help` for Makefile targets
- Check `wails build --help` for advanced wails options

## Advanced Usage

### Custom Build Tags

```bash
wails build --tags "production,custom" --platform windows/amd64
```

### Cross-Compilation Requirements

For cross-compilation to work properly:

1. **Windows builds on Linux**: Install `gcc-multilib` and Windows cross-compiler
2. **Linux builds on Windows**: Use WSL or Docker
3. **macOS builds**: Require macOS host (due to Cocoa dependencies)

### Environment Variables

You can override config values with environment variables:

- `BACKEND`: Override backend URL
- `EXECUTABLE`: Override executable path  
- `COLOR_PALETTE`: Override color palette
- `MODE`: Override mode

Example:
```bash
BACKEND=https://staging.yourgame.com ./build-client.sh --config=my-config.json
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Build PPatcher Clients
on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
      with:
        go-version: '1.21'
    - uses: actions/setup-node@v3
      with:
        node-version: '18'
    
    - name: Install dependencies
      run: make install-deps
      
    - name: Build clients
      run: |
        make create-config CONFIG_FILE=prod-config.json
        # Edit config here or use sed/jq
        make build-all CONFIG_FILE=prod-config.json
        
    - name: Upload artifacts
      uses: actions/upload-artifact@v3
      with:
        name: ppatcher-clients
        path: build/bin/
```

This build system provides flexibility for different deployment scenarios while maintaining ease of use for end users.