package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
)

// LogAnalyticsClient handles queries to Azure Log Analytics
type LogAnalyticsClient struct {
	client      *azquery.LogsClient
	workspaceID string
}

// QueryResult represents the result of a Log Analytics query
type QueryResult struct {
	Tables      []Table
	Statistics  string
	Duration    time.Duration
	RowCount    int
	QueryStatus string
}

// Table represents a result table from a query
type Table struct {
	Name    string
	Columns []Column
	Rows    [][]interface{}
}

// Column represents a column in a result table
type Column struct {
	Name string
	Type string
}

// TimeSpan represents a time range for queries
type TimeSpan struct {
	Start time.Time
	End   time.Time
}

// NewLogAnalyticsClient creates a new Log Analytics client
func NewLogAnalyticsClient(cred azcore.TokenCredential, workspaceID string) (*LogAnalyticsClient, error) {
	client, err := azquery.NewLogsClient(cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create logs client: %w", err)
	}

	return &LogAnalyticsClient{
		client:      client,
		workspaceID: workspaceID,
	}, nil
}

// SetWorkspace changes the workspace ID
func (c *LogAnalyticsClient) SetWorkspace(workspaceID string) {
	c.workspaceID = workspaceID
}

// GetWorkspace returns the current workspace ID
func (c *LogAnalyticsClient) GetWorkspace() string {
	return c.workspaceID
}

// Query executes a KQL query against the workspace
func (c *LogAnalyticsClient) Query(ctx context.Context, query string, timespan *TimeSpan) (*QueryResult, error) {
	start := time.Now()

	body := azquery.Body{
		Query: &query,
	}

	// Set timespan if provided
	if timespan != nil {
		ts := azquery.NewTimeInterval(timespan.Start, timespan.End)
		body.Timespan = &ts
	}

	resp, err := c.client.QueryWorkspace(ctx, c.workspaceID, body, nil)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	duration := time.Since(start)
	result := &QueryResult{
		Duration:    duration,
		QueryStatus: "Success",
	}

	// Handle partial errors
	if resp.Error != nil && resp.Error.Code != "" {
		result.QueryStatus = fmt.Sprintf("Partial: %s", resp.Error.Code)
	}

	// Process tables
	for _, t := range resp.Tables {
		table := Table{
			Name: *t.Name,
		}

		// Process columns
		for _, col := range t.Columns {
			table.Columns = append(table.Columns, Column{
				Name: *col.Name,
				Type: string(*col.Type),
			})
		}

		// Process rows
		for _, row := range t.Rows {
			table.Rows = append(table.Rows, row)
			result.RowCount++
		}

		result.Tables = append(result.Tables, table)
	}

	return result, nil
}

// QueryWithTimeout executes a query with a specific timeout
func (c *LogAnalyticsClient) QueryWithTimeout(ctx context.Context, query string, timespan *TimeSpan, timeout time.Duration) (*QueryResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.Query(ctx, query, timespan)
}

// GetAvailableTables returns a list of tables in the workspace
func (c *LogAnalyticsClient) GetAvailableTables(ctx context.Context) ([]string, error) {
	query := `search * | summarize count() by $table | project $table | order by $table asc`
	result, err := c.Query(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	var tables []string
	if len(result.Tables) > 0 && len(result.Tables[0].Rows) > 0 {
		for _, row := range result.Tables[0].Rows {
			if len(row) > 0 {
				if tableName, ok := row[0].(string); ok {
					tables = append(tables, tableName)
				}
			}
		}
	}

	return tables, nil
}

// GetTableSchema returns the schema for a specific table
func (c *LogAnalyticsClient) GetTableSchema(ctx context.Context, tableName string) ([]Column, error) {
	query := fmt.Sprintf("%s | getschema", tableName)
	result, err := c.Query(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	var columns []Column
	if len(result.Tables) > 0 {
		for _, row := range result.Tables[0].Rows {
			if len(row) >= 2 {
				col := Column{}
				if name, ok := row[0].(string); ok {
					col.Name = name
				}
				if colType, ok := row[1].(string); ok {
					col.Type = colType
				}
				columns = append(columns, col)
			}
		}
	}

	return columns, nil
}
