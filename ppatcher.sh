#!/bin/bash

# ppatcher.sh - Simplified PPatcher workflow script
# Sets up servers, creates configs, and builds patcher clients in one place.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

info()    { echo -e "${BLUE}→${NC} $1"; }
success() { echo -e "${GREEN}✓${NC} $1"; }
warn()    { echo -e "${YELLOW}!${NC} $1"; }
error()   { echo -e "${RED}✗${NC} $1"; }
header()  { echo -e "\n${BOLD}${CYAN}$1${NC}\n"; }

usage() {
    echo -e "${BOLD}ppatcher.sh${NC} — Simplified PPatcher workflow
${BOLD}USAGE${NC}
    ./ppatcher.sh <command> [options]

${BOLD}COMMANDS${NC}
    ${GREEN}init${NC} <name>              Create a new project directory with server + client config
    ${GREEN}server${NC} <project-dir>     Start the patch server for a project
    ${GREEN}build${NC} <project-dir>      Build the patcher client for distribution
    ${GREEN}setup${NC}                     Launch interactive setup wizard (web UI)
    ${GREEN}dev${NC}                      Start the Wails dev environment
    ${GREEN}status${NC} <project-dir>     Show project info (files, config, server URL)
    ${GREEN}help${NC}                     Show this help message

${BOLD}WORKFLOW${NC}
    1. ${CYAN}./ppatcher.sh init my-game${NC}
       Creates my-game/ with files/ directory and config template.

    2. Put your game/app files into ${CYAN}my-game/files/${NC}

    3. ${CYAN}./ppatcher.sh server my-game${NC}
       Starts the file server. It watches for file changes automatically.

    4. Edit ${CYAN}my-game/config.json${NC} with your server URL, branding, etc.

    5. ${CYAN}./ppatcher.sh build my-game${NC}
       Builds patcher client executables ready for distribution.

${BOLD}EXAMPLES${NC}
    ./ppatcher.sh init my-game
    ./ppatcher.sh init my-game --port 8080
    ./ppatcher.sh server my-game
    ./ppatcher.sh server my-game --port 4000
    ./ppatcher.sh build my-game
    ./ppatcher.sh build my-game --platforms linux/amd64,windows/amd64
    ./ppatcher.sh status my-game
    ./ppatcher.sh dev"
}

# ──────────────────────────────────────────────────────────────────────────────
# INIT: Create a new project directory
# ──────────────────────────────────────────────────────────────────────────────
cmd_init() {
    local name=""
    local port="3000"

    while [[ $# -gt 0 ]]; do
        case $1 in
            --port) port="$2"; shift 2 ;;
            --port=*) port="${1#*=}"; shift ;;
            -*) error "Unknown option: $1"; exit 1 ;;
            *) name="$1"; shift ;;
        esac
    done

    if [[ -z "$name" ]]; then
        error "Project name required: ./ppatcher.sh init <name>"
        exit 1
    fi

    local project_dir="$name"

    if [[ -d "$project_dir" ]]; then
        error "Directory '$project_dir' already exists"
        exit 1
    fi

    header "Creating project: $name"

    mkdir -p "$project_dir/files"

    # Create client build config
    cat > "$project_dir/config.json" <<EOF
{
  "backend": "http://localhost:$port",
  "executable": "",
  "colorPalette": "neutral",
  "mode": "production",
  "outputName": "$name-patcher",
  "version": "1.0.0",
  "description": "Keep your files up to date",
  "displayName": "$name",
  "title": "$name Patcher",
  "logo": "",
  "icon": ""
}
EOF

    success "Created $project_dir/"
    success "Created $project_dir/files/        ← put your files here"
    success "Created $project_dir/config.json   ← client configuration"

    echo ""
    info "Next steps:"
    echo "  1. Add files to $project_dir/files/"
    echo "  2. Start the server:  ./ppatcher.sh server $project_dir"
    echo "  3. Edit config:       $project_dir/config.json"
    echo "  4. Build the client:  ./ppatcher.sh build $project_dir"
}

