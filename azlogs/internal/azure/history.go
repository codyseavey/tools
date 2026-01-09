package azure

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// HistoryEntry represents a query history entry
type HistoryEntry struct {
	Query       string    `json:"query"`
	Workspace   string    `json:"workspace"`
	ExecutedAt  time.Time `json:"executed_at"`
	Duration    string    `json:"duration"`
	RowCount    int       `json:"row_count"`
	WasSuccess  bool      `json:"was_success"`
	ErrorMsg    string    `json:"error_msg,omitempty"`
}

// History manages query history
type History struct {
	Entries  []HistoryEntry `json:"entries"`
	MaxSize  int            `json:"max_size"`
	filePath string
}

// NewHistory creates a new history manager
func NewHistory(maxSize int) *History {
	h := &History{
		Entries: []HistoryEntry{},
		MaxSize: maxSize,
	}
	h.setDefaultPath()
	return h
}

// setDefaultPath sets the default history file path
func (h *History) setDefaultPath() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	configDir := filepath.Join(homeDir, ".config", "azlogs")
	h.filePath = filepath.Join(configDir, "history.json")
}

// Load reads history from disk
func (h *History) Load() error {
	data, err := os.ReadFile(h.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No history file yet
		}
		return err
	}

	return json.Unmarshal(data, h)
}

// Save writes history to disk
func (h *History) Save() error {
	// Ensure directory exists
	dir := filepath.Dir(h.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(h.filePath, data, 0644)
}

// Add adds a new entry to history
func (h *History) Add(entry HistoryEntry) {
	// Add to beginning
	h.Entries = append([]HistoryEntry{entry}, h.Entries...)

	// Trim if exceeds max size
	if len(h.Entries) > h.MaxSize {
		h.Entries = h.Entries[:h.MaxSize]
	}
}

// GetRecent returns the n most recent entries
func (h *History) GetRecent(n int) []HistoryEntry {
	if n > len(h.Entries) {
		n = len(h.Entries)
	}
	return h.Entries[:n]
}

// Search searches history for entries containing the given string
func (h *History) Search(query string) []HistoryEntry {
	var results []HistoryEntry
	for _, entry := range h.Entries {
		if containsIgnoreCase(entry.Query, query) {
			results = append(results, entry)
		}
	}
	return results
}

// Clear clears all history
func (h *History) Clear() {
	h.Entries = []HistoryEntry{}
}

// containsIgnoreCase checks if s contains substr (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	// Simple case-insensitive contains
	sLower := toLower(s)
	substrLower := toLower(substr)
	return contains(sLower, substrLower)
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Config holds application configuration
type Config struct {
	DefaultWorkspace string `json:"default_workspace"`
	DefaultAuthMethod AuthMethod `json:"default_auth_method"`
	QueryTimeout     int    `json:"query_timeout_seconds"`
	MaxHistorySize   int    `json:"max_history_size"`
	SavedWorkspaces  []SavedWorkspace `json:"saved_workspaces"`
}

// SavedWorkspace represents a saved workspace
type SavedWorkspace struct {
	Name        string `json:"name"`
	WorkspaceID string `json:"workspace_id"`
	Description string `json:"description,omitempty"`
}

// NewConfig creates a new config with defaults
func NewConfig() *Config {
	return &Config{
		DefaultAuthMethod: AuthDefault,
		QueryTimeout:     300,
		MaxHistorySize:   1000,
		SavedWorkspaces:  []SavedWorkspace{},
	}
}

// Load reads config from disk
func (c *Config) Load() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	configPath := filepath.Join(homeDir, ".config", "azlogs", "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, c)
}

// Save writes config to disk
func (c *Config) Save() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	configDir := filepath.Join(homeDir, ".config", "azlogs")
	configPath := filepath.Join(configDir, "config.json")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// AddWorkspace adds a workspace to saved workspaces
func (c *Config) AddWorkspace(ws SavedWorkspace) {
	// Check if already exists
	for i, existing := range c.SavedWorkspaces {
		if existing.WorkspaceID == ws.WorkspaceID {
			c.SavedWorkspaces[i] = ws // Update existing
			return
		}
	}
	c.SavedWorkspaces = append(c.SavedWorkspaces, ws)
}

// RemoveWorkspace removes a workspace from saved workspaces
func (c *Config) RemoveWorkspace(workspaceID string) {
	for i, ws := range c.SavedWorkspaces {
		if ws.WorkspaceID == workspaceID {
			c.SavedWorkspaces = append(c.SavedWorkspaces[:i], c.SavedWorkspaces[i+1:]...)
			return
		}
	}
}
