# clipboard

A Linux implementation of macOS's `pbcopy` and `pbpaste` commands, written in Go.

## Features

- **pbcopy**: Copy stdin to the system clipboard
- **pbpaste**: Output clipboard contents to stdout
- Supports both X11 (via `xclip` or `xsel`) and Wayland (via `wl-clipboard`)
- Pre-built binaries for both amd64 and arm64 architectures

## Installation

### From Source

```bash
cd clipboard

# Build both commands
go build -o pbcopy ./cmd/pbcopy
go build -o pbpaste ./cmd/pbpaste

# Install to /usr/local/bin
sudo mv pbcopy pbpaste /usr/local/bin/
```

## Prerequisites

You need one of the following clipboard tools installed:

### For X11
```bash
# Debian/Ubuntu
sudo apt install xclip
# or
sudo apt install xsel

# Fedora
sudo dnf install xclip
# or
sudo dnf install xsel

# Arch
sudo pacman -S xclip
# or
sudo pacman -S xsel
```

### For Wayland
```bash
# Debian/Ubuntu
sudo apt install wl-clipboard

# Fedora
sudo dnf install wl-clipboard

# Arch
sudo pacman -S wl-clipboard
```

## Usage

### pbcopy

Copy text to clipboard:

```bash
# Copy text
echo "Hello, World!" | pbcopy

# Copy file contents
cat file.txt | pbcopy

# Using redirection
pbcopy < file.txt
```

### pbpaste

Paste from clipboard:

```bash
# Output clipboard to terminal
pbpaste

# Save clipboard to file
pbpaste > output.txt

# Use in pipeline
pbpaste | grep pattern
```

## How It Works

The tool auto-detects your display server:

1. If `WAYLAND_DISPLAY` is set and `wl-copy`/`wl-paste` are available, uses Wayland
2. Otherwise, tries `xclip` for X11
3. Falls back to `xsel` for X11

## Cross-compilation

```bash
# Build for arm64
GOOS=linux GOARCH=arm64 go build -o pbcopy-arm64 ./cmd/pbcopy
GOOS=linux GOARCH=arm64 go build -o pbpaste-arm64 ./cmd/pbpaste

# Build for amd64
GOOS=linux GOARCH=amd64 go build -o pbcopy-amd64 ./cmd/pbcopy
GOOS=linux GOARCH=amd64 go build -o pbpaste-amd64 ./cmd/pbpaste
```

## License

MIT License