# ──────────────────────────────────────────────────────────────────────────────
# SERVER: Start the patch server for a project
# ──────────────────────────────────────────────────────────────────────────────
cmd_server() {
    local project_dir=""
    local port=""
    local build_server=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --port) port="$2"; shift 2 ;;
            --port=*) port="${1#*=}"; shift ;;
            --build) build_server=true; shift ;;
            -*) error "Unknown option: $1"; exit 1 ;;
            *) project_dir="$1"; shift ;;
        esac
    done

    if [[ -z "$project_dir" ]]; then
        error "Project directory required: ./ppatcher.sh server <project-dir>"
        exit 1
    fi

    if [[ ! -d "$project_dir" ]]; then
        error "Directory '$project_dir' not found. Run './ppatcher.sh init $project_dir' first."
        exit 1
    fi

    # Resolve to absolute path
    project_dir="$(cd "$project_dir" && pwd)"

    # Detect port from config if not specified
    if [[ -z "$port" && -f "$project_dir/config.json" ]]; then
        if command -v jq &> /dev/null; then
            local backend_url
            backend_url=$(jq -r '.backend // empty' "$project_dir/config.json")
            if [[ -n "$backend_url" ]]; then
                port=$(echo "$backend_url" | grep -oP ':\K[0-9]+$' || true)
            fi
        fi
    fi
    port="${port:-3000}"

    local files_dir="$project_dir/files"
    if [[ ! -d "$files_dir" ]]; then
        mkdir -p "$files_dir"
        warn "Created empty $files_dir/"
    fi

    local file_count
    file_count=$(find "$files_dir" -type f 2>/dev/null | wc -l)

    header "Starting server for: $(basename "$project_dir")"
    info "Files directory: $files_dir ($file_count files)"
    info "Port: $port"
    info "URL: http://localhost:$port"
    echo ""

    # Check if we have a prebuilt server binary in the project
    local server_bin=""
    if [[ -f "$project_dir/fileserver" ]]; then
        server_bin="$project_dir/fileserver"
    elif [[ -f "$SCRIPT_DIR/build/bin/fileserver" ]]; then
        server_bin="$SCRIPT_DIR/build/bin/fileserver"
    fi

    if [[ -n "$server_bin" && "$build_server" != "true" ]]; then
        info "Using server binary: $server_bin"
        cd "$project_dir"
        FILES_DIR="$files_dir" PORT=":$port" exec "$server_bin"
    else
        # Build and run from source
        if ! command -v go &> /dev/null; then
            error "Go not found. Install from https://golang.org/dl/"
            exit 1
        fi

        info "Running server from source..."
        cd "$SCRIPT_DIR"
        FILES_DIR="$files_dir" PORT=":$port" exec go run ./server/
    fi
}

# ──────────────────────────────────────────────────────────────────────────────
# BUILD: Build the patcher client
# ──────────────────────────────────────────────────────────────────────────────
cmd_build() {
    local project_dir=""
    local platforms="windows/amd64,linux/amd64"
    local clean=false
    local debug=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --platforms) platforms="$2"; shift 2 ;;
            --platforms=*) platforms="${1#*=}"; shift ;;
            --clean) clean=true; shift ;;
            --debug) debug=true; shift ;;
            -*) error "Unknown option: $1"; exit 1 ;;
            *) project_dir="$1"; shift ;;
        esac
    done

    if [[ -z "$project_dir" ]]; then
        error "Project directory required: ./ppatcher.sh build <project-dir>"
        exit 1
    fi

    if [[ ! -d "$project_dir" ]]; then
        error "Directory '$project_dir' not found"
        exit 1
    fi

    local config_file="$project_dir/config.json"
    if [[ ! -f "$config_file" ]]; then
        error "Config not found: $config_file"
        exit 1
    fi

    header "Building patcher client"
    info "Project: $project_dir"
    info "Config: $config_file"
    info "Platforms: $platforms"

    # Copy config to project root for the build
    cp "$config_file" "$SCRIPT_DIR/config.json"

    cd "$SCRIPT_DIR"

    local build_args="--config config.json --platforms $platforms"
    if [[ "$clean" == "true" ]]; then
        build_args="$build_args --clean"
    fi
    if [[ "$debug" == "true" ]]; then
        build_args="$build_args --debug"
    fi

    ./build-client.sh $build_args

    # Copy built files to project output directory
    local output_dir="$project_dir/dist"
    mkdir -p "$output_dir"

    local output_name="ppatcher"
    if command -v jq &> /dev/null; then
        local config_name
        config_name=$(jq -r '.outputName // "ppatcher"' "$config_file")
        if [[ -n "$config_name" && "$config_name" != "null" ]]; then
            output_name="$config_name"
        fi
    fi

    local copied=0
    for f in "$SCRIPT_DIR/build/bin/${output_name}"*; do
        if [[ -f "$f" ]]; then
            cp "$f" "$output_dir/"
            copied=$((copied + 1))
            success "→ $output_dir/$(basename "$f")"
        fi
    done

    if [[ $copied -eq 0 ]]; then
        warn "No output files found matching '$output_name' in build/bin/"
    else
        echo ""
        success "Built $copied executable(s) in $output_dir/"
        info "Distribute these files to your users."
    fi
}

