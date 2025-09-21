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

Create a configuration file (e.g., `my-config.json`) with your specific settings. The configuration file supports all aspects of client customization:

```json
{
  "backend": "https://patches.yourgame.com",
  "executable": "game/yourgame",
  "colorPalette": "purple",
  "mode": "production",
  "outputName": "yourgame-patcher",
  "version": "v3.2.1",
  "description": "Your Game Patcher",
  "logo": "https://cdn.yourgame.com/logo.png",
  "icon": "assets/app-icon.ico"
}
```

#### Complete Configuration Options

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| **`backend`** | String | URL of your patch server | `"https://patches.yourgame.com"` |
| **`executable`** | String | Path to executable to launch (relative to patcher) | `"game/yourgame"` (`.exe` added automatically on Windows) |
| **`colorPalette`** | String | UI color theme | `"green"`, `"blue"`, `"red"`, `"purple"`, etc. |
| **`mode`** | String | Build mode | `"production"` or `"dev"` |
| **`outputName`** | String | Name for output executable (without extension) | `"yourgame-patcher"` |
| **`version`** | String | Version displayed in UI and executable metadata | `"v3.2.1"`, `"2.0.0"` |
| **`description`** | String | App title shown in UI and executable metadata | `"Your Game Patcher"` |
| **`logo`** | String | Path or URL to logo image for client UI | `"assets/logo.png"`, `"https://example.com/logo.png"` |
| **`icon`** | String | Path or URL to app icon for executable | `"assets/icon.ico"`, `"https://example.com/icon.png"` |

#### Branding and UI Customization

**Dynamic UI Elements:**
- The `description` field becomes the main title shown in the client interface
- The `version` is displayed in the footer as "Description Version" 
- Custom `logo` replaces the default PPatcher logo in the interface
- The `colorPalette` changes the entire UI theme and color scheme

**Asset Handling:**
- **Local files**: Specify relative paths like `"assets/logo.png"` or `"../branding/icon.ico"`
- **Remote URLs**: Use HTTP/HTTPS URLs like `"https://cdn.example.com/logo.png"`
- **Automatic processing**: Images are downloaded/copied during build and integrated into the executable
- **Format support**: PNG, JPG, ICO, and other common image formats

#### Platform-Specific Behavior

**Windows:**
- Executable names automatically get `.exe` extension (e.g., `"mygame"` becomes `"mygame.exe"`)
- Custom icons are embedded in the executable file
- Version and description appear in file properties

**Linux:**
- Executable names used as-is (no automatic extensions)
- Desktop integration with custom icons
- AppImage packaging support

**macOS:**
- App bundle creation with custom icons
- Info.plist integration with version and description
- Native macOS look and feel

#### Common Configuration Examples

**Minimal Setup (Development):**
```json
{
  "backend": "http://localhost:3000",
  "executable": "test-app",
  "colorPalette": "green",
  "mode": "dev"
}
```

**Professional Game Patcher:**
```json
{
  "backend": "https://patches.coolgame.com",
  "executable": "game/CoolGame",
  "colorPalette": "purple",
  "mode": "production",
  "outputName": "CoolGame-Updater",
  "version": "v4.2.1",
  "description": "Cool Game Updater",
  "logo": "https://cdn.coolgame.com/updater-logo.png",
  "icon": "https://cdn.coolgame.com/updater-icon.ico"
}
```

**Enterprise Software Updater:**
```json
{
  "backend": "https://updates.company.com/software-suite",
  "executable": "bin/enterprise-app.exe",
  "colorPalette": "blue", 
  "mode": "production",
  "outputName": "CompanyApp-Updater",
  "version": "v2024.1",
  "description": "Company Software Updater",
  "logo": "assets/company-logo.png",
  "icon": "assets/company-icon.ico"
}
```

**Multi-Environment Template:**

