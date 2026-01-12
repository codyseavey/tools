package azure

import (
	"context"
	"os"
	"testing"
	"time"
)

func getTestWorkspaceID(t *testing.T) string {
	ws := os.Getenv("AZURE_LOG_ANALYTICS_WORKSPACE_ID")
	if ws == "" {
		t.Skip("AZURE_LOG_ANALYTICS_WORKSPACE_ID not set, skipping integration test")
	}
	return ws
}

func TestNewLogAnalyticsClient(t *testing.T) {
	auth, err := NewAuthenticator(AuthCLI)
	if err != nil {
		t.Fatalf("Failed to create authenticator: %v", err)
	}

	client, err := NewLogAnalyticsClient(auth.GetCredential(), "test-workspace-id")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.GetWorkspace() != "test-workspace-id" {
		t.Errorf("Expected workspace 'test-workspace-id', got '%s'", client.GetWorkspace())
	}
}

func TestLogAnalyticsClient_SetWorkspace(t *testing.T) {
	auth, err := NewAuthenticator(AuthCLI)
	if err != nil {
		t.Fatalf("Failed to create authenticator: %v", err)
	}

	client, err := NewLogAnalyticsClient(auth.GetCredential(), "initial-workspace")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	client.SetWorkspace("new-workspace")
	if client.GetWorkspace() != "new-workspace" {
		t.Errorf("Expected workspace 'new-workspace', got '%s'", client.GetWorkspace())
	}
}

func TestLogAnalyticsClient_Query_Integration(t *testing.T) {
	workspaceID := getTestWorkspaceID(t)

	auth, err := NewAuthenticator(AuthCLI)
	if err != nil {
		t.Fatalf("Failed to create authenticator: %v", err)
	}

	// First validate the token works
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Log("Validating Azure CLI credentials...")
	if err := auth.Validate(ctx); err != nil {
		t.Fatalf("Token validation failed: %v\nMake sure you are logged in with 'az login'", err)
	}
	t.Log("Credentials validated successfully")

	client, err := NewLogAnalyticsClient(auth.GetCredential(), workspaceID)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Log("Executing test query...")
	result, err := client.Query(ctx, "AzureActivity | take 1", nil)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("Query completed: %d rows returned in %s", result.RowCount, result.Duration)
}

func TestLogAnalyticsClient_Query_WithTimespan(t *testing.T) {
	workspaceID := getTestWorkspaceID(t)

	auth, err := NewAuthenticator(AuthCLI)
	if err != nil {
		t.Fatalf("Failed to create authenticator: %v", err)
	}

	client, err := NewLogAnalyticsClient(auth.GetCredential(), workspaceID)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	timespan := &TimeSpan{
		Start: time.Now().Add(-24 * time.Hour),
		End:   time.Now(),
	}

	result, err := client.Query(ctx, "AzureActivity | take 1", timespan)
	if err != nil {
		t.Fatalf("Query with timespan failed: %v", err)
	}

	t.Logf("Query completed: %d rows returned in %s", result.RowCount, result.Duration)
}

func TestLogAnalyticsClient_QueryWithTimeout(t *testing.T) {
	workspaceID := getTestWorkspaceID(t)

	auth, err := NewAuthenticator(AuthCLI)
	if err != nil {
		t.Fatalf("Failed to create authenticator: %v", err)
	}

	client, err := NewLogAnalyticsClient(auth.GetCredential(), workspaceID)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	result, err := client.QueryWithTimeout(ctx, "AzureActivity | take 1", nil, 30*time.Second)
	if err != nil {
		t.Fatalf("Query with timeout failed: %v", err)
	}

	t.Logf("Query completed: %d rows returned in %s", result.RowCount, result.Duration)
}