# ──────────────────────────────────────────────────────────────────────────────
# SETUP: Launch interactive setup wizard
# ──────────────────────────────────────────────────────────────────────────────
cmd_setup() {
    header "Starting Setup Wizard"
    cd "$SCRIPT_DIR"

    local extra_args=()
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --no-browser) extra_args+=("--no-browser"); shift ;;
            --port)       extra_args+=("--port" "$2"); shift 2 ;;
            *)            shift ;;
        esac
    done

    exec go run ./wizard/ "${extra_args[@]}"
}

# ──────────────────────────────────────────────────────────────────────────────
# DEV: Start Wails development environment
# ──────────────────────────────────────────────────────────────────────────────
cmd_dev() {
    header "Starting development environment"

    cd "$SCRIPT_DIR"

    # Auto-detect webkit2gtk tag
    local tags=""
    if [[ "$(uname -s)" == "Linux" ]] && pkg-config --exists webkit2gtk-4.1 2>/dev/null; then
        tags="-tags webkit2_41"
        info "Detected webkit2gtk-4.1, using webkit2_41 build tag"
    fi

    # Clean snap env vars (VS Code snap interference)
    unset GTK_PATH GTK_EXE_PREFIX LOCPATH GTK_IM_MODULE_FILE GSETTINGS_SCHEMA_DIR GIO_MODULE_DIR 2>/dev/null

    info "Running: wails dev $tags"
    echo ""
    exec wails dev $tags
}

# ──────────────────────────────────────────────────────────────────────────────
# STATUS: Show project info
# ──────────────────────────────────────────────────────────────────────────────
cmd_status() {
    local project_dir="$1"

    if [[ -z "$project_dir" ]]; then
        error "Project directory required: ./ppatcher.sh status <project-dir>"
        exit 1
    fi

    if [[ ! -d "$project_dir" ]]; then
        error "Directory '$project_dir' not found"
        exit 1
    fi

    header "Project: $(basename "$project_dir")"

    # Files info
    local files_dir="$project_dir/files"
    if [[ -d "$files_dir" ]]; then
        local file_count total_size
        file_count=$(find "$files_dir" -type f 2>/dev/null | wc -l)
        total_size=$(du -sh "$files_dir" 2>/dev/null | cut -f1)
        info "Files: $file_count files ($total_size)"
    else
        warn "No files/ directory"
    fi

    # Config info
    local config_file="$project_dir/config.json"
    if [[ -f "$config_file" ]]; then
        info "Config: $config_file"
        if command -v jq &> /dev/null; then
            local backend display_name version palette output_name
            backend=$(jq -r '.backend // "not set"' "$config_file")
            display_name=$(jq -r '.displayName // "not set"' "$config_file")
            version=$(jq -r '.version // "not set"' "$config_file")
            palette=$(jq -r '.colorPalette // "not set"' "$config_file")
            output_name=$(jq -r '.outputName // "not set"' "$config_file")
            echo "    Backend:     $backend"
            echo "    Name:        $display_name"
            echo "    Version:     $version"
            echo "    Palette:     $palette"
            echo "    Output name: $output_name"
        fi
    else
        warn "No config.json"
    fi

    # Dist info
    local dist_dir="$project_dir/dist"
    if [[ -d "$dist_dir" ]]; then
        local dist_count
        dist_count=$(find "$dist_dir" -type f 2>/dev/null | wc -l)
        info "Built executables: $dist_count in $dist_dir/"
        for f in "$dist_dir"/*; do
            [[ -f "$f" ]] && echo "    $(basename "$f")  ($(du -h "$f" | cut -f1))"
        done
    else
        info "No builds yet (run: ./ppatcher.sh build $project_dir)"
    fi
}

# ──────────────────────────────────────────────────────────────────────────────
# Main
# ──────────────────────────────────────────────────────────────────────────────
command="${1:-help}"
shift 2>/dev/null || true

case "$command" in
    init)   cmd_init "$@" ;;
    setup)  cmd_setup "$@" ;;
    server) cmd_server "$@" ;;
    build)  cmd_build "$@" ;;
    dev)    cmd_dev "$@" ;;
    status) cmd_status "$@" ;;
    help|--help|-h) usage ;;
    *)
        error "Unknown command: $command"
        usage
        exit 1
        ;;
esac