Development:
```json
{
  "backend": "http://localhost:8080",
  "executable": "debug/app-debug",
  "colorPalette": "green",
  "mode": "dev",
  "outputName": "app-dev-updater", 
  "version": "dev-build",
  "description": "App Development Updater"
}
```

Staging:
```json
{
  "backend": "https://staging-api.company.com",
  "executable": "app/application",
  "colorPalette": "orange",
  "mode": "production",
  "outputName": "app-staging-updater",
  "version": "v3.1.0-rc2", 
  "description": "App Staging Updater",
  "logo": "assets/staging-logo.png"
}
```

Production:
```json
{
  "backend": "https://api.company.com",
  "executable": "app/application", 
  "colorPalette": "blue",
  "mode": "production",
  "outputName": "app-updater",
  "version": "v3.1.0",
  "description": "App Updater", 
  "logo": "https://cdn.company.com/logo.png",
  "icon": "https://cdn.company.com/icon.ico"
}
```

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

### Complete Game Patcher Setup

This example shows how to create a fully branded game patcher with custom logo and icon:

1. **Create your configuration file:**
   ```bash
   ./build-client.sh --create-config=mygame-patcher-config.json
   ```

2. **Edit the config with complete branding:**
   ```json
   {
     "backend": "https://patches.mygame.com",
     "executable": "game/MyAwesomeGame",
     "colorPalette": "purple",
     "mode": "production",
     "outputName": "MyAwesomeGame-Updater",
     "version": "v4.1.2",
     "description": "My Awesome Game Updater",
     "logo": "https://cdn.mygame.com/branding/logo.png",
     "icon": "https://cdn.mygame.com/branding/app-icon.ico"
   }
   ```

3. **Build for multiple platforms:**
   ```bash
   ./build-client.sh --config=mygame-patcher-config.json --platforms=windows/amd64,linux/amd64,darwin/amd64
   ```

**Result**: Creates branded executables like:
- `MyAwesomeGame-Updater-windows-amd64.exe` (with custom icon)
- `MyAwesomeGame-Updater-linux-amd64` (with custom branding)
- `MyAwesomeGame-Updater-darwin-amd64` (macOS app bundle)

Each executable will:
- Display "My Awesome Game Updater v4.1.2" as the title
- Use your custom logo in the interface
- Have a purple color theme throughout
- Launch `MyAwesomeGame.exe` (Windows) or `MyAwesomeGame` (Linux/macOS) after updates

### Multi-Environment Development

Create different configurations for different deployment stages:

```bash
# Development environment
./build-client.sh --create-config=dev-config.json
```

```json
{
  "backend": "http://localhost:3000",
  "executable": "test/game-debug",
  "colorPalette": "green",
  "mode": "dev",
  "outputName": "game-dev-patcher",
  "version": "dev-build",
  "description": "Game Development Patcher",
  "logo": "",
  "icon": ""
}
```

```bash
# Staging environment  
./build-client.sh --create-config=staging-config.json
```

```json
{
  "backend": "https://staging-patches.mygame.com",
  "executable": "game/mygame",
  "colorPalette": "orange", 
  "mode": "production",
  "outputName": "mygame-staging-patcher",
  "version": "v3.1.0-rc1",
  "description": "My Game Staging Patcher",
  "logo": "assets/logo-staging.png",
  "icon": "assets/icon-staging.ico"
}
```

```bash
# Production environment
./build-client.sh --create-config=prod-config.json
```

```json
{
  "backend": "https://patches.mygame.com",
  "executable": "game/mygame",
  "colorPalette": "blue",
  "mode": "production", 
  "outputName": "mygame-patcher",
  "version": "v3.1.0",
  "description": "My Game Patcher",
  "logo": "https://cdn.mygame.com/logo.png",
  "icon": "https://cdn.mygame.com/icon.ico"
}
```

```bash
# Build all environments
./build-client.sh --config=dev-config.json --platforms=windows/amd64 --debug
./build-client.sh --config=staging-config.json --platforms=windows/amd64,linux/amd64
./build-client.sh --config=prod-config.json --platforms=windows/amd64,linux/amd64,darwin/amd64
```

