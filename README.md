# PPatcher

## 🚀 Blazingly fast file patcher

<img width="962" height="576" alt="image" src="https://github.com/user-attachments/assets/1bf376bf-23bf-42c9-b2a7-fedca51f6bec" />

## Tutorial

https://github.com/user-attachments/assets/7d2722f3-c8f6-470d-8164-9a305d3577d4


A modern, cross-platform game patcher built with Wails and Go. PPatcher provides a sleek interface for downloading and applying game updates with full customization support for creating branded client executables.

## Features

- **Cross-platform support** (Windows, Linux, macOS)
- **Real-time download progress** tracking with visual indicators
- **Ready to go backend server** Compile and run the executable on any platform
- **Automatic file integrity verification** with checksum validation
- **Custom executable launching** after updates complete
- **Customizable UI color themes** (green, blue, red, purple, etc.)
- **Dynamic branding** with custom titles, versions, and descriptions
- **Custom logo and icon support** for fully branded clients

## Building Custom Clients

PPatcher includes a comprehensive build system that allows you to create fully customized client executables for your own games or applications. The system supports complete branding and configuration through a single config file.

### Quick Start

```bash
# Build your backend
go build -o fileserver ./server/main.go

# Put your files inside a folder called "files" and put your backend in the same directory
# your-game-root-dir/
# ├── files/
# └── fileserver*

# Run your backend
PORT=3000 ./fileserver

# Create a configuration file with all available options and build your client
./build-client.sh --create-config=my-game-config.json

# You are ready to distribute it to your friends

# You can also checkout what is available
./build-client.sh -h
```

### Complete Configuration Example

```json
{
  "backend": "https://patches.mygame.com",
  "executable": "game/mygame",
  "colorPalette": "purple", 
  "mode": "production",
  "outputName": "mygame-patcher",
  "version": "v3.2.1",
  "description": "My Awesome Game Patcher",
  "logo": "https://cdn.example.com/logo.png",
  "icon": "assets/app-icon.ico"
}
```

This configuration creates a fully branded patcher that:
- ✅ Connects to your patch server at `https://patches.mygame.com`
- ✅ Launches `mygame.exe` (automatically adds `.exe` on Windows)
- ✅ Uses a purple color theme throughout the UI
- ✅ Shows "My Awesome Game Patcher v3.2.1" as the title and version
- ✅ Uses your custom logo in the client interface
- ✅ Has a custom icon for the executable file
- ✅ Outputs as `mygame-patcher-windows-amd64.exe` and similar for other platforms

### Build System Features

The build system supports:

#### **Multi-Platform Builds**
- Windows (x64, ARM64) with automatic `.exe` handling
- Linux (x64, ARM64) with GTK support  
- macOS (Intel, Apple Silicon) with native integration

#### **Complete Customization**
- **Backend URLs**: Point to your own patch server
- **Game executables**: Configure any executable to launch after patching
- **UI themes**: Choose from multiple color palettes or specify custom colors
- **Branding**: Custom titles, versions, descriptions displayed in the UI
- **Visual assets**: Custom logos and executable icons
- **Output naming**: Control the name of generated executable files

#### **Asset Management**
- **Local files**: Use files from your project directory
- **Remote URLs**: Download logos and icons from CDNs or websites  
- **Automatic processing**: Images are downloaded/copied and integrated during build
- **Format flexibility**: Support for PNG, JPG, ICO, and other common formats

#### **Build Modes**
- **Production**: Optimized builds for distribution
- **Development**: Debug builds with additional logging
- **Clean builds**: Start fresh by clearing previous build artifacts
- **Debug mode**: Include development symbols and verbose output

#### **Multiple Interfaces**
- **Build script**: User-friendly `./build-client.sh` with comprehensive options
- **Makefile**: Standard `make` targets for developers
- **Direct wails**: Advanced users can call wails CLI directly
- **CI/CD integration**: Easy integration with GitHub Actions, GitLab CI, etc.

### Platform-Specific Features

#### Windows
- Automatic `.exe` suffix addition to executable paths
- Native Windows installer generation (NSIS)
- Custom executable icons and metadata
- Windows-specific file associations

#### Linux  
- GTK-based UI with native look and feel
- Standard executable distribution
- Compatible with major Linux distributions

#### macOS
- Native macOS application bundle generation
- Code signing and notarization support
- Standard .app bundle distribution

## Development

For development setup, detailed build instructions, troubleshooting, and advanced configuration options, see [`BUILD.md`](BUILD.md).

### Development Quick Start

```bash
# Install dependencies
make install-deps

# Create a development config
./build-client.sh --create-config=dev-config.json

# Edit dev-config.json for local development:
# {
#   "backend": "http://localhost:3000",
#   "executable": "test-program",
#   "mode": "dev",
#   "colorPalette": "green"
# }

# Build and test
./build-client.sh --config=dev-config.json --debug
```

## Use Cases

### Game Developers
- Create branded patchers for your games
- Distribute updates through your own servers  
- Customize the look and feel to match your game's theme
- Support multiple platforms with a single configuration

### Software Publishers
- Build update systems for desktop applications
- Maintain version control across different deployment environments
- Create staging and production update channels
- Integrate with existing CI/CD pipelines

### System Administrators  
- Deploy custom update tools for internal software
- Manage software deployments across different environments
- Create branded tools for client organizations
- Maintain consistent branding across multiple projects

## Examples

See the `demo-config.json` file for a complete configuration example, and check [`BUILD.md`](BUILD.md) for detailed workflows including:

- Single-platform builds for specific operating systems
- Multi-platform builds for cross-platform distribution  
- Development vs. production build configurations
- CI/CD integration examples with GitHub Actions
- Advanced customization with environment variable overrides

## License

This project is open source. Please check the repository for license details.
