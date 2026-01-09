# Tools

A collection of useful CLI tools for Linux, written in Go.

## Tools

| Tool | Description |
|------|-------------|
| [azlogs](./azlogs) | Interactive terminal application for querying Azure Log Analytics workspaces |
| [clipboard](./clipboard) | Linux implementation of macOS's `pbcopy` and `pbpaste` commands |

## Structure

This repository is organized as a monorepo with each tool in its own subdirectory:

```
tools/
├── azlogs/           # Azure Log Analytics CLI
│   ├── main.go
│   ├── go.mod
│   ├── internal/
│   └── README.md
├── clipboard/        # pbcopy/pbpaste for Linux
│   ├── cmd/
│   ├── go.mod
│   ├── internal/
│   └── README.md
└── README.md
```

Each tool has its own `go.mod` and can be built independently.

## Quick Start

### azlogs

```bash
cd azlogs
go build -o azlogs .
./azlogs -w "your-workspace-id"
```

### clipboard

```bash
cd clipboard
go build -o pbcopy ./cmd/pbcopy
go build -o pbpaste ./cmd/pbpaste
echo "Hello" | ./pbcopy
./pbpaste
```

## Building All Tools

```bash
# Build azlogs
cd azlogs && go build -o azlogs . && cd ..

# Build clipboard tools
cd clipboard && go build -o pbcopy ./cmd/pbcopy && go build -o pbpaste ./cmd/pbpaste && cd ..

# Install to /usr/local/bin
sudo cp azlogs/azlogs clipboard/pbcopy clipboard/pbpaste /usr/local/bin/
```

## License

MIT License