### Local Asset Management

For projects where you manage assets locally rather than using URLs:

```bash
# Create assets directory structure
mkdir -p assets/branding
```

Directory structure:
```
your-project/
├── assets/
│   └── branding/
│       ├── logo.png          # 300x200px logo for UI
│       ├── icon.ico          # 256x256px icon for executable
│       └── icon-large.png    # High-res icon source
├── my-config.json
└── build-client.sh
```

Configuration with local assets:
```json
{
  "backend": "https://patches.yourgame.com",
  "executable": "bin/yourgame",
  "colorPalette": "blue",
  "mode": "production",
  "outputName": "yourgame-patcher", 
  "version": "v2.5.3",
  "description": "Your Game Patcher",
  "logo": "assets/branding/logo.png",
  "icon": "assets/branding/icon.ico"
}
```

### Quick Prototyping

For rapid testing and development:

```bash
# Create minimal config
./build-client.sh --create-config=quick-test.json
```

```json
{
  "backend": "http://localhost:8080",
  "executable": "notepad",
  "colorPalette": "green", 
  "mode": "dev",
  "outputName": "test-patcher",
  "version": "test",
  "description": "Test Patcher"
}
```

```bash
# Quick build for testing
./build-client.sh --config=quick-test.json --platforms=windows/amd64 --debug
```

This creates a test patcher that launches Notepad, useful for testing the update mechanism without needing a real game.

## Branding and Asset Management

### Logo and Icon Requirements

**Logo Image (`logo` field):**
- **Recommended size**: 300x200 pixels or similar aspect ratio
- **Supported formats**: PNG, JPG, JPEG, GIF
- **Usage**: Displayed in the main client interface
- **Aspect ratio**: Maintain reasonable proportions for UI display

**App Icon (`icon` field):**
- **Recommended size**: 256x256 pixels (will be resized as needed)
- **Supported formats**: ICO, PNG, JPG (ICO preferred for Windows)
- **Usage**: Used as the executable file icon
- **Platform behavior**: 
  - Windows: Embedded in .exe file, shown in taskbar and file explorer
  - Linux: Used for desktop integration and window manager
  - macOS: Used in app bundle and dock

### Asset Sources

**Local Files:**
```json
{
  "logo": "assets/branding/logo.png",
  "icon": "assets/branding/app-icon.ico"
}
```

**Remote URLs:**
```json
{
  "logo": "https://cdn.yourgame.com/branding/logo.png", 
  "icon": "https://cdn.yourgame.com/branding/icon.ico"
}
```

**Mixed Sources:**
```json
{
  "logo": "https://cdn.yourgame.com/logo.png",
  "icon": "local-assets/icon.ico" 
}
```

### Asset Processing

During the build process:

1. **Logo Processing:**
   - Downloaded/copied to `frontend/src/assets/images/logo-custom.png`
   - React component automatically updated to use custom logo
   - Original logo backed up if replacing existing

2. **Icon Processing:**
   - Downloaded/copied to `build/appicon.png`
   - Integrated into executable during wails build process
   - Automatically converted to platform-specific formats

3. **Error Handling:**
   - Failed downloads/copies log warnings but don't stop build
   - Invalid image formats are detected and reported
   - Network timeouts handled gracefully

### UI Customization

**Color Palette Options:**
- `"green"` - Default green theme
- `"blue"` - Professional blue theme  
- `"red"` - Bold red theme
- `"purple"` - Modern purple theme
- `"orange"` - Energetic orange theme
- `"teal"` - Calm teal theme

**Dynamic UI Elements:**
- **Main Title**: Uses `description` field (e.g., "My Game Patcher")
- **Footer**: Shows `description` + `version` (e.g., "My Game Patcher v2.1.0")
- **Window Title**: Uses `description` for window title bar
- **Color Scheme**: Applied consistently across all UI components

