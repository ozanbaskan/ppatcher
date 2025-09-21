# PPatcher

<img width="960" height="576" alt="PPatcher Screenshot" src="https://github.com/user-attachments/assets/2ef50606-c1c8-4c02-8d26-e9b0d5624cf3" />

A modern, cross-platform game patcher built with Wails and Go. PPatcher provides a sleek interface for downloading and applying game updates.

## Features

- Cross-platform support (Windows, Linux, macOS)
- Real-time download progress tracking
- Configurable backend server support
- Automatic file integrity verification
- Custom executable launching
- Customizable UI color themes

## Building Custom Clients

PPatcher includes a comprehensive build system that allows you to create customized client executables for your own games or applications. See [`BUILD.md`](BUILD.md) for detailed instructions.

### Quick Start

```bash
# Create a configuration file
./build-client.sh --create-config=my-game-config.json

# Edit the config with your settings, then build
./build-client.sh --config=my-game-config.json

# Or use the Makefile
make create-config CONFIG_FILE=my-game-config.json
make build CONFIG_FILE=my-game-config.json
```

The build system supports:
- Custom backend server URLs
- Configurable game executable paths
- UI color theme customization
- Multiple target platforms (Windows, Linux, macOS)
- Both x64 and ARM64 architectures

## Development

For development setup and detailed build instructions, see [`BUILD.md`](BUILD.md).

## License

This project is open source. Please check the repository for license details.
