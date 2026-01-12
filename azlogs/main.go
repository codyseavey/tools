package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/codyseavey/tools/azlogs/internal/azure"
	"github.com/codyseavey/tools/azlogs/internal/ui"
)

const version = "1.0.0"

func main() {
	// Command line flags
	workspaceID := flag.String("workspace", "", "Azure Log Analytics Workspace ID")
	workspaceShort := flag.String("w", "", "Azure Log Analytics Workspace ID (shorthand)")
	authMethod := flag.String("auth", "default", "Authentication method: default, cli, browser, managed-identity")
	query := flag.String("query", "", "Execute a query and exit (non-interactive mode)")
	queryShort := flag.String("q", "", "Execute a query and exit (shorthand)")
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")

	flag.Parse()

	if *showVersion {
		fmt.Printf("azlogs version %s\n", version)
		os.Exit(0)
	}

	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// Resolve workspace ID
	ws := *workspaceID
	if ws == "" {
		ws = *workspaceShort
	}
	if ws == "" {
		ws = os.Getenv("AZURE_LOG_ANALYTICS_WORKSPACE_ID")
	}

	// Resolve query
	q := *query
	if q == "" {
		q = *queryShort
	}

	// Resolve auth method
	auth := parseAuthMethod(*authMethod)

	// Non-interactive mode
	if q != "" {
		if ws == "" {
			fmt.Fprintln(os.Stderr, "Error: workspace ID is required. Use -w flag or set AZURE_LOG_ANALYTICS_WORKSPACE_ID")
			os.Exit(1)
		}
		runNonInteractive(ws, q, auth)
		return
	}

	// Interactive mode
	runInteractive(ws, auth)
}

func parseAuthMethod(method string) azure.AuthMethod {
	switch method {
	case "cli":
		return azure.AuthCLI
	case "browser":
		return azure.AuthBrowser
	case "managed-identity", "msi":
		return azure.AuthManagedIdentity
	default:
		return azure.AuthDefault
	}
}

func runInteractive(workspaceID string, auth azure.AuthMethod) {
	// Print banner
	fmt.Print(ui.LogoStyled())
	fmt.Println()

	// Create the model - Init() will auto-connect if workspace is provided
	m := ui.NewModel(workspaceID, auth)

	// Create and run the program
	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

func runNonInteractive(workspaceID, query string, authMethod azure.AuthMethod) {
	// Create authenticator
	auth, err := azure.NewAuthenticator(authMethod)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Authentication failed: %v\n", err)
		os.Exit(1)
	}

	// Create client
	client, err := azure.NewLogAnalyticsClient(auth.GetCredential(), workspaceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
		os.Exit(1)
	}

	// Execute query
	fmt.Fprintf(os.Stderr, "Executing query...\n")
	result, err := client.Query(context.Background(), query, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	}

	// Print results as tab-separated values
	if len(result.Tables) > 0 {
		table := result.Tables[0]

		// Print header
		for i, col := range table.Columns {
			if i > 0 {
				fmt.Print("\t")
			}
			fmt.Print(col.Name)
		}
		fmt.Println()

		// Print rows
		for _, row := range table.Rows {
			for i, cell := range row {
				if i > 0 {
					fmt.Print("\t")
				}
				fmt.Print(formatValue(cell))
			}
			fmt.Println()
		}
	}

	fmt.Fprintf(os.Stderr, "\n%d rows returned in %s\n", result.RowCount, result.Duration)
}

func formatValue(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func printHelp() {
	help := `Azure Log Analytics CLI (azlogs)

A terminal-based application for querying Azure Log Analytics workspaces.

USAGE:
    azlogs [OPTIONS]

OPTIONS:
    -w, --workspace <ID>    Azure Log Analytics Workspace ID
                            Can also be set via AZURE_LOG_ANALYTICS_WORKSPACE_ID

    -q, --query <KQL>       Execute a KQL query in non-interactive mode
                            Results are printed as tab-separated values

    --auth <METHOD>         Authentication method:
                            - default   : Auto-detect (tries multiple methods)
                            - cli       : Use Azure CLI credentials
                            - browser   : Interactive browser login
                            - managed-identity : Azure Managed Identity

    --version               Show version information
    --help                  Show this help message

INTERACTIVE MODE:
    Run without -q to start the interactive TUI where you can:
    - Write and execute KQL queries
    - Browse results in a table view
    - View query history
    - Save and switch between workspaces

EXAMPLES:
    # Start interactive mode
    azlogs -w "your-workspace-id"

    # Execute a query and exit
    azlogs -w "your-workspace-id" -q "AzureActivity | take 10"

    # Use Azure CLI authentication
    azlogs -w "your-workspace-id" --auth cli

    # Use environment variable for workspace
    export AZURE_LOG_ANALYTICS_WORKSPACE_ID="your-workspace-id"
    azlogs

KEYBOARD SHORTCUTS (Interactive Mode):
    F5, Ctrl+Enter    Execute query
    Tab               Switch between editor and results
    F1                Show help
    F2                Show query history
    F3                Change workspace
    Ctrl+Q            Quit

For more information, visit: https://github.com/codyseavey/tools/azlogs
`
	fmt.Print(help)
}