**Fallback Behavior:**
- Missing `description`: Falls back to "PPatcher"
- Missing `version`: Falls back to "v1.0.0" 
- Missing `logo`: Uses default PPatcher logo
- Missing `icon`: Uses default application icon
- Invalid `colorPalette`: Falls back to "green"

### Best Practices

**For Professional Distribution:**
```json
{
  "description": "Your Brand Game Updater",
  "version": "v2.1.0",
  "logo": "https://cdn.yourbrand.com/updater-logo.png",
  "icon": "https://cdn.yourbrand.com/updater-icon.ico",
  "colorPalette": "blue"
}
```

**For Development Testing:**
```json
{
  "description": "Dev Build Updater", 
  "version": "dev-2024.01.15",
  "logo": "",
  "icon": "",
  "colorPalette": "green"
}
```

**For Staging Environment:**
```json
{
  "description": "Your Brand Staging Updater",
  "version": "v2.1.0-rc1", 
  "logo": "assets/staging-logo.png",
  "icon": "assets/staging-icon.ico",
  "colorPalette": "orange"
}
```


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

You can override config values with environment variables for flexible deployment:

| Environment Variable | Config Field | Description | Example |
|---------------------|--------------|-------------|---------|
| `BACKEND` | `backend` | Override backend URL | `https://staging.yourgame.com` |
| `EXECUTABLE` | `executable` | Override executable path | `game/yourgame-dev` |
| `COLOR_PALETTE` | `colorPalette` | Override color palette | `red` |
| `MODE` | `mode` | Override build mode | `dev` |
| `VERSION` | `version` | Override version string | `v4.0.0-beta` |
| `DESCRIPTION` | `description` | Override description/title | `My Game Beta Patcher` |

**Usage examples:**

```bash
# Override backend for staging builds
BACKEND=https://staging.yourgame.com ./build-client.sh --config=prod-config.json

# Override multiple values for development
BACKEND=http://localhost:3000 MODE=dev VERSION=dev-build ./build-client.sh --config=my-config.json

# Create a debug version of production config
MODE=dev DESCRIPTION="My Game Debug Patcher" ./build-client.sh --config=prod-config.json --debug
```

**Note**: Environment variables take precedence over config file values, making them perfect for CI/CD pipelines and deployment automation.

## Integration with CI/CD

### GitHub Actions Example

Complete workflow for building branded patchers with all customizations:

