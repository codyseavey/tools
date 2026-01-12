#!/bin/bash
#
# Build all tools and symlink them to an install directory
#
# Usage: ./build.sh [options]
#
# Options:
#   -o, --os       Target OS (default: current OS)
#   -a, --arch     Target architecture (default: current arch)
#   -d, --dir      Install directory (default: ~/.local/bin)
#   -b, --build    Build output directory (default: ./build)
#   -n, --no-link  Build only, don't create symlinks
#   -c, --clean    Clean build directory before building
#   -h, --help     Show this help message

set -e

# Default values
TARGET_OS="${GOOS:-$(go env GOOS)}"
TARGET_ARCH="${GOARCH:-$(go env GOARCH)}"
INSTALL_DIR="$HOME/.local/bin"
BUILD_DIR="./build"
CREATE_LINKS=true
CLEAN=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory (where the tools repo is)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

usage() {
    cat << EOF
Build all tools and symlink them to an install directory

Usage: $(basename "$0") [options]

Options:
  -o, --os       Target OS (default: $TARGET_OS)
  -a, --arch     Target architecture (default: $TARGET_ARCH)
  -d, --dir      Install directory for symlinks (default: ~/.local/bin)
  -b, --build    Build output directory (default: ./build)
  -n, --no-link  Build only, don't create symlinks
  -c, --clean    Clean build directory before building
  -h, --help     Show this help message

Examples:
  $(basename "$0")                           # Build for current platform, install to ~/.local/bin
  $(basename "$0") -o linux -a arm64         # Cross-compile for linux/arm64
  $(basename "$0") -d /usr/local/bin         # Install to /usr/local/bin
  $(basename "$0") -n                        # Build only, no symlinks

Supported platforms:
  OS:   linux, darwin, windows
  Arch: amd64, arm64, arm, 386
EOF
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -o|--os)
            TARGET_OS="$2"
            shift 2
            ;;
        -a|--arch)
            TARGET_ARCH="$2"
            shift 2
            ;;
        -d|--dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        -b|--build)
            BUILD_DIR="$2"
            shift 2
            ;;
        -n|--no-link)
            CREATE_LINKS=false
            shift
            ;;
        -c|--clean)
            CLEAN=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Expand tilde in INSTALL_DIR
INSTALL_DIR="${INSTALL_DIR/#\~/$HOME}"

# Make BUILD_DIR absolute
if [[ "$BUILD_DIR" != /* ]]; then
    BUILD_DIR="$SCRIPT_DIR/$BUILD_DIR"
fi

# Check for Go
if ! command -v go &> /dev/null; then
    log_error "Go is not installed or not in PATH"
    exit 1
fi

log_info "Building for ${TARGET_OS}/${TARGET_ARCH}"
log_info "Build directory: ${BUILD_DIR}"
if [ "$CREATE_LINKS" = true ]; then
    log_info "Install directory: ${INSTALL_DIR}"
fi

# Clean build directory if requested
if [ "$CLEAN" = true ] && [ -d "$BUILD_DIR" ]; then
    log_info "Cleaning build directory..."
    rm -rf "$BUILD_DIR"
fi

# Create build directory
mkdir -p "$BUILD_DIR"

# Track built binaries for symlinking
declare -a BUILT_BINARIES

# Function to build a Go binary
build_binary() {
    local name="$1"
    local src_path="$2"
    local output_path="$BUILD_DIR/$name"

    # Add .exe extension for Windows
    if [ "$TARGET_OS" = "windows" ]; then
        output_path="${output_path}.exe"
    fi

    log_info "Building $name..."

    GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" go build \
        -ldflags="-s -w" \
        -o "$output_path" \
        "$src_path"

    if [ $? -eq 0 ]; then
        log_success "Built $name -> $output_path"
        BUILT_BINARIES+=("$name:$output_path")
    else
        log_error "Failed to build $name"
        return 1
    fi
}

# Find and build all tools
cd "$SCRIPT_DIR"

log_info "Discovering tools..."

for tool_dir in */; do
    # Skip hidden directories and non-tool directories
    [[ "$tool_dir" == .* ]] && continue

    tool_dir="${tool_dir%/}"  # Remove trailing slash

    # Check if this is a Go module
    if [ ! -f "$tool_dir/go.mod" ]; then
        continue
    fi

    log_info "Found tool: $tool_dir"

    cd "$SCRIPT_DIR/$tool_dir"

    # Check for cmd/ subdirectory pattern (multiple binaries)
    if [ -d "cmd" ]; then
        for cmd_dir in cmd/*/; do
            cmd_dir="${cmd_dir%/}"
            cmd_name="$(basename "$cmd_dir")"

            if [ -f "$cmd_dir/main.go" ]; then
                build_binary "$cmd_name" "./$cmd_dir"
            fi
        done
    fi

    # Check for main.go in root (single binary, named after directory)
    if [ -f "main.go" ]; then
        build_binary "$tool_dir" "."
    fi

    cd "$SCRIPT_DIR"
done

echo ""
log_info "Build complete. Built ${#BUILT_BINARIES[@]} binaries."

# Create symlinks if requested
if [ "$CREATE_LINKS" = true ]; then
    echo ""
    log_info "Creating symlinks in $INSTALL_DIR..."

    # Check if we can create symlinks (same OS or not Windows target)
    if [ "$TARGET_OS" != "$(go env GOOS)" ]; then
        log_warn "Cross-compiling for different OS. Symlinks will point to binaries for $TARGET_OS/$TARGET_ARCH."
        log_warn "These binaries won't run on the current system."
        echo ""
        read -p "Create symlinks anyway? [y/N] " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Skipping symlink creation."
            exit 0
        fi
    fi

    # Create install directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"

    for entry in "${BUILT_BINARIES[@]}"; do
        name="${entry%%:*}"
        path="${entry#*:}"
        link_path="$INSTALL_DIR/$name"

        # Add .exe extension for Windows
        if [ "$TARGET_OS" = "windows" ]; then
            link_path="${link_path}.exe"
        fi

        # Remove existing symlink or file
        if [ -L "$link_path" ] || [ -e "$link_path" ]; then
            rm -f "$link_path"
        fi

        ln -s "$path" "$link_path"
        log_success "Linked $name -> $link_path"
    done

    echo ""
    log_success "Installation complete!"

    # Check if install directory is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        log_warn "$INSTALL_DIR is not in your PATH"
        log_info "Add it to your shell config:"
        echo ""
        echo "    export PATH=\"\$PATH:$INSTALL_DIR\""
        echo ""
    fi
fi

# Print summary
echo ""
echo "=== Summary ==="
echo "Platform:   ${TARGET_OS}/${TARGET_ARCH}"
echo "Build dir:  ${BUILD_DIR}"
echo "Binaries:"
for entry in "${BUILT_BINARIES[@]}"; do
    name="${entry%%:*}"
    echo "  - $name"
done
