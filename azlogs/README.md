# azlogs

An interactive terminal application for querying Azure Log Analytics workspaces.

## Features

- Interactive KQL query editor with syntax highlighting
- Results displayed in a navigable table
- Query history with persistence
- Multiple authentication methods (Azure CLI, Browser, Managed Identity)
- Workspace management and switching
- Non-interactive mode for scripting

## Installation

```bash
# From source
cd azlogs
go build -o azlogs .

# Install to PATH
sudo mv azlogs /usr/local/bin/
```

## Prerequisites

- Azure subscription with Log Analytics workspace
- One of the following for authentication:
  - Azure CLI installed and logged in (`az login`)
  - Web browser for interactive login
  - Managed Identity (when running in Azure)

## Usage

### Interactive Mode

```bash
# Start with workspace ID
azlogs -w "your-workspace-id"

# Or set via environment variable
export AZURE_LOG_ANALYTICS_WORKSPACE_ID="your-workspace-id"
azlogs

# Use specific authentication method
azlogs -w "your-workspace-id" --auth cli      # Azure CLI
azlogs -w "your-workspace-id" --auth browser  # Browser login
```

### Non-Interactive Mode

```bash
# Execute a query and get tab-separated output
azlogs -w "your-workspace-id" -q "AzureActivity | take 10"

# Pipe to other tools
azlogs -w "your-workspace-id" -q "SecurityEvent | take 100" | cut -f1,2,3
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `F5` / `Ctrl+Enter` | Execute query |
| `Tab` | Switch between editor and results |
| `F1` | Show help |
| `F2` | Show query history |
| `F3` | Change workspace |
| `Ctrl+Q` | Quit |
| `j/k` or `Up/Down` | Navigate rows (in results) |
| `h/l` or `Left/Right` | Scroll columns |
| `PgUp/PgDown` | Page navigation |
| `g/G` or `Home/End` | Jump to start/end |

## KQL Quick Reference

```kql
# Fetch rows
TableName | take 10

# Filter
TableName | where TimeGenerated > ago(1h)
TableName | where Column == "value"

# Select columns
TableName | project Column1, Column2

# Aggregate
TableName | summarize count() by Category

# Sort
TableName | order by TimeGenerated desc

# Combine operations
AzureActivity
| where TimeGenerated > ago(24h)
| where Level == "Error"
| project TimeGenerated, OperationName, Caller
| order by TimeGenerated desc
| take 50
```

## Configuration

azlogs stores configuration and history in `~/.config/azlogs/`:

- `config.json` - Application settings and saved workspaces
- `history.json` - Query history

## License

MIT License