```yaml
name: Build Branded PPatcher Clients
on: 
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        config: [production, staging, development]
        
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
      
    - name: Create configuration
      run: |
        ./build-client.sh --create-config=${{ matrix.config }}-config.json
        
    - name: Configure for production
      if: matrix.config == 'production'
      run: |
        cat > production-config.json << 'EOF'
        {
          "backend": "https://patches.yourgame.com",
          "executable": "game/yourgame",
          "colorPalette": "blue",
          "mode": "production",
          "outputName": "yourgame-patcher",
          "version": "${{ github.ref_name }}",
          "description": "Your Game Patcher",
          "logo": "https://cdn.yourgame.com/logo.png",
          "icon": "https://cdn.yourgame.com/icon.ico"
        }
        EOF
        
    - name: Configure for staging
      if: matrix.config == 'staging'
      run: |
        cat > staging-config.json << 'EOF'
        {
          "backend": "https://staging-patches.yourgame.com", 
          "executable": "game/yourgame",
          "colorPalette": "orange",
          "mode": "production",
          "outputName": "yourgame-staging-patcher",
          "version": "${{ github.ref_name }}-staging",
          "description": "Your Game Staging Patcher",
          "logo": "assets/staging-logo.png",
          "icon": "assets/staging-icon.ico"
        }
        EOF
        
    - name: Configure for development
      if: matrix.config == 'development'
      run: |
        cat > development-config.json << 'EOF'
        {
          "backend": "https://dev-patches.yourgame.com",
          "executable": "game/yourgame-dev", 
          "colorPalette": "green",
          "mode": "dev",
          "outputName": "yourgame-dev-patcher",
          "version": "dev-${{ github.sha }}",
          "description": "Your Game Dev Patcher",
          "logo": "",
          "icon": ""
        }
        EOF
        
    - name: Build clients
      run: |
        if [ "${{ matrix.config }}" == "production" ]; then
          ./build-client.sh --config=production-config.json --platforms=windows/amd64,linux/amd64,darwin/amd64
        elif [ "${{ matrix.config }}" == "staging" ]; then
          ./build-client.sh --config=staging-config.json --platforms=windows/amd64,linux/amd64
        else
          ./build-client.sh --config=development-config.json --platforms=windows/amd64 --debug
        fi
        
    - name: Upload artifacts
      uses: actions/upload-artifact@v3
      with:
        name: ppatcher-clients-${{ matrix.config }}
        path: build/bin/
        
    - name: Create release (production only)
      if: matrix.config == 'production' && startsWith(github.ref, 'refs/tags/')
      uses: softprops/action-gh-release@v1
      with:
        files: build/bin/*
        name: "Your Game Patcher ${{ github.ref_name }}"
        body: |
          ## Your Game Patcher ${{ github.ref_name }}
          
          Branded patcher clients for Your Game.
          
          ### Downloads
          - Windows: `yourgame-patcher-windows-amd64.exe`
          - Linux: `yourgame-patcher-linux-amd64`
          - macOS: `yourgame-patcher-darwin-amd64`
          
          ### Features
          - Custom branding with Your Game logo and colors
          - Automatic game launching after updates
          - Cross-platform support
```

### GitLab CI Example

```yaml
stages:
  - build
  - deploy

variables:
  GO_VERSION: "1.21"
  NODE_VERSION: "18"

.build_template: &build_template
  stage: build
  image: golang:${GO_VERSION}
  before_script:
    - apt-get update && apt-get install -y nodejs npm curl
    - npm install -g npm@latest
    - make install-deps
  script:
    - ./build-client.sh --create-config=${ENVIRONMENT}-config.json
    - echo "$CONFIG_JSON" > ${ENVIRONMENT}-config.json
    - ./build-client.sh --config=${ENVIRONMENT}-config.json --platforms=${PLATFORMS}
  artifacts:
    paths:
      - build/bin/
    expire_in: 1 week

build_production:
  <<: *build_template
  variables:
    ENVIRONMENT: production
    PLATFORMS: "windows/amd64,linux/amd64,darwin/amd64"
    CONFIG_JSON: |
      {
        "backend": "https://patches.yourgame.com",
        "executable": "game/yourgame",
        "colorPalette": "blue", 
        "mode": "production",
        "outputName": "yourgame-patcher",
        "version": "${CI_COMMIT_TAG:-v1.0.0}",
        "description": "Your Game Patcher",
        "logo": "https://cdn.yourgame.com/logo.png",
        "icon": "https://cdn.yourgame.com/icon.ico"
      }
  only:
    - main
    - tags

build_staging:
  <<: *build_template
  variables:
    ENVIRONMENT: staging
    PLATFORMS: "windows/amd64,linux/amd64"
    CONFIG_JSON: |
      {
        "backend": "https://staging.yourgame.com", 
        "executable": "game/yourgame",
        "colorPalette": "orange",
        "mode": "production",
        "outputName": "yourgame-staging-patcher", 
        "version": "${CI_COMMIT_SHORT_SHA}",
        "description": "Your Game Staging Patcher",
        "logo": "assets/staging-logo.png",
        "icon": "assets/staging-icon.ico"
      }
  only:
    - develop
```

These examples demonstrate how to:
- Build multiple configurations in parallel
- Use environment-specific settings
- Handle secrets and configuration via CI variables
- Create releases with proper naming
- Upload build artifacts for distribution
- Handle different branching strategies (main/develop/feature)

This build system provides flexibility for different deployment scenarios while maintaining ease of use for end users.